package operators

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	clientset "k8s.io/client-go/kubernetes"

	"github.com/rh-messaging/shipshape/pkg/framework/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"net/http"
	"strings"
)

//All the base operator stuff goes into this class. All operator-specific things go into specific classes.
type BaseOperatorBuilder struct {
	yamls         [][]byte
	yamlURLs      []string
	image         string
	namespace     string
	restConfig    *rest.Config
	rawConfig     *clientcmdapi.Config
	operatorName  string
	keepCdrs      bool
	apiVersion    string
	customCommand string
	finalized     bool
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
	role              rbacv1.Role
	cRole             rbacv1.ClusterRole
	roleBinding       rbacv1.RoleBinding
	cRoleBinding      rbacv1.ClusterRoleBinding
	crds              []apiextv1b1.CustomResourceDefinition
	keepCRD           bool
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
	panic("implement me")
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

func (b *BaseOperatorBuilder) WithApiVersion(apiVersion string) OperatorSetupBuilder {
	if !b.finalized {
		b.apiVersion = apiVersion
		return b
	} else {
		panic(fmt.Errorf("can't edit operator builder post-finalization"))
	}
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
	if err := baseOperator.Setup(); err != nil {
		return nil, fmt.Errorf("failed to set up operator %s: %v", baseOperator.operatorName, err)
	}
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
	if _, err := b.kubeClient.RbacV1().Roles(b.namespace).Create(&b.role); err != nil {
		b.errorItemCreate("role", err)
	}
}

func (b *BaseOperator) setupClusterRole(jsonObj []byte) {
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

func (b *BaseOperator) setupCRD(jsonObj []byte) {
	log.Logf("Setting up CRD")
	var CRD apiextv1b1.CustomResourceDefinition
	if err := json.Unmarshal(jsonObj, &CRD); err != nil {
		b.errorItemLoad("CRD", jsonObj, err)
	}
	if _, err := b.extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&CRD); err != nil {
		b.errorItemCreate("CRD", err)
	}
	b.crds = append(b.crds, CRD)
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
	if b.yamlURLs != nil {
		return b.setupYamlsFromUrls()
	} else if b.yamls != nil {
		return b.setupPreparedYamls()
	} else {
		return fmt.Errorf("yaml definitions were not supplied to operator builder")
	}
}

func (b *BaseOperator) setupDeployment(jsonItem []byte) {
	log.Logf("Setting up Deployment")
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
	if _, err := b.kubeClient.AppsV1().Deployments(b.namespace).Create(&b.deploymentConfig); err != nil {
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
	log.Logf("setting up yamls: %v", b.yamls)
	if err := b.SetupYamls(); err != nil {
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
			err = b.extClient.ApiextensionsV1beta1().
				CustomResourceDefinitions().Delete(
				crd.Name, metav1.NewDeleteOptions(1))
			if err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
		return nil
	}
}
