// Copyright 2018 The Operator-SDK Authors
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
	"github.com/fgiorgetti/shipshape/pkg/framework/log"
	"github.com/onsi/gomega"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (c *ContextData) GetDeployment(name string) (*appsv1.Deployment, error) {
	return c.Clients.KubeClient.AppsV1().Deployments(c.Namespace).Get(name, metav1.GetOptions{})
}

func (c *ContextData) ListPodsForDeploymentName(name string) (*corev1.PodList, error) {
	// Retrieves the deployment
	deployment, err := c.GetDeployment(name)
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(deployment).NotTo(gomega.BeNil())

	// Returns the list of pods
	return c.ListPodsForDeployment(deployment)
}

func (c *ContextData) ListPodsForDeployment(deployment *appsv1.Deployment) (*corev1.PodList, error) {
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return nil, err
	}
	listOps := metav1.ListOptions{LabelSelector: selector.String()}
	return c.Clients.KubeClient.CoreV1().Pods(c.Namespace).List(listOps)
}

func WaitForDeployment(kubeclient kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Logf("Waiting for availability of %s deployment\n", name)
				return false, nil
			}
			return false, err
		}

		if int(deployment.Status.AvailableReplicas) == replicas {
			return true, nil
		}
		log.Logf("Waiting for full availability of %s deployment (%d/%d)\n", name, deployment.Status.AvailableReplicas, replicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("Deployment available (%d/%d)\n", replicas, replicas)
	return nil
}

func (c *ContextData) GetDaemonSet(name string) (*appsv1.DaemonSet, error) {
	return c.Clients.KubeClient.AppsV1().DaemonSets(c.Namespace).Get(name, metav1.GetOptions{})
}

func WaitForDaemonSet(kubeclient kubernetes.Interface, namespace, name string, count int, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ds, err := kubeclient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Logf("Waiting for availability of %s daemon set\n", name)
				return false, nil
			}
			return false, err
		}

		if int(ds.Status.NumberReady) >= count {
			return true, nil
		}
		log.Logf("Waiting for full availability of %s daemonset (%d/%d)\n", name, ds.Status.NumberReady, count)
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("Daemonset ready (%d)\n", count)
	return nil
}

func WaitForDeletion(t *testing.T, dynclient client.Client, obj runtime.Object, retryInterval, timeout time.Duration) error {
	key, err := client.ObjectKeyFromObject(obj)
	if err != nil {
		return err
	}

	kind := obj.GetObjectKind().GroupVersionKind().Kind
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = dynclient.Get(ctx, key, obj)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		log.Logf("Waiting for %s %s to be deleted\n", kind, key)
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("%s %s was deleted\n", kind, key)
	return nil
}
