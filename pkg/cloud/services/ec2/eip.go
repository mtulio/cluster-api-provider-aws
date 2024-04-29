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

// ReconcileElasticIp reconciles the elastic IP from a custom Public IPv4 Pool.
func (s *Service) ReconcileElasticIpFromPublicPool(instance *infrav1.Instance) error {
	additionalTags := s.scope.AdditionalTags()
	fmt.Println("Placeholder", additionalTags)
	// TODO
	// Check if the instance is in the state allowing EIP association.
	// Expected instance states: pending (?) or running
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-lifecycle.html
	fmt.Println("\n INSTANCE STATE: ", instance.ID)
	fmt.Printf("\n\n INSTANCE DEBUG 2 \n\n")
	if err := s.getAndAssociateAddressesToInstance(fmt.Sprintf("ec2-%s", instance.ID), instance.ID); err != nil {
		fmt.Printf("\n\n ASSOCIATE ERR %v \n\n", err)
		return fmt.Errorf("failed to allocate EIP from Custom Public IPv4 Pool %v: %v", s.scope.VPC().PublicIpv4Pool, err)
	}
	fmt.Printf("\n\n INSTANCE DEBUG X \n\n")
	return nil
}

// ReconcileElasticIp reconciles the elastic IP from a custom Public IPv4 Pool.
func (s *Service) ReleaseElasticIp(instanceId string) error {
	// TODO create a function to generate the role name

	if err := s.eip.ReleaseAddressWithRole(fmt.Sprintf("ec2-%s", instanceId)); err != nil {
		return err
	}
	return nil
}

// getAndAssociateAddressesToInstance find or create an EIP from an instance
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
