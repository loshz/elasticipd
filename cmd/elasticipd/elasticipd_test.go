package main

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	eIP string = "127.0.0.1"
)

type mockEC2 struct {
	DescribeAddrFunc     func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error)
	DescribeInsFunc      func(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error)
	AssociateAddrFunc    func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error)
	DisassociateAddrFunc func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error)
}

func (m mockEC2) DescribeAddresses(ctx context.Context, input *ec2.DescribeAddressesInput, opts ...func(*ec2.Options)) (*ec2.DescribeAddressesOutput, error) {
	return m.DescribeAddrFunc(input)
}
func (m mockEC2) DescribeInstances(ctx context.Context, input *ec2.DescribeInstancesInput, opts ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.DescribeInsFunc(input)
}
func (m mockEC2) AssociateAddress(ctx context.Context, input *ec2.AssociateAddressInput, opts ...func(*ec2.Options)) (*ec2.AssociateAddressOutput, error) {
	return m.AssociateAddrFunc(input)
}
func (m mockEC2) DisassociateAddress(ctx context.Context, input *ec2.DisassociateAddressInput, opts ...func(*ec2.Options)) (*ec2.DisassociateAddressOutput, error) {
	return m.DisassociateAddrFunc(input)
}

type mockMetadata struct {
	GetFunc func() (*imds.GetInstanceIdentityDocumentOutput, error)
}

func (m mockMetadata) GetInstanceIdentityDocument(context.Context, *imds.GetInstanceIdentityDocumentInput, ...func(*imds.Options)) (*imds.GetInstanceIdentityDocumentOutput, error) {
	return m.GetFunc()
}

func TestGetAssociation(t *testing.T) {
	tests := []struct {
		name string
		ec2  associaterDescriber
		err  error
	}{
		{
			name: "TestDescribeAddrError",
			ec2: mockEC2{
				DescribeAddrFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
					return nil, fmt.Errorf("describe error")
				},
			},
			err: fmt.Errorf("error describing address: describe error"),
		},
		{
			name: "TestNoAddrsError",
			ec2: mockEC2{
				DescribeAddrFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
					return &ec2.DescribeAddressesOutput{}, nil
				},
			},
			err: fmt.Errorf("failed to find address info"),
		},
		{
			name: "TestNoAllocationError",
			ec2: mockEC2{
				DescribeAddrFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
					return &ec2.DescribeAddressesOutput{
						Addresses: []types.Address{{}},
					}, nil
				},
			},
			err: fmt.Errorf("allocation id is nil"),
		},
		{
			name: "TestSuccess",
			ec2: mockEC2{
				DescribeAddrFunc: func(*ec2.DescribeAddressesInput) (*ec2.DescribeAddressesOutput, error) {
					return &ec2.DescribeAddressesOutput{
						Addresses: []types.Address{{
							AllocationId:  aws.String("eipalloc-123"),
							AssociationId: aws.String("eipassoc-123"),
							InstanceId:    aws.String("i-123"),
						}},
					}, nil
				},
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := svc{
				ec2: tt.ec2,
			}

			assoc, err := s.getAssociation(eIP)
			if err != nil {
				if tt.err == nil {
					t.Fatalf("expected error: 'nil', got: '%v'", err)
				}

				if err.Error() != tt.err.Error() {
					t.Fatalf("expected error: '%v', got: '%v'", tt.err, err)
				}
			} else {
				if tt.err != nil {
					t.Fatalf("expected error: '%v', got: 'nil'", tt.err)
				}

				if assoc.id != "eipassoc-123" {
					t.Errorf("expected association id: 'eipassoc-123', got: %q", assoc.id)
				}
				if assoc.instanceID != "i-123" {
					t.Errorf("expected instance id: 'i-123', got: %q", assoc.instanceID)
				}
				if assoc.allocationID != "eipalloc-123" {
					t.Errorf("expected allocation id id: 'eipalloc-123', got: %q", assoc.allocationID)
				}
			}
		})
	}
}

func TestGetInstanceDetails(t *testing.T) {
	tests := []struct {
		name string
		ec2  associaterDescriber
		imds metadata
		err  error
	}{
		{
			name: "TestIdentityDocumentError",
			imds: mockMetadata{
				GetFunc: func() (*imds.GetInstanceIdentityDocumentOutput, error) {
					return nil, fmt.Errorf("identity doc error")
				},
			},
			err: fmt.Errorf("error getting instance identity document: identity doc error"),
		},
		{
			name: "TestDescribeInstancesError",
			imds: mockMetadata{
				GetFunc: func() (*imds.GetInstanceIdentityDocumentOutput, error) {
					return &imds.GetInstanceIdentityDocumentOutput{
						InstanceIdentityDocument: imds.InstanceIdentityDocument{
							InstanceID: "i-123",
						},
					}, nil
				},
			},
			ec2: mockEC2{
				DescribeInsFunc: func(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return nil, fmt.Errorf("describe error")
				},
			},
			err: fmt.Errorf("error describing instance: describe error"),
		},
		{
			name: "TestNoReservationsError",
			imds: mockMetadata{
				GetFunc: func() (*imds.GetInstanceIdentityDocumentOutput, error) {
					return &imds.GetInstanceIdentityDocumentOutput{
						InstanceIdentityDocument: imds.InstanceIdentityDocument{
							InstanceID: "i-123",
						},
					}, nil
				},
			},
			ec2: mockEC2{
				DescribeInsFunc: func(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return &ec2.DescribeInstancesOutput{}, nil
				},
			},
			err: fmt.Errorf("invalid instance description: no reservations"),
		},
		{
			name: "TestNoInstancesError",
			imds: mockMetadata{
				GetFunc: func() (*imds.GetInstanceIdentityDocumentOutput, error) {
					return &imds.GetInstanceIdentityDocumentOutput{
						InstanceIdentityDocument: imds.InstanceIdentityDocument{
							InstanceID: "i-123",
						},
					}, nil
				},
			},
			ec2: mockEC2{
				DescribeInsFunc: func(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return &ec2.DescribeInstancesOutput{
						Reservations: []types.Reservation{{}},
					}, nil
				},
			},
			err: fmt.Errorf("instance not found in reservation"),
		},
		{
			name: "TestSuccess",
			imds: mockMetadata{
				GetFunc: func() (*imds.GetInstanceIdentityDocumentOutput, error) {
					return &imds.GetInstanceIdentityDocumentOutput{
						InstanceIdentityDocument: imds.InstanceIdentityDocument{
							InstanceID: "i-123",
						},
					}, nil
				},
			},
			ec2: mockEC2{
				DescribeInsFunc: func(*ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
					return &ec2.DescribeInstancesOutput{
						Reservations: []types.Reservation{{
							Instances: []types.Instance{{
								NetworkInterfaces: []types.InstanceNetworkInterface{{
									NetworkInterfaceId: aws.String("eni-123"),
								}},
							}},
						}},
					}, nil
				},
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := svc{
				ec2:  tt.ec2,
				imds: tt.imds,
			}

			instance, err := s.getInstanceDetails()
			if err != nil {
				if tt.err == nil {
					t.Fatalf("expected error: 'nil', got: '%v'", err)
				}

				if err.Error() != tt.err.Error() {
					t.Fatalf("expected error: '%v', got: '%v'", tt.err, err)
				}
			} else {
				if tt.err != nil {
					t.Fatalf("expected error: '%v', got: 'nil'", tt.err)
				}
				if instance.id != "i-123" {
					t.Errorf("expected instance id: 'i-123', got: %q", instance.id)
				}
				if len(instance.networkInterfaceIDs) != 1 {
					t.Error("expected 1 instance network interface")
				}
			}
		})
	}
}

func TestAssociateAddress(t *testing.T) {
	t.Run("TestAssociateError", func(t *testing.T) {
		s := svc{
			ec2: mockEC2{
				AssociateAddrFunc: func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error) {
					return nil, fmt.Errorf("associate error")
				},
			},
		}

		if err := s.associateAddr(association{}, instance{}, true); err == nil {
			t.Error("expected association error, got: 'nil'")
		}
	})

	t.Run("TestAssociateSuccess", func(t *testing.T) {
		s := svc{
			ec2: mockEC2{
				AssociateAddrFunc: func(*ec2.AssociateAddressInput) (*ec2.AssociateAddressOutput, error) {
					return nil, nil
				},
			},
		}

		if err := s.associateAddr(association{}, instance{}, true); err != nil {
			t.Errorf("expected error 'nil', got: '%v'", err)
		}
	})
}

func TestDisssociateAddress(t *testing.T) {
	t.Run("TestDisassociateError", func(t *testing.T) {
		s := svc{
			ec2: mockEC2{
				DisassociateAddrFunc: func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
					return nil, fmt.Errorf("disassociate error")
				},
			},
		}

		assoc := association{
			id: "eipassoc-123",
		}

		if err := s.disassociateAddr(assoc); err == nil {
			t.Error("expected disassociation error, got: 'nil'")
		}
	})

	t.Run("TestSuccess", func(t *testing.T) {
		s := svc{
			ec2: mockEC2{
				DisassociateAddrFunc: func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
					return nil, nil
				},
			},
		}

		assoc := association{
			id: "eipassoc-123",
		}

		if err := s.disassociateAddr(assoc); err != nil {
			t.Errorf("expected error 'nil', got: '%v'", err)
		}
	})

	t.Run("TestNoIDSuccess", func(t *testing.T) {
		s := svc{
			ec2: mockEC2{
				DisassociateAddrFunc: func(*ec2.DisassociateAddressInput) (*ec2.DisassociateAddressOutput, error) {
					return nil, nil
				},
			},
		}

		if err := s.disassociateAddr(association{}); err != nil {
			t.Errorf("expected error 'nil', got: '%v'", err)
		}
	})
}
