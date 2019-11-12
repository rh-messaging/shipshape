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

package framework

import (
	"fmt"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	qdrclient "github.com/interconnectedcloud/qdr-operator/pkg/client/clientset/versioned"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

const (
	qdrOperatorName = "qdr-operator"
	crdName         = "interconnects.interconnectedcloud.github.io"
	groupName       = "interconnectedcloud.github.io"
	apiVersion      = "v1alpha1"
)

var (
	RetryInterval        = time.Second * 5
	Timeout              = time.Second * 600
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 5
	GVR                  = groupName + "/" + apiVersion
)

type ClientSet struct {
	KubeClient clientset.Interface
	ExtClient  apiextension.Interface
	QdrClient  qdrclient.Interface
}

// ContextData holds clients and data related with namespaces
//             created within
type ContextData struct {
	Id                 string
	Clients            ClientSet
	Namespace          string
	namespacesToDelete []*corev1.Namespace // Some tests have more than one
	// Set together with creating the ClientSet and the namespace.
	// Guaranteed to be unique in the cluster even when running the same
	// test multiple times in parallel.
	UniqueName         string
	CertManagerPresent bool // if crd is detected
}

type Framework struct {
	BaseName string

	// Map that ties clients and namespaces for each available context
	ContextMap map[string]*ContextData

	SkipNamespaceCreation bool // Whether to skip creating a namespace
	cleanupHandleEach     CleanupActionHandle
	cleanupHandleSuite    CleanupActionHandle
	afterEachDone         bool
}

// NewFramework creates a test framework
func NewFramework(baseName string, contexts ...string) *Framework {

	f := &Framework{
		BaseName:   baseName,
		ContextMap: make(map[string]*ContextData),
	}

	f.BeforeEach(contexts...)

	return f
}

// BeforeEach gets clients and makes a namespace
func (f *Framework) BeforeEach(contexts ...string) {

	f.cleanupHandleEach = AddCleanupAction(AfterEach, f.AfterEach)
	f.cleanupHandleSuite = AddCleanupAction(AfterSuite, f.AfterSuite)

	// Loop through contexts
	// 1 - Set the current context
	// 2 - Create the config object
	// 3 - Generate the clients for given context

	ginkgo.By("Creating kubernetes clients")
	config, err := clientcmd.LoadFromFile(TestContext.KubeConfig)
	//if err != nil || config == nil {
	//	fmt.Sprintf("Unable to retrieve config from %s - %s", TestContext.KubeConfig, err))
	//}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	namespaceLabels := map[string]string{
		"e2e-framework": f.BaseName,
	}

	// Loop through provided contexts (or use current-context)
	// and loading all context info
	for _, context := range contexts {

		// Populating ContextMap with clients for each provided context
		var clients ClientSet

		// Set current context and serialize config
		config.CurrentContext = context
		bytes, err := clientcmd.Write(*config)
		if err != nil {
			ginkgo.Fail(fmt.Sprintf("Unable to serialize config %s - %s", TestContext.KubeConfig, err))
		}

		// Generating restConfig
		clientConfig, err := clientcmd.NewClientConfigFromBytes(bytes)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		restConfig, err := clientConfig.ClientConfig()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Create the client instances
		kubeClient, err := clientset.NewForConfig(restConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		extClient, err := apiextension.NewForConfig(restConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		qdrClient, err := qdrclient.NewForConfig(restConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Initilizing the ClientSet for context
		clients = ClientSet{
			KubeClient: kubeClient,
			ExtClient:  extClient,
			QdrClient:  qdrClient,
		}

		// Generating the namespace on provided contexts
		ginkgo.By(fmt.Sprintf("Building namespace api objects, basename %s", f.BaseName))
		// Keep original label for now (maybe we can remove or rename later)
		var namespace *corev1.Namespace
		if !f.SkipNamespaceCreation {
			namespace = generateNamespace(kubeClient, f.BaseName, namespaceLabels)
		}
		gomega.Expect(namespace).NotTo(gomega.BeNil())

		_, err = extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get("issuers.certmanager.k8s.io", metav1.GetOptions{})
		certManagerPresent := false
		if err == nil {
			certManagerPresent = true
		} else if _, err = extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get("issuers.cert-manager.io", metav1.GetOptions{}); err == nil {
			certManagerPresent = true
		}

		f.ContextMap[context] = &ContextData{
			Id:                 context,
			Namespace:          namespace.GetName(),
			UniqueName:         namespace.GetName(),
			Clients:            clients,
			CertManagerPresent: certManagerPresent,
		}

		if !f.SkipNamespaceCreation {
			f.ContextMap[context].AddNamespacesToDelete(namespace)
		}

	}

	// setup the operator
	err = f.Setup()
	if err != nil {
		f.AfterEach()
	}
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

// AfterEach deletes the namespace, after reading its events.
func (f *Framework) AfterEach() {
	// In case already executed, skip
	if f.afterEachDone {
		return
	}

	// Remove cleanup action
	RemoveCleanupAction(AfterEach, f.cleanupHandleEach)

	// teardown the operator
	err := f.TeardownEach()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// DeleteNamespace at the very end in defer, to avoid any
	// expectation failures preventing deleting the namespace.
	defer func() {
		nsDeletionErrors := map[string][]error{}
		// Whether to delete namespace is determined by 3 factors: delete-namespace flag, delete-namespace-on-failure flag and the test result
		// if delete-namespace set to false, namespace will always be preserved.
		// if delete-namespace is true and delete-namespace-on-failure is false, namespace will be preserved if test failed.
		for _, contextData := range f.ContextMap {
			for _, ns := range contextData.namespacesToDelete {
				ginkgo.By(fmt.Sprintf("Destroying namespace %q for this suite on all clusters.", ns.Name))
				if errors := contextData.DeleteNamespace(ns); errors != nil {
					nsDeletionErrors[ns.Name] = errors
				}
			}

			// Paranoia-- prevent reuse!
			contextData.Namespace = ""
			contextData.Clients.KubeClient = nil
			contextData.namespacesToDelete = nil
		}

		// if we had errors deleting, report them now.
		if len(nsDeletionErrors) != 0 {
			messages := []string{}
			for namespaceKey, namespaceErrors := range nsDeletionErrors {
				for clusterIdx, namespaceErr := range namespaceErrors {
					messages = append(messages, fmt.Sprintf("Couldn't delete ns: %q (@cluster %d): %s (%#v)",
						namespaceKey, clusterIdx, namespaceErr, namespaceErr))
				}
			}
			log.Failf(strings.Join(messages, ","))
		}
	}()

	f.afterEachDone = true
}

// AfterSuite deletes the cluster level resources
func (f *Framework) AfterSuite() {
	// Remove cleanup action
	RemoveCleanupAction(AfterSuite, f.cleanupHandleSuite)

	// teardown suite
	err := f.TeardownSuite()
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func (f *Framework) TeardownEach() error {

	// Skip the qdr-operator teardown if the operator image was not specified
	if len(TestContext.OperatorImage) == 0 {
		return nil
	}

	// Iterate through all contexts and deleting namespace related resources
	for _, contextData := range f.ContextMap {
		err := contextData.Clients.KubeClient.CoreV1().ServiceAccounts(contextData.Namespace).Delete(qdrOperatorName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator service account: %v", err)
		}
		err = contextData.Clients.KubeClient.RbacV1().Roles(contextData.Namespace).Delete(qdrOperatorName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator role: %v", err)
		}
		err = contextData.Clients.KubeClient.RbacV1().RoleBindings(contextData.Namespace).Delete(qdrOperatorName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator role binding: %v", err)
		}
		err = contextData.Clients.KubeClient.AppsV1().Deployments(contextData.Namespace).Delete(qdrOperatorName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator deployment: %v", err)
		}
	}

	log.Logf("e2e teardown namespace successful")
	return nil
}

func (f *Framework) TeardownSuite() error {

	// Skip the qdr-operator teardown if the operator image was not specified
	if len(TestContext.OperatorImage) == 0 {
		return nil
	}

	// Iterate through all contexts
	for _, contextData := range f.ContextMap {
		err := contextData.Clients.KubeClient.RbacV1().ClusterRoles().Delete(qdrOperatorName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator cluster role: %v", err)
		}
		err = contextData.Clients.KubeClient.RbacV1().ClusterRoleBindings().Delete(qdrOperatorName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator cluster role binding: %v", err)
		}
		err = contextData.Clients.ExtClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete qdr-operator crd: %v", err)
		}
	}

	log.Logf("e2e teardown suite successful")
	return nil
}

func (f *Framework) Setup() error {

	for _, ctxData := range f.ContextMap {
		err := ctxData.setupQdrServiceAccount()
		if err != nil {
			return fmt.Errorf("failed to setup qdr operator [setupQdrServiceAccount]: %v", err)
		}
		err = ctxData.setupQdrRole()
		if err != nil {
			return fmt.Errorf("failed to setup qdr operator [setupQdrRole]: %v", err)
		}
		err = ctxData.setupQdrClusterRole()
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to setup qdr operator [setupQdrClusterRole]: %T - %v", err, err)
		}
		err = ctxData.setupQdrRoleBinding()
		if err != nil {
			return fmt.Errorf("failed to setup qdr operator [setupQdrRoleBinding]: %v", err)
		}
		err = ctxData.setupQdrClusterRoleBinding()
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to setup qdr operator [setupQdrClusterRoleBinding]: %v", err)
		}
		err = ctxData.setupQdrCrd()
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to setup qdr operator [setupQdrCrd]: %v", err)
		}
		err = ctxData.setupQdrDeployment()
		if err != nil {
			return fmt.Errorf("failed to setup qdr operator [setupQdrDeployment]: %v", err)
		}
		err = WaitForDeployment(ctxData.Clients.KubeClient, ctxData.Namespace, "qdr-operator", 1, RetryInterval, Timeout)
		if err != nil {
			return fmt.Errorf("Failed to wait for qdr operator: %v", err)
		}
	}
	return nil
}

func (c *ContextData) setupQdrServiceAccount() error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: qdrOperatorName,
		},
	}
	_, err := c.Clients.KubeClient.CoreV1().ServiceAccounts(c.Namespace).Create(sa)
	if err != nil {
		return fmt.Errorf("create qdr-operator service account failed: %v", err)
	}
	return nil
}

func (c *ContextData) setupQdrRole() error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: qdrOperatorName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "services", "serviceaccounts", "endpoints", "persistentvolumeclaims", "events", "configmaps", "secrets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"rolebindings", "roles"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{"extensions"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"namespaces"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{"apps"},
				Resources: []string{"deployments", "daemonsets", "replicasets", "statefulsets"},
				Verbs:     []string{"*"},
			},
			{
				APIGroups: []string{"certmanager.k8s.io"},
				Resources: []string{"issuers", "certificates"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{"issuers.cert-manager.io"},
				Resources: []string{"issuers", "certificates"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{"monitoring.coreos.com"},
				Resources: []string{"servicemonitors"},
				Verbs:     []string{"get", "create"},
			},
			{
				APIGroups: []string{"route.openshift.io"},
				Resources: []string{"routes", "routes/custom-host", "routes/status"},
				Verbs:     []string{"get", "list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{"interconnectedcloud.github.io"},
				Resources: []string{"*"},
				Verbs:     []string{"*"},
			},
		},
	}
	_, err := c.Clients.KubeClient.RbacV1().Roles(c.Namespace).Create(role)
	if err != nil {
		return fmt.Errorf("create qdr-operator role failed: %v", err)
	}
	return nil
}

func (c *ContextData) setupQdrClusterRole() error {
	crole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: qdrOperatorName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err := c.Clients.KubeClient.RbacV1().ClusterRoles().Create(crole)
	if err != nil {
		return fmt.Errorf("create qdr-operator cluster role failed: %v", err)
	}
	return nil
}

func (c *ContextData) setupQdrRoleBinding() error {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: qdrOperatorName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     qdrOperatorName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      qdrOperatorName,
				Namespace: c.Namespace,
			},
		},
	}
	_, err := c.Clients.KubeClient.RbacV1().RoleBindings(c.Namespace).Create(rb)
	if err != nil {
		return fmt.Errorf("create qdr-operator role binding failed: %v", err)
	}
	return nil
}

func (c *ContextData) setupQdrClusterRoleBinding() error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: qdrOperatorName,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     qdrOperatorName,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      qdrOperatorName,
				Namespace: c.Namespace,
			},
		},
	}
	_, err := c.Clients.KubeClient.RbacV1().ClusterRoleBindings().Create(crb)
	if err != nil {
		return fmt.Errorf("create qdr-operator cluster role binding failed: %v", err)
	}
	return nil
}

func (c *ContextData) setupQdrCrd() error {
	crd := &apiextv1b1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextv1b1.CustomResourceDefinitionSpec{
			Group: "interconnectedcloud.github.io",
			Names: apiextv1b1.CustomResourceDefinitionNames{
				Kind:     "Interconnect",
				ListKind: "InterconnectList",
				Plural:   "interconnects",
				Singular: "interconnect",
			},
			Scope:   "Namespaced",
			Version: "v1alpha1",
		},
	}
	_, err := c.Clients.ExtClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return fmt.Errorf("create qdr-operator crd failed: %v", err)
	}
	return nil
}

func (c *ContextData) setupQdrDeployment() error {
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: qdrOperatorName,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": qdrOperatorName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": qdrOperatorName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: qdrOperatorName,
					Containers: []corev1.Container{
						{
							Command:         []string{qdrOperatorName},
							Name:            qdrOperatorName,
							Image:           TestContext.OperatorImage,
							ImagePullPolicy: corev1.PullAlways,
							Env: []corev1.EnvVar{
								{
									Name:      "WATCH_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
								},
								{
									Name:      "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}},
								},
								{
									Name:  "OPERATOR_NAME",
									Value: qdrOperatorName,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "metrics",
									ContainerPort: 60000,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := c.Clients.KubeClient.AppsV1().Deployments(c.Namespace).Create(dep)
	if err != nil {
		return fmt.Errorf("create qdr-operator deployment failed: %v", err)
	}
	return nil
}

// GetFirstContext returns the first entry in the ContextMap or nil if none
func (f *Framework) GetFirstContext() *ContextData {
	for _, cd := range f.ContextMap {
		return cd
	}
	return nil
}

func int32Ptr(i int32) *int32 { return &i }
