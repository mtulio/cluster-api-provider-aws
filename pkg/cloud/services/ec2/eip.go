package ec2

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/record"
)

// ReconcileElasticIPFromPublicPool reconciles the elastic IP from a custom Public IPv4 Pool.
func (s *Service) ReconcileElasticIPFromPublicPool(instance *infrav1.Instance) error {
	// Check if the instance is in the state allowing EIP association.
	// Expected instance states: pending (?) or running
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html
	if err := s.getAndAssociateAddressesToInstance(fmt.Sprintf("ec2-%s", instance.ID), instance.ID); err != nil {
		return fmt.Errorf("failed to reconcile EIP: %v", err)
	}
	return nil
}

// ReleaseElasticIP reconciles the elastic IP from a custom Public IPv4 Pool.
func (s *Service) ReleaseElasticIP(instanceID string) error {
	return s.eip.ReleaseAddressByRole(fmt.Sprintf("ec2-%s", instanceID))
}

// getAndAssociateAddressesToInstance find or create an EIP from an instance and role.
func (s *Service) getAndAssociateAddressesToInstance(role string, instance string) (err error) {
	eips, err := s.eip.GetOrAllocateAddresses(1, role)
	if err != nil {
		record.Warnf(s.scope.InfraCluster(), "FailedAllocateEIP", "Failed to get Elastic IP for %q: %v", role, err)
		return err
	}
	if len(eips) != 1 {
		record.Warnf(s.scope.InfraCluster(), "FailedAllocateEIP", "Failed to allocate Elastic IP for %q: %v", role, err)
		return errors.Wrapf(err, "unexpected number of Elastic IP to instance %s: %d", instance, len(eips))
	}
	_, err = s.EC2Client.AssociateAddressWithContext(context.TODO(), &ec2.AssociateAddressInput{
		InstanceId:   aws.String(instance),
		AllocationId: aws.String(eips[0]),
	})
	if err != nil {
		record.Warnf(s.scope.InfraCluster(), "FailedAssociateEIP", "Failed to associate Elastic IP for %q: %v", role, err)
		return errors.Wrapf(err, "failed to associate Elastic IP %s to instance %s", eips[0], instance)
	}
	return nil
}
