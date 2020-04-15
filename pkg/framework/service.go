// Copyright 2017 The Interconnectedcloud Authors
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

package framework

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func (c *ContextData) GetService(name string) (*corev1.Service, error) {
	return c.Clients.KubeClient.CoreV1().Services(c.Namespace).Get(name, metav1.GetOptions{})
}

func (c *ContextData) WaitForService(name string, timeout time.Duration, interval time.Duration) (*corev1.Service, error) {
	var service *corev1.Service
	var err error
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	err = RetryWithContext(ctx, interval, func() (bool, error) {
		service, err = c.GetService(name)
		if err != nil {
			// service does not exist yet
			return false, nil
		}
		return service != nil, nil
	})

	return service, err
}

// GetPorts returns an int slice with all ports exposed
// by the provided corev1.Service object
func GetPorts(service corev1.Service) []int {
	if len(service.Spec.Ports) == 0 {
		return []int{}
	}
	var svcPorts []int
	for _, port := range service.Spec.Ports {
		svcPorts = append(svcPorts, int(port.Port))
	}
	return svcPorts
}
