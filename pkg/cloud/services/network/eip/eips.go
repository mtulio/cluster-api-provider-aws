/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eip

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/awserrors"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/filter"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/services/wait"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/tags"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/record"
)

func (s *Service) allocateAddress(role string) (string, error) {
	tagSpecifications := tags.BuildParamsToTagSpecification(ec2.ResourceTypeElasticIp, s.getEIPTagParams(role))
	allocInput := &ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
		TagSpecifications: []*ec2.TagSpecification{
			tagSpecifications,
		},
	}

	// check and set BYO IP pool when available
	err := s.setByoPublicIpv4(allocInput)
	if err != nil {
		return "", err
	}

	out, err := s.EC2Client.AllocateAddressWithContext(context.TODO(), allocInput)
	if err != nil {
		record.Warnf(s.InfraCluster(), "FailedAllocateEIP", "Failed to allocate Elastic IP for %q: %v", role, err)
		return "", errors.Wrap(err, "failed to allocate Elastic IP")
	}
	return aws.StringValue(out.AllocationId), nil
}

// setByoPublicIpv4 check if the config has Public IPv4 Pool defined, then
// check if there are IPs available to consume from allocation, otherwise
// fallback to Amazon pool when explicty failure isn't defined.
func (s *Service) setByoPublicIpv4(alloc *ec2.AllocateAddressInput) error {
	// no BYO IP set, do nothing
	if s.VPC().PublicIpv4Pool == nil {
		return nil
	}

	// check if pool has free IP
	ok, err := s.publicIpv4PoolHasFreeIPs(1)
	if err != nil {
		record.Warnf(s.InfraCluster(), "FailedAllocateEIP", "Failed to allocate Elastic IP in Public IPv4 Pool %s", s.VPC().PublicIpv4Pool)
		return errors.New("failed to allocate Elastic IP from PublicIpv4 Pool")
	}

	// has free IPs
	if ok {
		alloc.PublicIpv4Pool = s.VPC().PublicIpv4Pool
		return nil
	}

	// fail when fallback is forced to 'none' (avoid fallback to Amazon pool [default])
	if s.VPC().PublicIpv4PoolFallBackOrder != nil && s.VPC().PublicIpv4PoolFallBackOrder.Equal(infrav1.PublicIpv4PoolFallbackOrderNone) {
		record.Warnf(s.InfraCluster(), "FailedAllocateEIPFromBYOIP", "Failed to allocate Elastic IP in Public IPv4 Pool %s and fallback isnt enabled", s.VPC().PublicIpv4Pool)
		return fmt.Errorf("failed to allocate Elastic IP from PublicIpv4 Pool and use fallback with strategy %s", *s.VPC().PublicIpv4PoolFallBackOrder)
	}

	// default use Amazon pool
	return nil
}

func (s *Service) describeAddresses(role string) (*ec2.DescribeAddressesOutput, error) {
	x := []*ec2.Filter{filter.EC2.Cluster(s.Name())}
	if role != "" {
		x = append(x, filter.EC2.ProviderRole(role))
	}

	return s.EC2Client.DescribeAddressesWithContext(context.TODO(), &ec2.DescribeAddressesInput{
		Filters: x,
	})
}

func (s *Service) disassociateAddress(ip *ec2.Address) error {
	err := wait.WaitForWithRetryable(wait.NewBackoff(), func() (bool, error) {
		_, err := s.EC2Client.DisassociateAddressWithContext(context.TODO(), &ec2.DisassociateAddressInput{
			AssociationId: ip.AssociationId,
		})
		if err != nil {
			cause, _ := awserrors.Code(errors.Cause(err))
			if cause != awserrors.AssociationIDNotFound {
				return false, err
			}
		}
		return true, nil
	}, awserrors.AuthFailure)
	if err != nil {
		record.Warnf(s.InfraCluster(), "FailedDisassociateEIP", "Failed to disassociate Elastic IP %q: %v", *ip.AllocationId, err)
		return errors.Wrapf(err, "failed to disassociate Elastic IP %q", *ip.AllocationId)
	}
	return nil
}

// releaseAddress releases an given EIP address back to the pool.
func (s *Service) releaseAddress(ip *ec2.Address) error {
	if ip.AssociationId != nil {
		if _, err := s.EC2Client.DisassociateAddressWithContext(context.TODO(), &ec2.DisassociateAddressInput{
			AssociationId: ip.AssociationId,
		}); err != nil {
			record.Warnf(s.InfraCluster(), "FailedDisassociateEIP", "Failed to disassociate Elastic IP %q: %v", *ip.AllocationId, err)
			return errors.Errorf("failed to disassociate Elastic IP %q with allocation ID %q: Still associated with association ID %q", *ip.PublicIp, *ip.AllocationId, *ip.AssociationId)
		}
	}

	if err := wait.WaitForWithRetryable(wait.NewBackoff(), func() (bool, error) {
		_, err := s.EC2Client.ReleaseAddressWithContext(context.TODO(), &ec2.ReleaseAddressInput{AllocationId: ip.AllocationId})
		if err != nil {
			if ip.AssociationId != nil {
				if s.disassociateAddress(ip) != nil {
					return false, err
				}
			}
			return false, err
		}
		return true, nil
	}, awserrors.AuthFailure, awserrors.InUseIPAddress); err != nil {
		record.Warnf(s.InfraCluster(), "FailedReleaseEIP", "Failed to disassociate Elastic IP %q: %v", *ip.AllocationId, err)
		return errors.Wrapf(err, "failed to release ElasticIP %q", *ip.AllocationId)
	}

	s.Info("released ElasticIP", "eip", *ip.PublicIp, "allocation-id", *ip.AllocationId)
	return nil
}

// releaseAddresses releases all address based in the filter.
func (s *Service) releaseAddresses(filters []*ec2.Filter) error {
	out, err := s.EC2Client.DescribeAddressesWithContext(context.TODO(), &ec2.DescribeAddressesInput{
		Filters: filters,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to describe elastic IPs %q", err)
	}
	if out == nil {
		return nil
	}
	for i := range out.Addresses {
		err := s.releaseAddress(out.Addresses[i])
		if err != nil {
			return err
		}
	}
	return nil
}

// ReleaseAddresses releases all EIPs filtering by tag CAPA cluster name.
func (s *Service) ReleaseAddresses() error {
	return s.releaseAddresses([]*ec2.Filter{filter.EC2.Cluster(s.Name())})
}

// ReleaseAddressByRole releases EIP addresses filtering by tag CAPA provider role.
func (s *Service) ReleaseAddressByRole(role string) error {
	clusterFilter := []*ec2.Filter{filter.EC2.Cluster(s.Name())}
	clusterFilter = append(clusterFilter, filter.EC2.ProviderRole(role))

	return s.releaseAddresses(clusterFilter)
}

// getEIPTagParams build the tags for EIP.
func (s *Service) getEIPTagParams(role string) infrav1.BuildParams {
	name := fmt.Sprintf("%s-eip-%s", s.Name(), role)

	return infrav1.BuildParams{
		ClusterName: s.Name(),
		Lifecycle:   infrav1.ResourceLifecycleOwned,
		Name:        aws.String(name),
		Role:        aws.String(role),
		Additional:  s.AdditionalTags(),
	}
}

// publicIpv4PoolHasFreeIPs check if there are N IPs address available in a Public IPv4 Pool.
func (s *Service) publicIpv4PoolHasFreeIPs(want int64) (bool, error) {
	pools, err := s.EC2Client.DescribePublicIpv4Pools(&ec2.DescribePublicIpv4PoolsInput{
		PoolIds: []*string{s.VPC().PublicIpv4Pool},
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to describe Public IPv4 Pool %v: %q", s.VPC().PublicIpv4Pool, err)
	}
	if len(pools.PublicIpv4Pools) != 1 {
		return false, fmt.Errorf("unexpected number of Public IPv4 Pools. want 1, got %d", len(pools.PublicIpv4Pools))
	}
	freeIPs := aws.Int64Value(pools.PublicIpv4Pools[0].TotalAvailableAddressCount)
	if freeIPs < want {
		return false, fmt.Errorf("the pool %v does not have requested size. want %d, got %d", s.VPC().PublicIpv4Pool, want, freeIPs)
	}
	s.Debug("public IPv4 pool has %d IPs available", "eip", freeIPs)
	return true, nil
}

// GetOrAllocateAddresses is a best effort function to lookup unused EIP by role, or allocate it when none.
func (s *Service) GetOrAllocateAddresses(num int, role string) (eips []string, err error) {
	out, err := s.describeAddresses(role)
	if err != nil {
		record.Eventf(s.InfraCluster(), "FailedDescribeAddresses", "Failed to query addresses for role %q: %v", role, err)
		return nil, errors.Wrap(err, "failed to query addresses")
	}

	// Reuse existing unallocated addreses with the same role.
	for _, address := range out.Addresses {
		if address.AssociationId == nil {
			eips = append(eips, aws.StringValue(address.AllocationId))
		}
	}

	// allocate addresses when missing unassociated EIPs
	for len(eips) < num {
		ip, err := s.allocateAddress(role)
		if err != nil {
			return nil, err
		}
		eips = append(eips, ip)
	}

	return eips, nil
}
