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
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
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


func WaitForStatefulSet(kubeclient kubernetes.Interface, namespace, name string, count int, retryInterval, timeout time.Duration) error { // I'd deprecate this method but it might be used in tests etc. 
	return WaitForStatefulSetReady(kubeclient,namespace,name,count,retryInterval, timeout)
}

func WaitForStatefulSetReady(kubeclient kubernetes.Interface, namespace, name string, count int, retryInterval, timeout time.Duration) error {
    err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		ds, err := kubeclient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Logf("Waiting for availability of %s stateful set", name)
				return false, nil
			}
			return false, err
		}
		//Needs to be *exactly* the count, not count or more. This matters for our tests.
		log.Logf("Waiting for full availability of %s stateful set (%d/%d)", name, ds.Status.ReadyReplicas, count)
		if int(ds.Status.ReadyReplicas) == count {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("Statefulset ready (%d)", count)
	return nil}

func WaitForStatefulSetCreation(kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
    err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		_, err = kubeclient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Logf("Waiting for availability of %s stateful set", name)
				return false, nil
			}
			return false, err
		} else {
            return true, nil
        }
	})
	if err != nil {
		return err
	}
	log.Logf("Statefulset created (%d)")
	return nil
}

func WaitForStatefulSetCreation(kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		_, err = kubeclient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Logf("Waiting for availability of %s stateful set", name)
				return false, nil
			}
			return false, err
		} else {
			return true, nil
		}
	})
	if err != nil {
		return err
	}
	log.Logf("Statefulset created (%d)")
	return nil
}

func WaitForDeployment(kubeclient kubernetes.Interface, namespace, name string, replicas int, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				log.Logf("Waiting for availability of %s deployment in %s namepsace", name, namespace)
				return false, nil
			}
			return false, err
		}

		// For debugging purposes
		log.Logf("Waiting for full availability of %s deployment (Replicas: %d/%d - Available: %d/%d - Updated: %d/%d - Ready: %d/%d - Unavailable: %d/%d)",
			name,
			deployment.Status.Replicas, replicas,
			deployment.Status.AvailableReplicas, replicas,
			deployment.Status.UpdatedReplicas, replicas,
			deployment.Status.ReadyReplicas, replicas,
			deployment.Status.UnavailableReplicas, replicas)

		totalReplicas := int(deployment.Status.Replicas) == replicas
		available := int(deployment.Status.AvailableReplicas) == replicas
		updated := int(deployment.Status.ReadyReplicas) == replicas
		ready := int(deployment.Status.ReadyReplicas) == replicas
		unavailable := int(deployment.Status.UnavailableReplicas) != 0

		if totalReplicas && available && updated && ready && !unavailable {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("Deployment available (%d/%d)", replicas, replicas)
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
				log.Logf("Waiting for availability of %s daemon set", name)
				return false, nil
			}
			return false, err
		}

		if int(ds.Status.NumberReady) >= count {
			return true, nil
		}
		log.Logf("Waiting for full availability of %s daemonset (%d/%d)", name, ds.Status.NumberReady, count)
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("Daemonset ready (%d)", count)
	return nil
}

func WaitForDeletion(dynclient client.Client, obj runtime.Object, retryInterval, timeout time.Duration) error {
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
		log.Logf("Waiting for %s %s to be deleted", kind, key)
		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("%s %s was deleted", kind, key)
	return nil
}

func WaitForDeploymentDeleted(ctx context.Context, kubeclient kubernetes.Interface, namespace, name string) error {
	err := RetryWithContext(ctx, RetryInterval, func() (bool, error) {
		deployment, err := kubeclient.AppsV1().Deployments(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		// For debugging purposes
		log.Logf("Waiting for deletion of %s deployment (Replicas: %d/0 - Available: %d/0 - Updated: %d/0 - Ready: %d/0 - Unavailable: %d/0)",
			name,
			deployment.Status.Replicas,
			deployment.Status.AvailableReplicas,
			deployment.Status.UpdatedReplicas,
			deployment.Status.ReadyReplicas,
			deployment.Status.UnavailableReplicas)

		return false, nil
	})
	if err != nil {
		return err
	}
	log.Logf("Deployment %s no longer present", name)
	return nil
}
