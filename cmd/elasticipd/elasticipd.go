package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Prometheus gauge for storing number of failed elastic ip operations
	criticalErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: service,
			Name:      "critical_error_count",
			Help:      "Counter representing the number of errors associating/disassociating the Elastic IP",
		},
		[]string{"operation"},
	)
)

// describer represents the required EC2 functions for describing addresses and instances
type describer interface {
	DescribeAddresses(context.Context, *ec2.DescribeAddressesInput, ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error)
	DescribeInstances(context.Context, *ec2.DescribeInstancesInput, ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
}

// associater represents the required EC2 functions needed to interact with addresses
type associater interface {
	AssociateAddress(context.Context, *ec2.AssociateAddressInput, ...func(*ec2.Options)) (*ec2.AssociateAddressOutput, error)
	DisassociateAddress(context.Context, *ec2.DisassociateAddressInput, ...func(*ec2.Options)) (*ec2.DisassociateAddressOutput, error)
}

type associaterDescriber interface {
	associater
	describer
}

// metadata represents the required EC2Metadata functions
type metadata interface {
	GetInstanceIdentityDocument(context.Context, *imds.GetInstanceIdentityDocumentInput, ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error)
}

// association represents data on an EC2 address association
type association struct {
	// The ID representing the association of the address with an instance in a
	// VPC.
	id string

	// The ID of the instance that the address is associated with (if any).
	instanceID string

	// The ID representing the allocation of the address for use with EC2-VPC.
	allocationID string
}

// instance represents data on a specific EC2 instance
type instance struct {
	id string

	// The IDs of the network interfaces
	networkInterfaceIDs []string
}

type svc struct {
	ec2  associaterDescriber
	imds metadata
}

// create a new service and attempt to load the default aws config
func newSvc(region string) (svc, error) {
	var s svc

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return s, fmt.Errorf("error loading aws config: %w", err)
	}

	s.ec2 = ec2.NewFromConfig(cfg)
	s.imds = imds.NewFromConfig(cfg)

	return s, nil
}

// getAssociation gets allocation and association information about the
// given Elastic IP address and the current EC2 instance.
func (s svc) getAssociation(ip string) (association, error) {
	var assoc association

	// describe the given ip address
	res, err := s.ec2.DescribeAddresses(context.Background(), &ec2.DescribeAddressesInput{
		PublicIps: []string{ip},
	})
	if err != nil {
		return assoc, fmt.Errorf("error describing address: %w", err)
	}

	if len(res.Addresses) == 0 {
		return assoc, fmt.Errorf("failed to find address info")
	}

	// check for valid Allocation ID
	addr := res.Addresses[0]
	if addr.AllocationId == nil {
		return assoc, fmt.Errorf("allocation id is nil")
	}

	assoc.id = aws.ToString(addr.AssociationId)
	assoc.instanceID = aws.ToString(addr.InstanceId)
	assoc.allocationID = aws.ToString(addr.AllocationId)

	return assoc, nil
}

// getInstanceDetails queries the EC2 metadata api to get the current instance id
// and then attempts to get the attached network interfaces
func (s svc) getInstanceDetails() (instance, error) {
	var ins instance

	// get identity document of current EC2 instance
	ident, err := s.imds.GetInstanceIdentityDocument(context.Background(), nil)
	if err != nil {
		return ins, fmt.Errorf("error getting instance identity document: %w", err)
	}

	ins.id = ident.InstanceID

	// get specific instance data
	res, err := s.ec2.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{ins.id},
	})
	if err != nil {
		return ins, fmt.Errorf("error describing instance: %w", err)
	}

	// check we received reservations with instances
	if len(res.Reservations) != 1 {
		return ins, fmt.Errorf("invalid instance description: no reservations")
	}
	if len(res.Reservations[0].Instances) != 1 {
		return ins, fmt.Errorf("instance not found in reservation")
	}

	for _, i := range res.Reservations[0].Instances[0].NetworkInterfaces {
		ins.networkInterfaceIDs = append(ins.networkInterfaceIDs, aws.ToString(i.NetworkInterfaceId))
	}

	return ins, nil
}

// associateAddr will attempt to associate an Elastic IP address to an EC2 instance.
func (s svc) associateAddr(assoc association, ins instance, reassoc bool) error {
	input := &ec2.AssociateAddressInput{
		AllocationId:       aws.String(assoc.allocationID),
		AllowReassociation: aws.Bool(reassoc),
	}

	// Specify the instance id if the instance has only one network interface, otherwise
	// specify a network interface id
	if len(ins.networkInterfaceIDs) > 1 {
		input.NetworkInterfaceId = aws.String(ins.networkInterfaceIDs[0])
	} else {
		input.InstanceId = aws.String(ins.id)
	}

	if _, err := s.ec2.AssociateAddress(context.Background(), input); err != nil {
		criticalErrors.WithLabelValues("association").Inc()
		return err
	}
	return nil
}

// disassociateAddr will attempt to disassociate an Elastic IP address to an EC2 instance.
func (s svc) disassociateAddr(assoc association) error {
	if assoc.id == "" {
		return nil
	}

	if _, err := s.ec2.DisassociateAddress(context.Background(), &ec2.DisassociateAddressInput{
		AssociationId: aws.String(assoc.id),
	}); err != nil {
		criticalErrors.WithLabelValues("disassociation").Inc()
		return err
	}
	return nil
}
