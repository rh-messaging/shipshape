package operators

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"github.com/rh-messaging/shipshape/pkg/framework/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlserial "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/dynamic"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type dynamicAction int

const (
	dynamicActionCreate dynamicAction = iota
	dynamicActionDelete
)

//All the base operator stuff goes into this class. All operator-specific things go into specific classes.
type BaseOperatorBuilder struct {
	yamls           [][]byte
	yamlURLs        []string
	image           string
	namespace       string
	restConfig      *rest.Config
	rawConfig       *clientcmdapi.Config
	operatorName    string
	keepCdrs        bool
	apiVersion      string
	customCommand   string
	finalized       bool
	crdsPrepared    bool
	globalNamespace bool
}

type BaseOperator struct {
	restConfig        *rest.Config
	rawConfig         *clientcmdapi.Config
	kubeClient        *clientset.Clientset
	extClient         *apiextension.Clientset
	context           string
	namespace         string
	operatorInterface interface{}
	image             string
	cdrNames          []string
	groupName         string
	operatorName      string
	apiVersion        string
	yamlURLs          []string
	yamls             [][]byte
	customCommand     string
	deploymentConfig  appsv1.Deployment
	serviceAccount    corev1.ServiceAccount
	configMap         corev1.ConfigMap
	role              rbacv1.Role
	cRole             rbacv1.ClusterRole
	roleBinding       rbacv1.RoleBinding
	cRoleBinding      rbacv1.ClusterRoleBinding
	crds              [][]byte
	keepCRD           bool
	crdsPrepared      bool
	globalNamespace   bool
}

type DefinitionStruct struct {
	ApiVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   interface{} `json:"metadata"`
	Spec       interface{} `json:"spec"`
}

func (b *BaseOperatorBuilder) NewBuilder(restConfig *rest.Config, rawConfig *clientcmdapi.Config) OperatorSetupBuilder {
	b.restConfig = restConfig
	b.rawConfig = rawConfig
	return b
}

func (b *BaseOperatorBuilder) WithNamespace(namespace string) OperatorSetupBuilder {
	if !b.finalized {
		b.namespace = namespace
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

func (b *BaseOperatorBuilder) OperatorType() OperatorType {
	// Delegate to concrete implementations
	panic("(don't) implement me")
}

func (b *BaseOperatorBuilder) WithGlobalNamespace() OperatorSetupBuilder {
	if !b.finalized {
		b.globalNamespace = true
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

func (b *BaseOperatorBuilder) WithImage(image string) OperatorSetupBuilder {
	if !b.finalized {
		b.image = image
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

func (b *BaseOperatorBuilder) WithCommand(command string) OperatorSetupBuilder {
	if !b.finalized {
		b.customCommand = command
	}
	return b
}

func (b *BaseOperatorBuilder) WithYamlURLs(yamls []string) OperatorSetupBuilder {
	if !b.finalized {
		b.yamlURLs = yamls
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

// Pass loaded yamls objects instead of URLs
func (b *BaseOperatorBuilder) WithYamls(yamls [][]byte) OperatorSetupBuilder {
	b.yamls = yamls
	return b
}

func (b *BaseOperatorBuilder) AddYamlURL(yaml string) OperatorSetupBuilder {
	if !b.finalized {
		b.yamlURLs = append(b.yamlURLs, yaml)
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

func (b *BaseOperatorBuilder) WithOperatorName(name string) OperatorSetupBuilder {
	if !b.finalized {
		b.operatorName = name
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

func (b *BaseOperatorBuilder) KeepCdr(keepCdrs bool) OperatorSetupBuilder {
	if !b.finalized {
		b.keepCdrs = keepCdrs
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

// When CRDs are already created on the cluster in question, we don't need to do anything with them.
func (b *BaseOperatorBuilder) SetAdminUnavailable() OperatorSetupBuilder {
	if !b.finalized {
		b.crdsPrepared = true
		// If they are pre-defined, we don't need to clean them up either
		b.keepCdrs = true
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

func (b *BaseOperatorBuilder) WithApiVersion(apiVersion string) OperatorSetupBuilder {
	if !b.finalized {
		b.apiVersion = apiVersion
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
}

// Allows for custom named operators to exist
func (b *BaseOperatorBuilder) SetOperatorName(operatorName string) OperatorSetupBuilder {
	b.operatorName = operatorName
	return b
}

func (b *BaseOperatorBuilder) Finalize() *BaseOperatorBuilder {
	b.finalized = true
	return b
}

func (b *BaseOperatorBuilder) Build() (OperatorSetup, error) {
	baseOperator := &BaseOperator{}
	if kubeClient, err := clientset.NewForConfig(b.restConfig); err != nil {
		return nil, err
	} else {
		baseOperator.kubeClient = kubeClient
	}

	if extClient, err := apiextension.NewForConfig(b.restConfig); err != nil {
		return nil, err
	} else {
		baseOperator.extClient = extClient
	}
	baseOperator.namespace = b.namespace
	baseOperator.apiVersion = b.apiVersion
	baseOperator.operatorName = b.operatorName
	baseOperator.yamlURLs = b.yamlURLs
	baseOperator.yamls = b.yamls
	baseOperator.keepCRD = b.keepCdrs
	baseOperator.customCommand = b.customCommand
	baseOperator.crdsPrepared = b.crdsPrepared
	baseOperator.globalNamespace = b.globalNamespace
	if err := baseOperator.Setup(); err != nil {
		return nil, fmt.Errorf("failed to set up operator %s: %v", baseOperator.operatorName, err)
	}

	log.Logf("Built operator builder")
	return baseOperator, nil
}

func (b *BaseOperator) InitFromBaseOperatorBuilder(builder *BaseOperatorBuilder) error {
	b.restConfig = builder.restConfig
	b.image = builder.image
	b.namespace = builder.namespace
	b.apiVersion = builder.apiVersion
	b.operatorName = builder.operatorName
	b.yamls = builder.yamls
	b.keepCRD = builder.keepCdrs
	b.crdsPrepared = builder.crdsPrepared
	b.globalNamespace = builder.globalNamespace

	// Initialize clients
	if kubeClient, err := clientset.NewForConfig(b.restConfig); err != nil {
		return err
	} else {
		b.kubeClient = kubeClient
	}

	if extClient, err := apiextension.NewForConfig(b.restConfig); err != nil {
		return err
	} else {
		b.extClient = extClient
	}

	return nil
}

func (b *BaseOperator) loadJson(url string) ([]byte, error) {
	resp, err := http.Get(url) //load yamls body from url
	if err != nil {
		log.Logf("error during loading %s: %v", url, err)
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Logf("error during loading %s: %v", url, err)
		return nil, err
	}
	jsonBody, err := yaml.YAMLToJSON(body)
	if err != nil {
		log.Logf("error during parsing %s: %v", url, err)
		return nil, err
	}
	return jsonBody, nil
}

func (b *BaseOperator) errorItemLoad(failedType string, jsonObj []byte, parentError error) {
	panic(fmt.Errorf("failed to load %s from json definition: %s %v", failedType, jsonObj, parentError))
}

func (b *BaseOperator) errorItemCreate(failedType string, parentError error) {
	if strings.Contains(parentError.Error(), "already exists") {
		//If any items already exist, don't remove those in teardown
		b.keepCRD = true
	} else {
		panic(fmt.Errorf("failed to create %s : %v", failedType, parentError))
	}
}

func (b *BaseOperator) setupServiceAccount(jsonObj []byte) {
	log.Logf("setting up service account (ns: %s)", b.namespace)
	if err := json.Unmarshal(jsonObj, &b.serviceAccount); err != nil {
		b.errorItemLoad("service account", jsonObj, err)
	}
	if _, err := b.kubeClient.CoreV1().ServiceAccounts(b.namespace).Create(&b.serviceAccount); err != nil {
		b.errorItemCreate("service account", err)
	}

}

func (b *BaseOperator) setupRole(jsonObj []byte) {
	log.Logf("Setting up Role")

	if err := json.Unmarshal(jsonObj, &b.role); err != nil {
		b.errorItemLoad("role", jsonObj, err)
	}
	for _, item := range b.role.Rules {
		log.Logf("Rule concerning %v is being created", item.Resources)
	}
	if _, err := b.kubeClient.RbacV1().Roles(b.namespace).Create(&b.role); err != nil {
		b.errorItemCreate("role", err)
	}
}

func (b *BaseOperator) setupClusterRole(jsonObj []byte) {
	log.Logf("Setting up cluster role")
	if err := json.Unmarshal(jsonObj, &b.cRole); err != nil {
		b.errorItemLoad("cluster role", jsonObj, err)
	}
	// Ignore errors if cluster level resource already exists
	if _, err := b.kubeClient.RbacV1().ClusterRoles().Create(&b.cRole); err != nil {
		b.errorItemCreate("cluster role", err)
	}
}

func (b *BaseOperator) setupRoleBinding(jsonObj []byte) {
	log.Logf("Setting up Role Binding")
	if err := json.Unmarshal(jsonObj, &b.roleBinding); err != nil {
		b.errorItemLoad("role binding", jsonObj, err)
	}
	b.roleBinding.Name = "rolebinding-" + util.String(8) //silly.
	//b.roleBinding.Subjects[0].Namespace = "openshift-operators" //hardcoded as cluster-wide operator install is hardcoded to openshift-operators
	if _, err := b.kubeClient.RbacV1().RoleBindings(b.namespace).Create(&b.roleBinding); err != nil {
		b.errorItemCreate("role binding", err)
	}
}

func (b *BaseOperator) setupClusterRoleBinding(jsonObj []byte) {
	log.Logf("Setting up Cluster Role Binding")
	if err := json.Unmarshal(jsonObj, &b.cRoleBinding); err != nil {
		b.errorItemLoad("cluster role binding", jsonObj, err)
	}
	if _, err := b.kubeClient.RbacV1().ClusterRoleBindings().Create(&b.cRoleBinding); err != nil {
		b.errorItemCreate("cluster role binding", err)
	}
}

func (b *BaseOperator) setupConfigMap(jsonObj []byte) {
	log.Logf("Setting up ConfigMap")
	if err := json.Unmarshal(jsonObj, &b.configMap); err != nil {
		b.errorItemLoad("cluster role binding", jsonObj, err)
	}
	if _, err := b.kubeClient.CoreV1().ConfigMaps(b.Namespace()).Create(&b.configMap); err != nil {
		b.errorItemCreate("cluster role binding", err)
	}
}

func (b *BaseOperator) setupCRD(json []byte) {
	if !b.crdsPrepared {
		log.Logf("Setting up CRD")
		yaml, err := yaml.JSONToYAML(json)
		if err != nil {
			b.errorItemLoad("CRD (json to yaml)", json, err)
		}
		if err := b.CreateResourcesFromYAMLBytes(yaml); err != nil && !strings.Contains(err.Error(), "already exists") {
			b.errorItemLoad("CRD", yaml, err)
		}
		b.crds = append(b.crds, yaml)
	} else {
		log.Logf("not setting up CRDs due to configuration flag")
	}
}

func (b *BaseOperator) setupYamlsFromUrls() error {
	for _, url := range b.yamlURLs {
		jsonItem, err := b.loadJson(url)
		if err != nil {
			return err
		}
		err = b.createKubeObject(jsonItem)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BaseOperator) createKubeObject(jsonItem []byte) error {
	var def DefinitionStruct
	err := json.Unmarshal(jsonItem, &def)
	if err != nil {
		return err
	}

	switch def.Kind {
	case "ServiceAccount":
		b.setupServiceAccount(jsonItem)
	case "Role":
		b.setupRole(jsonItem)
	case "ClusterRole":
		b.setupClusterRole(jsonItem)
	case "RoleBinding":
		b.setupRoleBinding(jsonItem)
	case "ClusterRoleBinding":
		b.setupClusterRoleBinding(jsonItem)
	case "CustomResourceDefinition":
		b.setupCRD(jsonItem)
	case "Deployment":
		b.setupDeployment(jsonItem)
	case "ConfigMap":

	default:
		return fmt.Errorf("can't find item type %s", def.Kind)
	}
	return nil
}

func (b *BaseOperator) setupPreparedYamls() error {
	for _, item := range b.yamls {
		jsonItem, err := yaml.YAMLToJSON(item)
		if err != nil {
			return err
		}
		err = b.createKubeObject(jsonItem)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *BaseOperator) SetupYamls() error {
	log.Logf("yamls being set up (normal)")
	if b.yamlURLs != nil {
		return b.setupYamlsFromUrls()
	} else if b.yamls != nil {
		return b.setupPreparedYamls()
	} else {
		return fmt.Errorf("yaml definitions were not supplied to operator builder")
	}
}

func (b *BaseOperator) setupDeployment(jsonItem []byte) {
	log.Logf("Setting up Deployment (base)")
	log.Logf("Namespace: %s", b.namespace)
	if err := json.Unmarshal(jsonItem, &b.deploymentConfig); err != nil {
		b.errorItemLoad("deployment", jsonItem, err)
	}
	if b.image != "" {
		//Customize the spec if that is requested
		b.deploymentConfig.Spec.Template.Spec.Containers[0].Image = b.image
	}
	if b.customCommand != "" {
		b.deploymentConfig.Spec.Template.Spec.Containers[0].Command = []string{b.customCommand}
	}
	if b.operatorName != "" {
		//b.deploymentConfig.Spec.Template.Spec.Containers[0].Name = b.operatorName
		b.deploymentConfig.ObjectMeta.Name = b.operatorName
	}

	if b.globalNamespace {
		log.Logf("Patching env for global operator support")
		watchNamespaces := &corev1.EnvVar{
			Name:      "WATCH_NAMESPACE",
			Value:     "",
			ValueFrom: nil,
		}
		b.deploymentConfig.Spec.Template.Spec.Containers[0].Env = append(b.deploymentConfig.Spec.Template.Spec.Containers[0].Env, *watchNamespaces)
		/* for _, item := range b.deploymentConfig.Spec.Template.Spec.Containers[0].Env {
			if item.Name == "WATCH_NAMESPACE" {
				log.Logf("Value: %s, ValueFrom: %s", item.Value, item.ValueFrom)
			}
		} */ // - may be better to overwrite value in env instead of appending new envvar, unssure yet
	}

	if err := b.CreateDeployment(); err != nil {
		b.errorItemCreate("deployment", err)
	}
}

func (b *BaseOperator) Namespace() string {
	return b.namespace
}

func (b *BaseOperator) Interface() interface{} {
	return b.operatorInterface
}

func (b *BaseOperator) Image() string {
	return b.image
}

func (b *BaseOperator) Name() string {
	return b.operatorName
}

func (b *BaseOperator) CRDNames() []string {
	return b.cdrNames
}

func (b *BaseOperator) GroupName() string {
	return b.groupName
}

func (b *BaseOperator) APIVersion() string {
	return b.apiVersion
}

func (b *BaseOperator) Setup() error {
	if err := b.SetupYamls(); err != nil {
		return err
	}
	return nil
}

func (b *BaseOperator) GetDeployment() (*appsv1.Deployment, error) {
	return b.kubeClient.AppsV1().Deployments(b.namespace).Get(b.deploymentConfig.Name, metav1.GetOptions{})
}

func (b *BaseOperator) UpdateDeployment(deployment *appsv1.Deployment) error {
	_, err := b.kubeClient.AppsV1().Deployments(b.namespace).Update(deployment)
	if err != nil {
		return err
	}
	return nil
}

func (b *BaseOperator) DeleteOperator() error {
	err := b.kubeClient.AppsV1().Deployments(b.namespace).Delete(b.deploymentConfig.Name, metav1.NewDeleteOptions(1))
	if err != nil {
		return err
	}
	return nil
}

func (b *BaseOperator) CreateDeployment() error {
	_, err := b.kubeClient.AppsV1().Deployments(b.namespace).Create(&b.deploymentConfig)
	if err != nil {
		return err
	}
	return nil
}

func (b *BaseOperator) TeardownEach() error {
	if b.keepCRD {
		return nil
	} else {
		err := b.kubeClient.CoreV1().
			ServiceAccounts(b.namespace).
			Delete(b.serviceAccount.Name, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = b.kubeClient.RbacV1().
			Roles(b.namespace).
			Delete(b.role.Name, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = b.kubeClient.RbacV1().
			RoleBindings(b.namespace).
			Delete(b.roleBinding.Name, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = b.kubeClient.AppsV1().
			Deployments(b.namespace).
			Delete(b.deploymentConfig.Name, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		log.Logf("%s teradown namespace succesful", b.namespace)
		return nil
	}
}

func (b *BaseOperator) TeardownSuite() error {
	if b.keepCRD {
		return nil
	} else {
		err := b.TeardownEach()
		if err != nil {
			return err
		}

		for _, crd := range b.crds {
			err = b.DeleteResourcesFromYAMLBytes(crd)
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}
}
func (b *BaseOperator) manageResourcesFromYAMLBytes(action dynamicAction, yamlData []byte) error {
	var err error

	// Creating a dynamic client
	dynClient, err := dynamic.NewForConfig(b.restConfig)
	if err != nil {
		return fmt.Errorf("error creating a dynamic k8s client: %s", err)
	}
	// Read YAML file removing all comments and blank lines
	// otherwise yamlDecoder does not work
	yamlBuffer, err := readYAMLIgnoringComments(yamlData)
	if err != nil {
		return err
	}

	yamlDecoder := yamlutil.NewYAMLOrJSONDecoder(bufio.NewReader(&yamlBuffer), 1024)

	for {
		// Decoding raw object from yaml
		var rawObj runtime.RawExtension
		if err = yamlDecoder.Decode(&rawObj); err != nil {
			if err != io.EOF {
				return fmt.Errorf("error decoding yaml: %s", err)
			}
			return nil
		}
		obj, gvk, err := yamlserial.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
		if err != nil {
			fmt.Println("unable to create gvk from raw data")
			return err
		}
		// Converts unstructured object into a map[string]interface{}
		unstructureMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return fmt.Errorf("unable to convert to unstructured map: %s", err)
		}
		// Create a generic unstructured object from map
		unstructuredObj := &unstructured.Unstructured{Object: unstructureMap}
		// Getting API Group Resources using discovery client
		gr, err := restmapper.GetAPIGroupResources(b.kubeClient.Discovery())
		if err != nil {
			return fmt.Errorf("error getting APIGroupResources: %s", err)
		}
		// Unstructured object mapper for the provided group and kind
		mapper := restmapper.NewDiscoveryRESTMapper(gr)
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("error obtaining mapping for: %s - %s", gvk.GroupVersion().String(), err)
		}
		// Dynamic resource handler
		var k8sResource dynamic.ResourceInterface
		if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			if unstructuredObj.GetNamespace() == "" {
				unstructuredObj.SetNamespace(b.Namespace())
			}
			k8sResource = dynClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
		} else {
			k8sResource = dynClient.Resource(mapping.Resource)
		}
		// Creating the dynamic resource
		errorAction := ""
		switch action {
		case dynamicActionCreate:
			_, err = k8sResource.Create(unstructuredObj, metav1.CreateOptions{})
			errorAction = "creating"
		case dynamicActionDelete:
			err = k8sResource.Delete(unstructuredObj.GetName(), &metav1.DeleteOptions{})
			errorAction = "deleting"
		}
		if err != nil {
			return fmt.Errorf("error %s resource [group=%s - kind=%s] - %s", errorAction, gvk.Group, gvk.Kind, err)
		}
	}

	return err
}

func (b *BaseOperator) CreateResourcesFromYAMLBytes(yamlData []byte) error {
	return b.manageResourcesFromYAMLBytes(dynamicActionCreate, yamlData)
}

func (b *BaseOperator) DeleteResourcesFromYAMLBytes(yamlData []byte) error {
	return b.manageResourcesFromYAMLBytes(dynamicActionDelete, yamlData)
}

// CreateResourcesFromYAML creates all resources from the provided YAML file
// or URL using an initialized VanClient instance.
func (b *BaseOperator) CreateResourcesFromYAML(fileOrUrl string) error {
	var yamlData []byte
	var err error

	// Load YAML from an http/https url or local file
	isUrl, _ := regexp.Compile("http[s]*://")
	if isUrl.MatchString(fileOrUrl) {
		yamlData, err = readYAMLFromUrl(fileOrUrl)
		if err != nil {
			return err
		}
	} else {
		// Read YAML file
		yamlData, err = ioutil.ReadFile(fileOrUrl)
		if err != nil {
			return fmt.Errorf("error reading yaml file: %s", err)
		}
	}

	return b.CreateResourcesFromYAMLBytes(yamlData)
}

// readYAMLFromUrl returns the content for the provided url
func readYAMLFromUrl(url string) ([]byte, error) {
	var yamlData []byte
	// Load from URL if url provided
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error loading yaml from url [%s]: %s", url, err)
	}
	defer resp.Body.Close()
	yamlData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading yaml from url [%s]: %s", url, err)
	}
	return yamlData, nil
}

// readYAMLIgnoringComments returns a bytes.Buffer that contains the
// content from the loaded yamlData removing all comments and empty lines
// that exists before the beginning of the yaml data. This is needed
// for the k8s yaml decoder to properly identify if content is a YAML
// or JSON.
func readYAMLIgnoringComments(yamlData []byte) (bytes.Buffer, error) {
	var yamlNoComments bytes.Buffer

	// We must strip all comments and blank lines from yaml file
	// otherwise the k8s yaml decoder might fail
	yamlBytesReader := bytes.NewReader(yamlData)
	yamlBufReader := bufio.NewReader(yamlBytesReader)
	yamlBufWriter := bufio.NewWriter(&yamlNoComments)

	// Regexp to exclude empty lines and lines with comments only
	// till beginning of yaml content (otherwise yaml decoder
	// won't be able to identify whether it is JSON or YAML).
	ignoreRegexp, _ := regexp.Compile("^\\s*(#|$)")
	yamlStarted := false
	for {
		line, err := yamlBufReader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if !yamlStarted && ignoreRegexp.MatchString(line) {
			continue
		}
		yamlStarted = true
		_, _ = yamlBufWriter.WriteString(line)
	}
	_ = yamlBufWriter.Flush()
	return yamlNoComments, nil
}
