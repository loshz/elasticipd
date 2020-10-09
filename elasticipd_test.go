package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	ip string = "127.0.0.1"
)

type mockEC2 struct {
	DescribeFunc     func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error)
	AssociateFunc    func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error)
	DisassociateFunc func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error)
}

func (m mockEC2) DescribeAddresses(input *ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
	return m.DescribeFunc(input)
}

func (m mockEC2) AssociateAddress(input *ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error) {
	return m.AssociateFunc(input)
}

func (m mockEC2) DisassociateAddress(input *ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
	return m.DisassociateFunc(input)
}

type mockMetadata struct {
	GetFunc func() (ec2metadata.EC2InstanceIdentityDocument, error)
}

func (m mockMetadata) GetInstanceIdentityDocument() (ec2metadata.EC2InstanceIdentityDocument, error) {
	return m.GetFunc()
}

type fixture struct {
	ec2      mockEC2
	metadata mockMetadata
	err      error
}

func TestDescribeAddress(t *testing.T) {
	t.Parallel()

	testTable := make(map[string]fixture)
	testTable["TestDescribeAddressesError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{}, fmt.Errorf("describe address error")
			},
		},
		err: fmt.Errorf("error describing address"),
	}
	testTable["TestAddressesLengthError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{}, nil
			},
		},
		err: fmt.Errorf("failed to find address info"),
	}
	testTable["TestAllocationIdNilError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{
					Addresses: []*ec2.Address{
						&ec2.Address{
							InstanceId:    aws.String("1"),
							AssociationId: aws.String("2"),
							AllocationId:  nil,
						},
					},
				}, nil
			},
		},
		err: errors.New("Allocation ID is nil"),
	}
	testTable["TestGetIdentityDocumentError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{
					Addresses: []*ec2.Address{
						&ec2.Address{
							InstanceId:    aws.String("1"),
							AssociationId: aws.String("2"),
							AllocationId:  aws.String("3"),
						},
					},
				}, nil
			},
		},
		metadata: mockMetadata{
			GetFunc: func() (ec2metadata.EC2InstanceIdentityDocument, error) {
				return ec2metadata.EC2InstanceIdentityDocument{}, fmt.Errorf("describe address error")
			},
		},
		err: fmt.Errorf("error getting instance identity document"),
	}
	testTable["TestSuccess"] = fixture{
		metadata: mockMetadata{
			GetFunc: func() (ec2metadata.EC2InstanceIdentityDocument, error) {
				return ec2metadata.EC2InstanceIdentityDocument{
					InstanceID: "1",
				}, nil
			},
		},
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{
					Addresses: []*ec2.Address{
						&ec2.Address{
							InstanceId:    aws.String("2"),
							AssociationId: aws.String("2"),
							AllocationId:  aws.String("3"),
						},
					},
				}, nil
			},
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			svc := awsSvc{
				ec2:         test.ec2,
				ec2metadata: test.metadata,
			}

			_, err := svc.describeAddr(ip)
			if test.err != nil && !strings.Contains(err.Error(), test.err.Error()) {
				t.Errorf("expected error: '%v', got: '%v'", test.err, err)
			}
			if test.err == nil && err != nil {
				t.Errorf("expected error: nil, got: '%v'", err)
			}
		})
	}
}

func TestAssociateAddress(t *testing.T) {
	t.Parallel()

	testTable := make(map[string]fixture)
	testTable["TestAssociateError"] = fixture{
		ec2: mockEC2{
			AssociateFunc: func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error) {
				return &ec2.AssociateAddressOutput{}, fmt.Errorf("error associating address")
			},
		},
		err: fmt.Errorf("failed to associate Elastic IP"),
	}
	testTable["TestSuccess"] = fixture{
		ec2: mockEC2{
			AssociateFunc: func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error) {
				return &ec2.AssociateAddressOutput{}, nil
			},
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			svc := awsSvc{
				ec2: test.ec2,
			}

			err := svc.associateAddr("association", "instance")
			if test.err != nil && !strings.Contains(err.Error(), test.err.Error()) {
				t.Errorf("expected error: '%v', got: '%v'", test.err, err)
			}
			if test.err == nil && err != nil {
				t.Errorf("expected error: nil, got: '%v'", err)
			}
		})
	}
}

func TestDisassociateAddress(t *testing.T) {
	t.Parallel()

	testTable := make(map[string]fixture)
	testTable["TestDisassociateError"] = fixture{
		ec2: mockEC2{
			DisassociateFunc: func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
				return &ec2.DisassociateAddressOutput{}, fmt.Errorf("error disassociating address")
			},
		},
		err: fmt.Errorf("failed to disassociate Elastic IP"),
	}
	testTable["TestSuccess"] = fixture{
		ec2: mockEC2{
			DisassociateFunc: func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
				return &ec2.DisassociateAddressOutput{}, nil
			},
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			svc := awsSvc{
				ec2: test.ec2,
			}

			err := svc.disassociateAddr("associationID")
			if test.err != nil && !strings.Contains(err.Error(), test.err.Error()) {
				t.Errorf("expected error: '%v', got: '%v'", test.err, err)
			}
			if test.err == nil && err != nil {
				t.Errorf("expected error: nil, got: '%v'", err)
			}
		})
	}
}
