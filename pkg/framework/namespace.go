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
	"fmt"
	"strings"
	"sync"
	"time"

	projectv1 "github.com/openshift/client-go/project/clientset/versioned"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"github.com/rh-messaging/shipshape/pkg/framework/util"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	openapiv1 "github.com/openshift/api/project/v1"
)

const (
	// namespaceNamePrefix is the prefix used when creating namespaces with a random name.
	namespaceNamePrefix     = "operator-e2e-"
	NamespaceCleanupTimeout = 2 * time.Minute
)

func createTestProject(client projectv1.Interface, name string, labels map[string]string) *openapiv1.Project {
	ginkgo.By(fmt.Sprintf("Creating a project named %s to execute the tests in", name))
	return createProject(client, name, labels)
}

func createProject(client projectv1.Interface, name string, labels map[string]string) *openapiv1.Project {
	projectObj := &openapiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}

	project, err := client.ProjectV1().Projects().Create(context.TODO(), projectObj, metav1.CreateOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Error creating project %v", project)
	return project
}

func (c *ContextData) CreateProject(clientSet *projectv1.Clientset,
	baseName string, labels map[string]string) *openapiv1.Project {
	ns := createTestProject(clientSet, baseName, labels)
	c.AddProjectsToDelete(ns)
	return ns
}

func createTestNamespace(client clientset.Interface, name string, labels map[string]string) *corev1.Namespace {
	ginkgo.By(fmt.Sprintf("Creating a namespace %s to execute the test in", name))
	namespace := createNamespace(client, name, labels)
	return namespace
}

func createNamespace(client clientset.Interface, name string, labels map[string]string) *corev1.Namespace {
	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}

	namespace, err := client.CoreV1().Namespaces().Create(context.TODO(), namespaceObj, metav1.CreateOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Error creating namespace %v", namespaceObj)
	return namespace
}

// CreateNamespace creates a namespace for e2e testing.
func (c *ContextData) CreateNamespace(clientSet *clientset.Clientset,
	baseName string, labels map[string]string) *corev1.Namespace {

	ns := createTestNamespace(clientSet, baseName, labels)
	c.AddNamespacesToDelete(ns)
	return ns
}

func (c *ContextData) AddProjectsToDelete(projects ...*openapiv1.Project) {
	for _, proj := range projects {
		if proj == nil {
			continue
		}
		c.projectsToDelete = append(c.projectsToDelete, proj)
	}
}

func (c *ContextData) AddNamespacesToDelete(namespaces ...*corev1.Namespace) {
	for _, ns := range namespaces {
		if ns == nil {
			continue
		}
		c.namespacesToDelete = append(c.namespacesToDelete, ns)
	}
}

func generateProject(client projectv1.Interface, baseName string, labels map[string]string) *openapiv1.Project {
	projectObj := &openapiv1.ProjectRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("shipshape-%v-%v", baseName, util.String(5)),
			Labels: labels,
		},
	}

	project, err := client.ProjectV1().ProjectRequests().Create(context.TODO(), projectObj, metav1.CreateOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Error generating project %v", projectObj)
	return project
}

func generateNamespace(client clientset.Interface, baseName string, labels map[string]string) *corev1.Namespace {
	namespaceObj := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   fmt.Sprintf("shipshape-%v-%v", baseName, util.String(5)),
			Labels: labels,
		},
	}

	namespace, err := client.CoreV1().Namespaces().Create(context.TODO(), namespaceObj, metav1.CreateOptions{})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Error generating namespace %v", namespaceObj)
	return namespace
}

// GenerateNamespace creates a namespace with a random name.
func (c *ContextData) GenerateNamespace() (*corev1.Namespace, error) {
	return c.Clients.KubeClient.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%v-%v", namespaceNamePrefix, util.String(5)),
		},
	}, metav1.CreateOptions{})
}

func (c *ContextData) GenerateProject() (*openapiv1.Project, error) {
	return c.Clients.OcpClient.ProjectsClient.ProjectV1().Projects().Create(context.TODO(), &openapiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%v-%v", namespaceNamePrefix, util.String(5)),
		},
	}, metav1.CreateOptions{})
}

func deleteNamespace(client clientset.Interface, namespaceName string) error {

	if !TestContext.DeleteNamespace {
		log.Logf("Skipping as namespaces are meant to be preserved")
		return nil
	}

	return client.CoreV1().Namespaces().Delete(context.TODO(),
		namespaceName,
		metav1.DeleteOptions{})

}

func deleteProject(client projectv1.Interface, projectName string) error {
	if !TestContext.DeleteNamespace {
		log.Logf("Skipping removal as projects are meant to be preserved")
		return nil
	}
	return client.ProjectV1().Projects().Delete(context.TODO(), projectName, metav1.DeleteOptions{})
}

func (c *ContextData) DeleteProject(prj *openapiv1.Project) []error {
	var errors []error

	if err := deleteProject(c.Clients.OcpClient.ProjectsClient, prj.Name); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			log.Logf("Namespace was already deleted")
		case apierrors.IsConflict(err):
			log.Logf("Namespace scheduled for deletion, resources being purged")
		default:
			log.Logf("Failed deleting namespace")
			errors = append(errors, err)
		}
	}

	return errors
}

func (c *ContextData) DeleteNamespace(ns *corev1.Namespace) []error {
	var errors []error

	if err := deleteNamespace(c.Clients.KubeClient, ns.Name); err != nil {
		switch {
		case apierrors.IsNotFound(err):
			log.Logf("Namespace was already deleted")
		case apierrors.IsConflict(err):
			log.Logf("Namespace scheduled for deletion, resources being purged")
		default:
			log.Logf("Failed deleting namespace")
			errors = append(errors, err)
		}
	}

	return errors
}

// DeleteNamespaces deletes all namespaces that match the given delete and skip filters.
// Filter is by simple strings.Contains; first skip filter, then delete filter.
// Returns the list of deleted namespaces or an error.
func DeleteNamespaces(c clientset.Interface, deleteFilter, skipFilter []string) ([]string, error) {
	ginkgo.By("Deleting namespaces")
	nsList, err := c.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	ExpectNoError(err, "Failed to get namespace list")
	var deleted []string
	var wg sync.WaitGroup
OUTER:
	for _, item := range nsList.Items {
		if skipFilter != nil {
			for _, pattern := range skipFilter {
				if strings.Contains(item.Name, pattern) {
					continue OUTER
				}
			}
		}
		if deleteFilter != nil {
			var shouldDelete bool
			for _, pattern := range deleteFilter {
				if strings.Contains(item.Name, pattern) {
					shouldDelete = true
					break
				}
			}
			if !shouldDelete {
				continue OUTER
			}
		}
		wg.Add(1)
		deleted = append(deleted, item.Name)
		go func(nsName string) {
			defer wg.Done()
			defer ginkgo.GinkgoRecover()
			gomega.Expect(c.CoreV1().Namespaces().Delete(context.TODO(), nsName, metav1.DeleteOptions{})).To(gomega.Succeed())
			log.Logf("namespace : %v api call to delete is complete ", nsName)
		}(item.Name)
	}
	wg.Wait()
	return deleted, nil
}

// WaitForNamespacesDeleted waits for the namespaces to be deleted.
func WaitForNamespacesDeleted(c clientset.Interface, namespaces []string, timeout time.Duration) error {
	ginkgo.By("Waiting for namespaces to vanish")
	nsMap := map[string]bool{}
	for _, ns := range namespaces {
		nsMap[ns] = true
	}

	return wait.Poll(2*time.Second, timeout,
		func() (bool, error) {
			nsList, err := c.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
			if err != nil {
				return false, err
			}
			for _, item := range nsList.Items {
				if _, ok := nsMap[item.Name]; ok {
					return false, nil
				}
			}
			return true, nil
		})
}
