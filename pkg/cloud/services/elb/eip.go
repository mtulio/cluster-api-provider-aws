package elb

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/pkg/errors"
)

func (s *Service) createLBAddressesFromPublicIpv4Pool(input *elbv2.CreateLoadBalancerInput, role string) error {
	// Does not require
	if s.scope.VPC().PublicIpv4Pool == nil {
		return nil
	}

	// Only NLB is supported
	if *input.Type != string(elbv2.LoadBalancerTypeEnumNetwork) {
		return fmt.Errorf("custom PublicIpv4Pool is supported only with Network Load Balancer type: %s", *input.Type)
	}

	// Custom SubnetMappings should not be defined, replaced user-defined mapping
	if len(input.SubnetMappings) > 0 {
		return fmt.Errorf("custom PublicIpv4Pool is mutual exclusive with SubnetMappings: %v", input.SubnetMappings)
	}

	eips, err := s.eip.GetOrAllocateAddresses(len(input.Subnets), role)
	if err != nil {
		return errors.Wrapf(err, "failed to allocate EIP from Custom Public IPv4 Pool %v to role: %s", s.scope.VPC().PublicIpv4Pool, role)
	}
	if len(eips) != len(input.Subnets) {
		return fmt.Errorf("allocated EIPs (%d) missmatch with the subnet count (%d)", len(eips), len(input.Subnets))
	}
	for cnt, sb := range input.Subnets {
		input.SubnetMappings = append(input.SubnetMappings, &elbv2.SubnetMapping{
			SubnetId:     aws.String(*sb),
			AllocationId: aws.String(eips[cnt]),
		})
	}
	// Subnets and SubnetMappings are mutual exclusive. Cleaning Subnets when BYO IP is defined,
	// and SubnetMappings are mounted.
	input.Subnets = []*string{}

	return nil
}
