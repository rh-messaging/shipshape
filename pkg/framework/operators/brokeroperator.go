package operators

import (
	"fmt"

	brokerclientset "github.com/artemiscloud/activemq-artemis-operator/pkg/client/clientset/versioned"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reusing BaseOperatorBuilder implementation and adding
// the "abstract" method Build() and OperatorType()
type BrokerOperatorBuilder struct {
	BaseOperatorBuilder
}

func (b *BrokerOperatorBuilder) Build() (OperatorSetup, error) {
	broker := &BrokerOperator{}
	if err := broker.InitFromBaseOperatorBuilder(&b.BaseOperatorBuilder); err != nil {
		return broker, err
	}

	if brokerclient, err := brokerclientset.NewForConfig(b.restConfig); err != nil {
		return broker, err
	} else {
		broker.brokerClient = brokerclient
	}

	broker.customCommand = b.customCommand
	// Setting up the defaults
	if broker.yamls != nil {

	} else if broker.yamlURLs == nil {
		baseImportPath := "https://raw.githubusercontent.com/rh-messaging/activemq-artemis-operator/master/deploy/"
		broker.yamlURLs = []string{
			baseImportPath + "service_account.yaml",
			baseImportPath + "role.yaml",
			baseImportPath + "role_binding.yaml",
			baseImportPath + "crds/broker_activemqartemis_crd.yaml",
			baseImportPath + "crds/broker_activemqartemisaddress_crd.yaml",
			baseImportPath + "crds/broker_activemqartemisscaledown_crd.yaml",
			baseImportPath + "crds/broker_activemqartemissecurity_crd.yaml",
			baseImportPath + "operator.yaml",
		}
	}

	return broker, nil
}

func (b *BrokerOperatorBuilder) OperatorType() OperatorType {
	return OperatorTypeBroker
}

type BrokerOperator struct {
	BaseOperator
	brokerClient brokerclientset.Interface
}

func (b *BrokerOperator) Namespace() string {
	return b.namespace
}

func (b *BrokerOperator) Setup() error {
	log.Logf("Setting up from YAMLs (brokeroperator)")
	if err := b.SetupYamls(); err != nil {
		return err
	}
	return nil
}

func (b *BrokerOperator) TeardownEach() error {
	log.Logf("deliting operator from %s", b.Namespace())
	err := b.kubeClient.CoreV1().ServiceAccounts(b.Namespace()).Delete(b.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s service account: %v", b.Name(), err)
	}
	err = b.kubeClient.RbacV1().Roles(b.Namespace()).Delete(b.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s role: %v", b.Name(), err)
	}
	err = b.kubeClient.RbacV1().RoleBindings(b.Namespace()).Delete(b.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s role binding: %v", b.Name(), err)
	}
	err = b.kubeClient.AppsV1().Deployments(b.Namespace()).Delete(b.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s deployment: %v", b.Name(), err)
	}

	log.Logf("%s teardown namespace successful", b.Name())
	return nil
}

func (b *BrokerOperator) TeardownSuite() error {
	// If CRD Was found prior to setup, keep cluster level resources
	if b.keepCRD {
		return nil
	}

	err := b.kubeClient.RbacV1().ClusterRoles().Delete(b.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s cluster role: %v", b.Name(), err)
	}
	err = b.kubeClient.RbacV1().ClusterRoleBindings().Delete(b.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s cluster role binding: %v", b.Name(), err)
	}
	for _, crdName := range b.CRDNames() {
		err = b.extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %s crd: %v", b.Name(), err)
		}
	}

	log.Logf("%s teardown suite successful", b.Name())
	return nil
}

func (b *BrokerOperator) Image() string {
	return b.image
}

func (b *BrokerOperator) CRDNames() []string {
	return []string{
		"activemqartemis." + b.GroupName(),
		"activemqartemisaddress." + b.GroupName(),
		"activemqartemisscaledown." + b.GroupName(),
	}
}

func (b *BrokerOperator) GroupName() string {
	return "broker.amq.io"
}

func (b *BrokerOperator) APIVersion() string {
	return b.apiVersion
}

func (b *BrokerOperator) Name() string {
	return b.operatorName
}

func (b *BrokerOperator) Interface() interface{} {
	return b.brokerClient
}
