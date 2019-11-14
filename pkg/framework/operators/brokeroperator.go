package operators

import (
	"fmt"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)


type BrokerOperatorBuilder struct {}


func (b* BrokerOperatorBuilder) NewForConfig(namespace string, restConfig *rest.Config) (OperatorSetup, Error) {
	broker := &BrokerOperator {
		&BaseOperator {
            namespace: namespace,
            restConfig: restConfig,
        }
	}
}

type BrokerOperator struct {
	baseOperator BaseOperator
}

func (b* BrokerOperator) Namespace() string {
	return b.baseOperator.namespace
}

func (b* BrokerOperator) Setup() error {
	log.Logf("Setting up Service Account")
	if err := b.SetupServiceAccount(); err != nil {
		return err
	}

	log.Logf("Setting up Role")
	if err := b.SetupRole(); err != nil {
		return err
	}

	log.Logf("Setting up Cluster Role")
	if err := b.SetupClusterRole(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	log.Logf("Setting up Role Binding")
	if err := b.SetupRoleBinding(); err != nil {
		return err
	}

	log.Logf("Setting up Cluster Role Binding")
	if err := b.SetupClusterRoleBinding(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	log.Logf("Setting up CRD")
	if err := b.SetupCRD(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	} else if err != nil {
		// In case CRD already exists, do not remove on clean up (to preserve original state)
		b.keepCRD = true

	}

	log.Logf("Setting up Operator Deployment")
	if err := b.SetupDeployment(); err != nil {
		return err
	}

	return nil
}

func (b* BrokerOperator) Name() string {
    return "broker-operator"
}

func (b* BrokerOperator) SetupRole() error {
	err:= b.baseOperator.SetupRole(b.Name())
	if (err!=nil) {    
        return fmt.Errorf("create broker-operator role failed: %v", err)
    }
    return nil
}

func (b* BrokerOperator) SetupServiceAccount() error {
	err:=b.baseOperator.SetupServiceAccount(b.Name())
    if (err!=nil) {
        return fmt.Errorf("create broker-operator service account failed: %v", err)
    }
    return nil
}

func (b* BrokerOperator) SetupClusterRole() error {
    err:= b.baseOprator.SetupRole(b.Name())
    if (err!=nil){
        return fmt.Errorf("create broker-operator cluster role failed: %v", err)
    }
    return nil
}

func (b* BrokerOperator) SetupRoleBinding() error {
    err:=b.baseOperator.SetupRoleBinding(b.Name())
    if (err!=nil) {
        return fmt.Errorf("create broker-operator role binding failed: %v", err)
    }
    return nil
}

func (b* BrokerOperator) SetupClusterRoleBinding() error {
    err:= b.baseOperator.SetupClusterRoleBinding(b.Name())
    if (err!=nil) {
        return fmt.Errorf("create broker-operator cluster role binding failed: %v", err)
    }
    return nil;
}

func (b* BrokerOperator) SetupCRDs() error {
    
    //Can't have const arrays, nearest we can get is this
    var crds [...] apiextv1b1.CustomResourceDefinition := {
        apiextv1b1.CustomResourceDefinition{
        ObjectMeta: metav1.ObjectMeta {
            Name: "activemqartemisaddresses.broker.amq.io",
        },
        Spec: apiextv1b1.CustomResourceDefinitionSpec{
            Group: b.GroupName(),
            Names: &apiextv1b1.CustomResourceDefinitionNames{
                Kind: "ActiveMQArtemisAddress",
                ListKind: "ActiveMQArtemisAddressList",
                Plural: "activemqartemisaddresses",
                Singular "activemqartemisaddress":
            },
            Scope: "Namespaced",
            Version: b.APIVersion(),
        },
   }, 
        apiextv1b1.CustomResourceDefinition{
        ObjectMeta: metav1.ObjectMeta {
            Name: "activemqartemises.broker.amq.io",
        },
        Spec: apiextv1b1.CustomResourceDefinitionSpec{
            Group: b.GroupName(),
            Names: apiextv1b1.CustomResourceDefinitionNames{
                Kind: "ActiveMQArtemis",
                ListKind: "ActiveMQArtemisList",
                Plural: "activemqartemises",
                Singular "activemqartemis":
            },
            Scope: "Namespaced",
            Version: b.APIVersion(),
        },
   },
        apiextv1b1.CustomResourceDefinition{
        ObjectMeta: metav1.ObjectMeta {
            Name: "activemqartemisscaledowns.broker.amq.io",
        },
        Spec: apiextv1b1.CustomResourceDefinitionSpec{
            Group: b.GroupName(),
            Names: &apiextv1b1.CustomResourceDefinitionNames{
                Kind: "ActiveMQArtemisScaledown",
                ListKind: "ActiveMQArtemisScaledownList",
                Plural: "activemqartemisscaledowns",
                Singular "activemqartemisscaledown":
            },
            Scope: "Namespaced",
            Version: b.APIVersion(),
        },
        }
    }
   
   for i=0;i<2;i++ {
       err := b.SetupCRD(crds[i])
       if (err!=nil) {
            return fmt.Errorf("create broker-operator crd failed: %v", err)
       }
   }
   return nil
}

func (b* BrokerOperator) SetupCRD(crd apiextv1b1.CustomResourceDefinition) error {
    _, err := b.extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
    return err

}

func (b* BrokerOperator) SetupDeployment() error {
    err := b.baseOperator.SetupDeployment(Name(), Image(), Namespace())
    if (err!=nil) {
        return fmt.Errorf("setup deployment for broker-operator failed: %v", err)
    }
    return nil
}

func (b* BrokerOperator) TeardownEach() error {
    err:= b.baseOperator.TeardownEach(Name())
    if (err!=nil) {
        return fmt.Errorf("teardown each failed: %v", err)
    }
    return nil
}

func (b* BrokerOperator) Image() string {
    if (b.baseOperator.imageName == nil) {
        return "quay.io/artemiscloud/activemq-artemis-operator"
    } else {
        return b.baseOperator.imageName
    }
}

func (b* BrokerOperator) APIVersion string {
    return "v2alpha1"
}

