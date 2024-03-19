/*
Copyright 2020 The Kubernetes Authors.

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

// Package eip provides a service to manage AWS Elastic IP resources.
package eip

import (
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/logger"
)

// Service holds a collection of interfaces.
// The interfaces are broken down like this to group functions together.
// One alternative is to have a large list of functions from the ec2 client.
type Service struct {
	logger.Wrapper
	EC2Client      ec2iface.EC2API
	additionalTags infrav1.Tags
	infraCluster   cloud.ClusterObject
	vpc            *infrav1.VPCSpec
	name           string
}

// ServiceInput defines input options to create the EIP service.
type ServiceInput struct {
	EC2Client      ec2iface.EC2API
	AdditionalTags infrav1.Tags
	InfraCluster   cloud.ClusterObject
	VPC            *infrav1.VPCSpec
	Name           string
}

// NewService build the EIP service to be used in different controllers.
func NewService(in *ServiceInput) *Service {
	return &Service{
		EC2Client:      in.EC2Client,
		additionalTags: in.AdditionalTags,
		infraCluster:   in.InfraCluster,
		vpc:            in.VPC,
		name:           in.Name,
	}
}

// InfraCluster is a wrapper for scope.InfraCluster function.
func (s *Service) InfraCluster() cloud.ClusterObject {
	return s.infraCluster
}

// VPC is a wrapper for scope.VPC function.
func (s *Service) VPC() *infrav1.VPCSpec {
	return s.vpc
}

// Name is a wrapper for scope.Name function.
func (s *Service) Name() string {
	return s.name
}

// AdditionalTags is a wrapper for scope.AdditionalTags function.
func (s *Service) AdditionalTags() map[string]string {
	return s.additionalTags
}
