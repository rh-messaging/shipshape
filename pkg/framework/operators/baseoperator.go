package operators

import (
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	appsv1 "k8s.io/api/apps/v1"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	clientset "k8s.io/client-go/kubernetes"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/client-go/rest"
	"net/http"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

)

//All the base operator stuff goes into this class. All operator-specific things go into specific classes.
type BaseOperatorBuilder struct{}
type BaseOperator struct {
	kubeClient        *clientset.Clientset
	extClient         *apiextension.Clientset
	namespace         string
	operatorInterface interface{}
	image             string
	cdrNames          []string
	groupName         string
	operatorName      string
	apiVersion        string
	yamls             []string
	deploymentConfig appsv1.Deployment
	serviceAccount corev1.ServiceAccount
	role rbacv1.Role
	roleBinding rbacv1.RoleBinding
	crds []apiextv1b1.CustomResourceDefinition
	keepCRD bool
}

type DefinitionStruct struct {
	ApiVersion string      `json:"apiVersion"`
	Kind       string      `json:"kind"`
	Metadata   interface{} `json:"metadata"`
	Spec       interface{} `json:"spec"`
}

func (b *BaseOperatorBuilder) NewForConfig(namespace string,
	restConfig *rest.Config,
	operatorConfig OperatorConfig) (OperatorDescription, error) {
	baseOperator := &BaseOperator{}
	if kubeClient, err := clientset.NewForConfig(restConfig); err != nil {
		return nil, err
	} else {
		baseOperator.kubeClient = kubeClient
	}

	if extClient, err := apiextension.NewForConfig(restConfig); err != nil {
		return nil, err
	} else {
		baseOperator.extClient = extClient
	}
	baseOperator.namespace = namespace
	baseOperator.apiVersion = operatorConfig.ApiVersion()
	baseOperator.operatorName = operatorConfig.OperatorName()
	baseOperator.yamls = operatorConfig.YamlUrls()
	baseOperator.keepCRD = operatorConfig.KeepCRD()
	if err := baseOperator.Setup(); err != nil {
		return nil, fmt.Errorf("failed to set up operator %s: %v", operatorName, err)
	}
	return baseOperator, nil
}

func (b *BaseOperator) loadJson(url string) ([]byte, error) {
	resp, err := http.Get(url) //load yaml body from url
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
	log.Logf("setting up service account")
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

func (b *BaseOperator) setupRoleBinding(jsonObj []byte) {
	log.Logf("Setting up Role Binding")
	if err := json.Unmarshal(jsonObj, &b.roleBinding); err != nil {
		b.errorItemLoad("role binding", jsonObj, err)
	}
	if _, err := b.kubeClient.RbacV1().RoleBindings(b.namespace).Create(&b.roleBinding); err != nil {
		b.errorItemCreate("role binding", err)
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

func (b *BaseOperator) SetupYamls() error {
	for i, url := range b.yamls {

		jsonItem, err := b.loadJson(url)
		if err != nil {
			return fmt.Errorf("failed to load yaml #%i: %v", i, err)
		}
		var def DefinitionStruct
		err = json.Unmarshal(jsonItem, &def)
		if err != nil {
			return fmt.Errorf("failed to load json #%i: %v", i, err)
		}
		switch def.Kind {
		case "ServiceAccount":
			b.setupServiceAccount(jsonItem)
		case "Role":
			b.setupRole(jsonItem)
		case "RoleBinding":
			b.setupRoleBinding(jsonItem)
		case "CustomResourceDefinition":
			b.setupCRD(jsonItem)
		case "Deployment":
			if err = json.Unmarshal(jsonItem, &b.deploymentConfig); err!=nil {
				b.errorItemLoad("deployment", jsonItem, err)
			}
		default:
			log.Logf("can't find item type %s", def.Kind)
		}
	}
	return nil
}

func (b *BaseOperator) setupDeployment() error {
	if _, err := b.kubeClient.AppsV1().Deployments(b.namespace).Create(&b.deploymentConfig); err!=nil {
		b.errorItemCreate("deployment", err)
	}
	// Should only have single container in deployment yaml.
	b.image = b.deploymentConfig.Spec.Template.Spec.Containers[0].Image
	return nil
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
	log.Logf("yamls setup complete, setting up deployment")
	if err := b.setupDeployment(); err !=nil {
		return err
	}
	return nil
}

func (b *BaseOperator) TeardownEach() error {
	if (b.keepCRD) {
		return nil
	} else {
		err := b.kubeClient.CoreV1().
			ServiceAccounts(b.namespace).
			Delete(b.serviceAccount.Name, metav1.NewDeleteOptions(1))
		if err!=nil && !apierrors.IsNotFound(err){
			return err
		}
		err = b.kubeClient.RbacV1().
			Roles(b.namespace).
			Delete(b.role.Name, metav1.NewDeleteOptions(1))
		if err!=nil && !apierrors.IsNotFound(err){
			return err
		}
		err = b.kubeClient.RbacV1().
			RoleBindings(b.namespace).
			Delete(b.roleBinding.Name, metav1.NewDeleteOptions(1))
		if err!=nil && !apierrors.IsNotFound(err){
			return err
		}
		err = b.kubeClient.AppsV1().
			Deployments(b.namespace).
			Delete(b.deploymentConfig.Name, metav1.NewDeleteOptions(1))
		if err!=nil && !apierrors.IsNotFound(err){
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
		err:= b.TeardownEach()
		if err!=nil {
			return err
		}

		for _, crd := range b.crds {
			err = b.extClient.ApiextensionsV1beta1().
				CustomResourceDefinitions().Delete(
				crd.Name, metav1.NewDeleteOptions(1))
			if err!=nil && !apierrors.IsNotFound(err){
				return err
			}
		}
		return nil
	}
}


type BaseOperatorConfig struct {
	apiVersion string
	operatorName string
	yamlUrls []string
	keepCRD bool
}

func (b *BaseOperatorConfig) ApiVersion() string {
	return b.apiVersion
}

func (b *BaseOperatorConfig) OperatorName() string {
	return b.operatorName
}

func (b *BaseOperatorConfig) YamlUrls() []string {
	return b.yamlUrls
}

func (b *BaseOperatorConfig) KeepCRD() bool {
	return b.keepCRD
}