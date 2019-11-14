package operators

import (
	"fmt"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
)

type BrokerOperatorBuilder struct{}

func (b *BrokerOperatorBuilder) NewForConfig(namespace string, restConfig *rest.Config) (OperatorSetup, error) {
	broker := &BrokerOperator{
		BaseOperator{
			namespace:  namespace,
			restConfig: restConfig,
		},
	}

	if client, err := clientset.NewForConfig(restConfig); err != nil {
		return broker, err
	} else {
		broker.baseOperator.kubeClient = client
	}

	if client, err := apiextension.NewForConfig(restConfig); err != nil {
		return broker, err
	} else {
		broker.baseOperator.extClient = client
	}

	return broker, nil
}

type BrokerOperator struct {
	baseOperator BaseOperator
}

func (b BrokerOperator) Interface() interface{} {
	return nil //broker doesn't have this kind of interface
}

func (b BrokerOperator) CRDName() string {
	//we have multiple CRDs so can't have it this way..
	return ""
}

func (b BrokerOperator) GroupName() string {
	return "broker.amq.io"
}

func (b *BrokerOperator) APIVersion() string {
	return "v2alpha1"
}

func (b *BrokerOperator) Namespace() string {
	return b.baseOperator.namespace
}

func (b *BrokerOperator) Setup() error {
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
	if err := b.SetupCRDs(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	} else if err != nil {
		// In case CRD already exists, do not remove on clean up (to preserve original state)
		b.baseOperator.keepCRD = true
	}
	log.Logf("Setting up Operator Deployment")
	if err := b.SetupDeployment(); err != nil {
		return err
	}

	return nil
}

func (b *BrokerOperator) Name() string {
	return "broker-operator"
}

func (b *BrokerOperator) SetupRole() error {
	err := b.baseOperator.SetupRole(b.Name())
	if (err != nil) {
		return fmt.Errorf("create broker-operator role failed: %v", err)
	}
	return nil
}

func (b *BrokerOperator) SetupServiceAccount() error {
	err := b.baseOperator.SetupServiceAccount(b.Name())
	if (err != nil) {
		return fmt.Errorf("create broker-operator service account failed: %v", err)
	}
	return nil
}

func (b *BrokerOperator) SetupClusterRole() error {
	err := b.baseOperator.SetupRole(b.Name())
	if (err != nil) {
		return fmt.Errorf("create broker-operator cluster role failed: %v", err)
	}
	return nil
}

func (b *BrokerOperator) SetupRoleBinding() error {
	err := b.baseOperator.SetupRoleBinding(b.Name())
	if (err != nil) {
		return fmt.Errorf("create broker-operator role binding failed: %v", err)
	}
	return nil
}

func (b *BrokerOperator) SetupClusterRoleBinding() error {
	err := b.baseOperator.SetupClusterRoleBinding(b.Name())
	if (err != nil) {
		return fmt.Errorf("create broker-operator cluster role binding failed: %v", err)
	}
	return nil;
}

func (b *BrokerOperator) SetupCRDs() error {

	//Can't have const arrays, nearest we can get is this
	var crds = [...]apiextv1b1.CustomResourceDefinition {
			apiextv1b1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: "activemqartemisaddresses.broker.amq.io",
			},
			Spec: apiextv1b1.CustomResourceDefinitionSpec{
				Group: b.GroupName(),
				Names: apiextv1b1.CustomResourceDefinitionNames{
					Kind:     "ActiveMQArtemisAddress",
					ListKind: "ActiveMQArtemisAddressList",
					Plural:   "activemqartemisaddresses",
					Singular: "activemqartemisaddress",
				},
				Scope:   "Namespaced",
				Version: b.APIVersion(),
			},
		},
			apiextv1b1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "activemqartemises.broker.amq.io",
				},
				Spec: apiextv1b1.CustomResourceDefinitionSpec{
					Group: b.GroupName(),
					Names: apiextv1b1.CustomResourceDefinitionNames{
						Kind:     "ActiveMQArtemis",
						ListKind: "ActiveMQArtemisList",
						Plural:   "activemqartemises",
						Singular: "activemqartemis",
					},
					Scope:   "Namespaced",
					Version: b.APIVersion(),
				},
			},
			apiextv1b1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "activemqartemisscaledowns.broker.amq.io",
				},
				Spec: apiextv1b1.CustomResourceDefinitionSpec{
					Group: b.GroupName(),
					Names: apiextv1b1.CustomResourceDefinitionNames{
						Kind:     "ActiveMQArtemisScaledown",
						ListKind: "ActiveMQArtemisScaledownList",
						Plural:   "activemqartemisscaledowns",
						Singular: "activemqartemisscaledown",
					},
					Scope:   "Namespaced",
					Version: b.APIVersion(),
				},
			},}

	for i := 0; i < 2; i++ {
		err := b.SetupCRD(&crds[i])
		if (err != nil) {
			return fmt.Errorf("create broker-operator crd failed: %v", err)
		}
	}
	return nil
}

func (b *BrokerOperator) SetupCRD(crd *apiextv1b1.CustomResourceDefinition) error {
	_, err := b.baseOperator.extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	return err

}

func (b *BrokerOperator) SetupDeployment() error {
	err := b.baseOperator.SetupDeployment(b.Name(), b.Image())
	if (err != nil) {
		return fmt.Errorf("setup deployment for broker-operator failed: %v", err)
	}
	return nil
}

func (b *BrokerOperator) TeardownEach() error {
	err := b.baseOperator.TeardownEach(b.Name())
	if (err != nil) {
		return fmt.Errorf("teardown each failed: %v", err)
	}
	return nil
}

func (b BrokerOperator) TeardownSuite() error {
	err := b.baseOperator.TeardownSuite(b.Name())
	if (err != nil) {
		return fmt.Errorf("failure during suite teardown: %v", err)
	}
	//remove CRDs
	return nil
}

func (b *BrokerOperator) Image() string {
	if (b.baseOperator.imageName == "") {
		return "quay.io/artemiscloud/activemq-artemis-operator"
	} else {
		return b.baseOperator.imageName
	}
}
