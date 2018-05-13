package elasticipd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	envElasticIP  = "ELASTIC_IP"
	errAssociated = errors.New("address already associated")
)

func main() {
	ip := os.Getenv(envElasticIP)
	if ip == "" {
		log.Fatalf("%s is not set", envElasticIP)
	}

	quit := make(chan os.Signal, 2)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	t := time.NewTicker(60 * time.Second)
	c := &ipChecker{
		ip: ip,
	}
	shutdown := false

	for {
		select {
		case <-quit:
			t.Stop()
			shutdown = true
			log.Printf("gracefully shutting down, disassociating address: %s", ip)
			checkElasticIPAssignment(c, shutdown)
			os.Exit(1)
		case <-t.C:
			checkElasticIPAssignment(c, shutdown)
		}
	}
}

func checkElasticIPAssignment(c *ipChecker, shutdown bool) {
	sess, err := session.NewSession()
	if err != nil {
		log.Printf("error creating AWS session: %v", err)
		return
	}

	c.ec2 = ec2.New(sess)
	c.ec2metadata = ec2metadata.New(sess)

	ad, err := c.describeAddr(c.ip)
	if err != nil {
		if err == errAssociated {
			log.Printf("%v: %s, skipping", err, c.ip)
			return
		}
		log.Println(err)
		return
	}

	if err := c.disassociateAddr(c.ip, ad.associationID, ad.instanceID); err != nil {
		log.Println(err)
		return
	}

	// don't associate the Elastic IP if shutting down.
	if !shutdown {
		if err := c.associateAddr(c.ip, ad.allocationID, ad.instanceID); err != nil {
			log.Println(err)
			return
		}
	}
}

type addresses interface {
	DescribeAddresses(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error)
	AssociateAddress(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error)
	DisassociateAddress(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error)
}

type metadata interface {
	GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error)
}

type asscDetails struct {
	instanceID    string
	associationID string
	allocationID  string
}

type ipChecker struct {
	ec2         addresses
	ec2metadata metadata
	ip          string
}

// describeAddr gets allocation and association information about the
// given Elastic IP address and the current EC2 instance.
func (c *ipChecker) describeAddr(ip string) (ad asscDetails, err error) {
	descRes, err := c.ec2.DescribeAddresses(&ec2.DescribeAddressesInput{
		PublicIps: aws.StringSlice([]string{ip}),
	})
	if err != nil {
		return ad, fmt.Errorf("error describing address: %v", err)
	}
	if len(descRes.Addresses) == 0 {
		return ad, fmt.Errorf("failed to find address: %s", ip)
	}
	addr := descRes.Addresses[0]
	if addr.InstanceId == nil {
		return ad, errors.New("InstanceId is nil")
	}
	if addr.AssociationId == nil {
		return ad, errors.New("AssociationId is nil")
	}
	if addr.AllocationId == nil {
		return ad, errors.New("AllocationId is nil")
	}

	ident, err := c.ec2metadata.GetInstanceIdentityDocument()
	if err != nil {
		return ad, fmt.Errorf("error getting instance identity document: %v", err)
	}
	log.Printf("found instance ID: %s", ident.InstanceID)

	if *addr.InstanceId == ident.InstanceID {
		return ad, errAssociated
	}

	ad = asscDetails{
		instanceID:    ident.InstanceID,
		associationID: *addr.AssociationId,
		allocationID:  *addr.AllocationId,
	}
	return ad, nil
}

// associateAddr will attempt to associate an Elastic IP address to an EC2 instance.
func (c *ipChecker) associateAddr(ip, allocationID, instanceID string) error {
	_, err := c.ec2.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: aws.String(allocationID),
		InstanceId:   aws.String(instanceID),
	})
	if err != nil {
		return fmt.Errorf("failed to associate address %s: %v", ip, err)
	}
	log.Printf("successfully associated address: %s to instance: %s", ip, instanceID)
	return nil
}

// disassociateAddr will attempt to disassociate an Elastic IP address to an EC2 instance.
func (c *ipChecker) disassociateAddr(ip, associationID, instanceID string) error {
	if associationID == "" {
		return nil
	}
	_, err := c.ec2.DisassociateAddress(&ec2.DisassociateAddressInput{
		AssociationId: aws.String(associationID),
	})
	if err != nil {
		return fmt.Errorf("failed to disassociate address %s: %v", ip, err)
	}
	log.Printf("successfully disassociated address: %s from instance: %s", ip, instanceID)
	return nil
}
