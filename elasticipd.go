package main

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// assignElasticIP configures an AWS session and attempts to disassociate the
// given IP from any current associations, and associate it to the current
// EC2 instance.
func assignElasticIP(ip string, shutdown bool) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("error creating aws session: %w", err)
	}

	svc := awsSvc{
		ec2:         ec2.New(sess),
		ec2metadata: ec2metadata.New(sess),
	}

	assc, err := svc.describeAddr(ip)
	if err != nil {
		return fmt.Errorf("error describing address: %w", err)
	}

	// is the ip address is already associated to the current ec2 instance, skip
	if assc.skippable {
		return nil
	}

	// if there is no current association, skip
	if assc.id == "" {
		if err := svc.disassociateAddr(assc.id); err != nil {
			return fmt.Errorf("error disassociating address: %w", err)
		}
		log.Printf("successfully disassociated address: %s from instance: %s", ip, assc.instance)
	}

	// don't associate the elastic ip if shutting down
	if !shutdown {
		if err := svc.associateAddr(assc.allocation, assc.instance); err != nil {
			return fmt.Errorf("error associating address: %w", err)
		}
		log.Printf("successfully associated address: %s to instance: %s", ip, assc.instance)
	}

	return nil
}

type awsSvc struct {
	ec2         addresses
	ec2metadata metadata
}

// addresses represents the required EC2 functions
type addresses interface {
	DescribeAddresses(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error)
	AssociateAddress(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error)
	DisassociateAddress(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error)
}

// metadata represents the required EC2Metadata functions
type metadata interface {
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

// association represents data on an EC2 address association
type association struct {
	// The ID representing the association of the address with an instance in a
	// VPC.
	id string

	// The ID of the instance that the address is associated with (if any).
	instance string

	// The ID representing the allocation of the address for use with EC2-VPC.
	allocation string

	// If the association Instance ID matches the given IP Instance ID, this will
	// be true, else false
	skippable bool
}

// describeAddr gets allocation and association information about the
// given Elastic IP address and the current EC2 instance.
func (svc awsSvc) describeAddr(ip string) (*association, error) {
	// describe the given ip address
	res, err := svc.ec2.DescribeAddresses(&ec2.DescribeAddressesInput{
		PublicIps: aws.StringSlice([]string{ip}),
	})
	if err != nil {
		return nil, fmt.Errorf("error describing address: %v", err)
	}

	if len(res.Addresses) == 0 {
		return nil, fmt.Errorf("failed to find address info")
	}

	// check for valid association details
	addr := res.Addresses[0]
	if addr.InstanceId == nil {
		return nil, fmt.Errorf("InstanceId is nil")
	}
	if addr.AssociationId == nil {
		return nil, fmt.Errorf("AssociationId is nil")
	}
	if addr.AllocationId == nil {
		return nil, fmt.Errorf("AllocationId is nil")
	}

	// get identity document of current ec2 instance
	ident, err := svc.ec2metadata.GetInstanceIdentityDocument()
	if err != nil {
		return nil, fmt.Errorf("error getting instance identity document: %v", err)
	}

	ad := &association{
		id:         aws.StringValue(addr.AssociationId),
		instance:   ident.InstanceID,
		allocation: aws.StringValue(addr.AllocationId),
		skippable:  aws.StringValue(addr.InstanceId) == ident.InstanceID,
	}
	return ad, nil
}

// associateAddr will attempt to associate an Elastic IP address to an EC2 instance.
func (svc awsSvc) associateAddr(allocation, instance string) error {
	_, err := svc.ec2.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: aws.String(allocation),
		InstanceId:   aws.String(instance),
	})
	if err != nil {
		return fmt.Errorf("failed to associate address: %w", err)
	}
	return nil
}

// disassociateAddr will attempt to disassociate an Elastic IP address to an EC2 instance.
func (svc awsSvc) disassociateAddr(associationID string) error {
	_, err := svc.ec2.DisassociateAddress(&ec2.DisassociateAddressInput{
		AssociationId: aws.String(associationID),
	})
	if err != nil {
		return fmt.Errorf("failed to disassociate address: %w", err)
	}
	return nil
}
