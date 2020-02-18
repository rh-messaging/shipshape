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
	"github.com/rh-messaging/shipshape/pkg/framework/operators"
	"k8s.io/client-go/rest"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	routev1 "github.com/openshift/client-go/route/clientset/versioned"
	e2elog "github.com/rh-messaging/shipshape/pkg/framework/log"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

var (
	RetryInterval        = time.Second * 5
	Timeout              = time.Second * 600
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 5
	RestConfig rest.Config

)

type ClientSet struct {
	KubeClient clientset.Interface
	ExtClient  apiextension.Interface
	DynClient  dynamic.Interface
	OcpClient  ocpClient
}

type ocpClient struct {
	RoutesClient *routev1.Clientset
}

func contains(target operators.OperatorType, collection []operators.OperatorType) bool {
	for _, a := range collection {
		if target == a {
			return true
		}
	}
	return false
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
	OperatorMap        map[operators.OperatorType]operators.OperatorSetup
	isOpenShift        *bool
}

type Framework struct {
	BaseName string

	// Map that ties clients and namespaces for each available context
	ContextMap map[string]*ContextData

	SkipNamespaceCreation bool // Whether to skip creating a namespace
	cleanupHandleEach     CleanupActionHandle
	cleanupHandleSuite    CleanupActionHandle
	afterEachDone         bool
	builders              []operators.OperatorSetupBuilder
}

// Framework Builder type
type Builder struct {
	f        *Framework
	contexts []string
}

// Helper for building frameworks with possible customizations
func NewFrameworkBuilder(baseName string) Builder {
	// In case no contexts available
	if len(TestContext.GetContexts()) == 0 {
		panic("No contexts available. Unable to create an instance of the Shipshape Framework.")
	}

	b := Builder{
		f: &Framework{
			BaseName: baseName,
			ContextMap: make(map[string]*ContextData),
		},
		contexts: []string{TestContext.GetContexts()[0]},
	}
	return b
}

// Customize contexts to use (default is the current-context only)
func (b Builder) WithContexts(contexts ...string) Builder {
	b.contexts = contexts
	return b
}

// Customize builders, by default when "BeforeEach" runs, the Framework iterates
// through all supported operators (from SupportedOperators map) and initializes
// all the default builder instances.
func (b Builder) WithBuilders(builders ...operators.OperatorSetupBuilder) Builder {
	b.f.SetOperatorBuilders(builders...)
	return b
}

// Generates and initialize the Framework
func (b Builder) Build() *Framework {
	// Initialize restConfig and kube clients for each provided context
	b.f.BeforeEach(b.contexts...)
	return b.f
}

// Defines a custom set of builders for the given Framework instance
func (f *Framework) SetOperatorBuilders(builders ...operators.OperatorSetupBuilder) {
	f.builders = builders
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
		RestConfig, err := clientConfig.ClientConfig()
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Create the client instances
		kubeClient, err := clientset.NewForConfig(RestConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		extClient, err := apiextension.NewForConfig(RestConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		dynClient, err := dynamic.NewForConfig(RestConfig)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())

		// Initilizing the ClientSet for context
		clients = ClientSet{
			KubeClient: kubeClient,
			ExtClient:  extClient,
			DynClient:  dynClient,
		}

		// Generating the namespace on provided contexts
		ginkgo.By(fmt.Sprintf("Building namespace api objects, basename %s", f.BaseName))
		// Keep original label for now (maybe we can remove or rename later)
		var namespace *corev1.Namespace
		if !f.SkipNamespaceCreation {
			namespace = generateNamespace(kubeClient, f.BaseName, namespaceLabels)
		}
		gomega.Expect(namespace).NotTo(gomega.BeNil())

		// Verify if Cert Manager is installed
		_, err = extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get("issuers.certmanager.k8s.io", metav1.GetOptions{})
		certManagerPresent := false
		if err == nil {
			certManagerPresent = true
		} else if _, err = extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Get("issuers.cert-manager.io", metav1.GetOptions{}); err == nil {
			certManagerPresent = true
		}

		// Initializing the context
		ctx := &ContextData{
			Id:                 context,
			Namespace:          namespace.GetName(),
			UniqueName:         namespace.GetName(),
			Clients:            clients,
			CertManagerPresent: certManagerPresent,
		}
		f.ContextMap[context] = ctx

		// OpenShift specific initialization
		if ctx.IsOpenShift() {
			ctx.Clients.OcpClient.RoutesClient, err = routev1.NewForConfig(RestConfig)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
		}

		// Initializing needed operators on given context
		ctx.OperatorMap = map[operators.OperatorType]operators.OperatorSetup{}
		if f.builders == nil || len(f.builders) == 0 {
			// populate builders with default values
			for _, builder := range operators.SupportedOperators {
				f.builders = append(f.builders, builder)
			}
		} else {
			log.Logf("CUSTOM BUILDERS PROVIDED")
		}
		for _, builder := range f.builders {
			builder.NewBuilder(RestConfig)
			builder.WithNamespace(namespace.GetName())
			operator, err := builder.Build()
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			ctx.OperatorMap[builder.OperatorType()] = operator
		}

		if !f.SkipNamespaceCreation {
			ctx.AddNamespacesToDelete(namespace)
		}
	}

	// setup the operators
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

	// Iterate through all contexts and deleting namespace related resources
	for _, contextData := range f.ContextMap {
		for _, operator := range contextData.OperatorMap {
			err := operator.TeardownEach()
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to teardown [each] operator [%s]: %v", operator.Name(), err)
			}
			log.Logf("%s teardown namespace [%s] successful", operator.Name(), contextData.Namespace)
		}
	}

	return nil
}

func (f *Framework) TeardownSuite() error {

	// Iterate through all contexts
	for _, contextData := range f.ContextMap {
		for _, operator := range contextData.OperatorMap {
			err := operator.TeardownSuite()
			if err != nil && !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete operator [%s] from namespace [%s]: %v", operator.Name(), contextData.Namespace, err)
			}
			log.Logf("%s teardown suite successful on %s", operator.Name(), contextData.Namespace)
		}
	}

	return nil
}

func (f *Framework) Setup() error {

	for _, ctxData := range f.ContextMap {
		for _, operator := range ctxData.OperatorMap {
			err := operator.Setup()
			if err != nil {
				return fmt.Errorf("failed to setup %s: %v", operator.Name(), err)
			}
			err = WaitForDeployment(ctxData.Clients.KubeClient, ctxData.Namespace, operator.Name(), 1, RetryInterval, Timeout)
			if err != nil {
				return fmt.Errorf("failed to wait for %s: %v", operator.Name(), err)
			}
		}
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

func (c *ContextData) IsOpenShift() bool {
	if c.isOpenShift != nil {
		return *c.isOpenShift
	}

	result := false
	apiList, err := c.Clients.KubeClient.Discovery().ServerGroups()
	if err != nil {
		e2elog.Failf("Error in getting ServerGroups from discovery client, returning false")
		result = false
		c.isOpenShift = &result
		return result
	}

	for _, v := range apiList.Groups {
		if v.Name == "route.openshift.io" {
			e2elog.Logf("OpenShift route detected in api groups, returning true")
			result = true
			c.isOpenShift = &result
			return result
		}
	}

	e2elog.Logf("OpenShift route not found in groups, returning false")
	result = false
	c.isOpenShift = &result
	return result
}

func Int32Ptr(i int32) *int32 { return &i }
