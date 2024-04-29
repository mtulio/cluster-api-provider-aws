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

// Package network provides a service to manage AWS network resources.
package eip

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"
	infrav1 "sigs.k8s.io/cluster-api-provider-aws/v2/api/v1beta2"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud"
	"sigs.k8s.io/cluster-api-provider-aws/v2/pkg/cloud/scope"
)

type eipCurrentScope string

const (
	eipScopeNetwork eipCurrentScope = "network"
	eipScopeELB     eipCurrentScope = "elb"
	eipScopeEC2     eipCurrentScope = "ec2"
)

// Service holds a collection of interfaces.
// The interfaces are broken down like this to group functions together.
// One alternative is to have a large list of functions from the ec2 client.
type Service struct {
	currentScope eipCurrentScope
	scopeNetwork scope.NetworkScope
	scopeELB     scope.ELBScope
	scopeEC2     scope.EC2Scope
	EC2Client    ec2iface.EC2API
}

// NewService returns a new service given the ec2 api client.
func NewServiceWithNetworkScope(scope scope.NetworkScope, ec2 ec2iface.EC2API) *Service {
	return &Service{
		scopeNetwork: scope,
		currentScope: eipScopeNetwork,
		EC2Client:    ec2,
	}
}

func NewServiceWithEC2Scope(scope scope.EC2Scope, ec2 ec2iface.EC2API) *Service {
	return &Service{
		scopeEC2:     scope,
		currentScope: eipScopeEC2,
		EC2Client:    ec2,
	}
}

func NewServiceWithELBScope(scope scope.ELBScope, ec2 ec2iface.EC2API) *Service {
	return &Service{
		scopeELB:     scope,
		currentScope: eipScopeELB,
		EC2Client:    ec2,
	}
}

func (s *Service) InfraCluster() cloud.ClusterObject {
	switch s.currentScope {
	case eipScopeNetwork:
		return s.scopeNetwork.InfraCluster()
	case eipScopeEC2:
		return s.scopeEC2.InfraCluster()
	case eipScopeELB:
		return s.scopeELB.InfraCluster()
	}
	return nil
}

func (s *Service) VPC() *infrav1.VPCSpec {
	switch s.currentScope {
	case eipScopeNetwork:
		return s.scopeNetwork.VPC()
	case eipScopeEC2:
		return s.scopeEC2.VPC()
	case eipScopeELB:
		return s.scopeELB.VPC()
	}
	return nil
}

func (s *Service) Name() string {
	switch s.currentScope {
	case eipScopeNetwork:
		return s.scopeNetwork.Name()
	case eipScopeEC2:
		return s.scopeEC2.Name()
	case eipScopeELB:
		return s.scopeELB.Name()
	}
	return ""
}

func (s *Service) AdditionalTags() map[string]string {
	switch s.currentScope {
	case eipScopeNetwork:
		return s.scopeNetwork.AdditionalTags()
	case eipScopeEC2:
		return s.scopeEC2.AdditionalTags()
	case eipScopeELB:
		return s.scopeELB.AdditionalTags()
	}
	return nil
}

func (s *Service) Debug(msg string, keysAndValues ...any) {
	// switch s.currentScope {
	// case eipScopeNetwork:
	// 	s.scopeNetwork.Debug(msg, keysAndValues)
	// 	fmt.Printf("\n DEBUG TMP: %v => %v", msg, keysAndValues)
	// case eipScopeEC2:
	// 	s.scopeEC2.Debug(msg, keysAndValues)
	// case eipScopeELB:
	// 	s.scopeELB.Debug(msg, keysAndValues)
	// }
	fmt.Printf("\n\n DEBUG TMP: %v => %v \n\n", msg, keysAndValues)
}

func (s *Service) Info(msg string, keysAndValues ...any) {
	// switch s.currentScope {
	// case eipScopeNetwork:
	// 	s.scopeNetwork.Info(msg, keysAndValues)
	// case eipScopeEC2:
	// 	s.scopeEC2.Info(msg, keysAndValues)
	// case eipScopeELB:
	// 	s.scopeELB.Info(msg, keysAndValues)
	// }

	fmt.Printf("\n\n INFO TMP: %v => %v \n\n", msg, keysAndValues)
}
