package main

import (
	"errors"
	"fmt"
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
	e := "describe address error"

	testTable := make(map[string]fixture)
	testTable["TestDescribeAddressesError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{}, errors.New(e)
			},
		},
		err: fmt.Errorf("error describing address: %s", e),
	}
	testTable["TestAddressesLengthError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{}, nil
			},
		},
		err: fmt.Errorf("failed to find address: %s", ip),
	}
	testTable["TestInstanceIdNilError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{
					Addresses: []*ec2.Address{
						&ec2.Address{
							InstanceId:    nil,
							AssociationId: aws.String("2"),
							AllocationId:  aws.String("3"),
						},
					},
				}, nil
			},
		},
		err: errors.New("InstanceId is nil"),
	}
	testTable["TestAssociationIdNilError"] = fixture{
		ec2: mockEC2{
			DescribeFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
				return &ec2.DescribeAddressesOutput{
					Addresses: []*ec2.Address{
						&ec2.Address{
							InstanceId:    aws.String("1"),
							AssociationId: nil,
							AllocationId:  aws.String("3"),
						},
					},
				}, nil
			},
		},
		err: errors.New("AssociationId is nil"),
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
		err: errors.New("AllocationId is nil"),
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
				return ec2metadata.EC2InstanceIdentityDocument{}, errors.New(e)
			},
		},
		err: fmt.Errorf("error getting instance identity document: %s", e),
	}
	testTable["TestAssociatedInstanceError"] = fixture{
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
				return ec2metadata.EC2InstanceIdentityDocument{
					InstanceID: "1",
				}, nil
			},
		},
		err: errAssociated,
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
			c := &ipChecker{
				ec2:         test.ec2,
				ec2metadata: test.metadata,
			}

			_, err := c.describeAddr(ip)
			if test.err != nil && test.err.Error() != err.Error() {
				t.Errorf("expected error: '%v', got: '%v'", test.err, err)
			}
			if test.err == nil && err != nil {
				t.Errorf("expected error: nil, got: '%v'", err)
			}
		})
	}
}

func TestAssociateAddress(t *testing.T) {
	e := "error associating address"

	testTable := make(map[string]fixture)
	testTable["TestAssociateError"] = fixture{
		ec2: mockEC2{
			AssociateFunc: func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error) {
				return &ec2.AssociateAddressOutput{}, errors.New(e)
			},
		},
		err: fmt.Errorf("failed to associate address %s: %v", ip, e),
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
			c := &ipChecker{
				ec2: test.ec2,
			}

			err := c.associateAddr(ip, "1", "2")
			if test.err != nil && test.err.Error() != err.Error() {
				t.Errorf("expected error: '%v', got: '%v'", test.err, err)
			}
			if test.err == nil && err != nil {
				t.Errorf("expected error: nil, got: '%v'", err)
			}
		})
	}
}

func TestDisassociateAddress(t *testing.T) {
	e := "error disassociating address"

	testTable := make(map[string]fixture)
	testTable["TestDisassociateError"] = fixture{
		ec2: mockEC2{
			DisassociateFunc: func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
				return &ec2.DisassociateAddressOutput{}, errors.New(e)
			},
		},
		err: fmt.Errorf("failed to disassociate address %s: %v", ip, e),
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
			c := &ipChecker{
				ec2: test.ec2,
			}

			err := c.disassociateAddr(ip, "1", "2")
			if test.err != nil && test.err.Error() != err.Error() {
				t.Errorf("expected error: '%v', got: '%v'", test.err, err)
			}
			if test.err == nil && err != nil {
				t.Errorf("expected error: nil, got: '%v'", err)
			}
		})
	}
}
