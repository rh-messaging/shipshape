// Copyright 2019 The Interconnectedcloud Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package deployment

import (
	"fmt"
	"github.com/interconnectedcloud/qdr-operator/pkg/apis/interconnectedcloud/v1alpha1"
	qdrclient "github.com/interconnectedcloud/qdr-operator/pkg/client/clientset/versioned"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/rh-messaging/shipshape/pkg/framework/operators"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InterconnectCustomizer represents a function that allows for
// customizing an Interconnect resource before it is created.
type InterconnectCustomizer func(interconnect *v1alpha1.Interconnect)

// CreateInterconnectFromSpec creates an Interconnect resource using the provided InterconnectSpec
func CreateInterconnectFromSpec(c framework.ContextData, size int32, name string, spec v1alpha1.InterconnectSpec) (*v1alpha1.Interconnect, error) {
	return CreateInterconnect(c, size, func(ic *v1alpha1.Interconnect) {
		ic.Name = name
		ic.Spec = spec
	})
}

// CreateInterconnect creates an interconnect resource
func CreateInterconnect(c framework.ContextData, size int32, fn ...InterconnectCustomizer) (*v1alpha1.Interconnect, error) {

	const IC_PREFIX = "ic"
	operator := c.OperatorMap[operators.OperatorTypeQdr]
	obj := &v1alpha1.Interconnect{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Interconnect",
			APIVersion: "interconnectedcloud.github.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", IC_PREFIX, c.UniqueName),
			Namespace: c.Namespace,
		},
		Spec: v1alpha1.InterconnectSpec{
			DeploymentPlan: v1alpha1.DeploymentPlanType{
				Size:      size,
				Image:     "quay.io/interconnectedcloud/qdrouterd",
				Role:      "interior",
				Placement: "Any",
			},
		},
	}

	// Customize the interconnect resource before creation
	for _, f := range fn {
		f(obj)
	}
	// create the interconnect resource
	return operator.Interface().(qdrclient.Interface).InterconnectedcloudV1alpha1().Interconnects(c.Namespace).Create(obj)
}

func DeleteInterconnect(c framework.ContextData, interconnect *v1alpha1.Interconnect) error {
	operator := c.OperatorMap[operators.OperatorTypeQdr]
	return operator.Interface().(qdrclient.Interface).InterconnectedcloudV1alpha1().Interconnects(c.Namespace).Delete(interconnect.Name, &metav1.DeleteOptions{})
}

func GetInterconnect(c framework.ContextData, name string) (*v1alpha1.Interconnect, error) {
	operator := c.OperatorMap[operators.OperatorTypeQdr]
	return operator.Interface().(qdrclient.Interface).InterconnectedcloudV1alpha1().Interconnects(c.Namespace).Get(name, metav1.GetOptions{})
}

func UpdateInterconnect(c framework.ContextData, interconnect *v1alpha1.Interconnect) (*v1alpha1.Interconnect, error) {
	operator := c.OperatorMap[operators.OperatorTypeQdr]
	return operator.Interface().(qdrclient.Interface).InterconnectedcloudV1alpha1().Interconnects(c.Namespace).Update(interconnect)
}
