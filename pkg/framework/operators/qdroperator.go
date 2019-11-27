package operators

import (
	"fmt"
	qdrclientset "github.com/interconnectedcloud/qdr-operator/pkg/client/clientset/versioned"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Reusing BaseOperatorBuilder implementation and adding
// the "abstract" method Build() and OperatorType()
type QdrOperatorBuilder struct{
	BaseOperatorBuilder
}

func (q *QdrOperatorBuilder) Build() (OperatorAccessor, error) {
	qdr := &QdrOperator{}
	if err := qdr.InitFromBaseOperatorBuilder(&q.BaseOperatorBuilder); err != nil {
		return qdr, err
	}

	// initializing qdrclient
	if client, err := qdrclientset.NewForConfig(q.restConfig); err != nil {
		return qdr, err
	} else {
		qdr.qdrClient = client
	}

	// Setting up the defaults
	baseImportPath := "https://raw.githubusercontent.com/interconnectedcloud/qdr-operator/master/deploy/"
	if qdr.yamls == nil {
		qdr.yamls = []string{
			baseImportPath + "service_account.yaml",
			baseImportPath + "role.yaml",
			baseImportPath + "role_binding.yaml",
			baseImportPath + "cluster_role.yaml",
			baseImportPath + "cluster_role_binding.yaml",
			baseImportPath + "crds/interconnectedcloud_v1alpha1_interconnect_crd.yaml",
			baseImportPath + "operator.yaml",
		}
	}

	return qdr, nil
}

func (q *QdrOperatorBuilder) OperatorType() OperatorType {
	return OperatorTypeQdr
}

type QdrOperator struct {
	BaseOperator
	qdrClient  qdrclientset.Interface
}

func (q *QdrOperator) Namespace() string {
	return q.namespace
}

func (q *QdrOperator) Setup() error {
	log.Logf("Setting up from YAMLs")
	if err := q.SetupYamls(); err != nil {
		return err
	}

	return nil

}

func (q *QdrOperator) TeardownEach() error {
	err := q.kubeClient.CoreV1().ServiceAccounts(q.Namespace()).Delete(q.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s service account: %v", q.Name(), err)
	}
	err = q.kubeClient.RbacV1().Roles(q.Namespace()).Delete(q.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s role: %v", q.Name(), err)
	}
	err = q.kubeClient.RbacV1().RoleBindings(q.Namespace()).Delete(q.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s role binding: %v", q.Name(), err)
	}
	err = q.kubeClient.AppsV1().Deployments(q.Namespace()).Delete(q.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s deployment: %v", q.Name(), err)
	}

	log.Logf("%s teardown namespace successful", q.Name())
	return nil
}

func (q *QdrOperator) TeardownSuite() error {
	// If CRD Was found prior to setup, keep cluster level resources
	if q.keepCRD {
		return nil
	}

	err := q.kubeClient.RbacV1().ClusterRoles().Delete(q.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s cluster role: %v", q.Name(), err)
	}
	err = q.kubeClient.RbacV1().ClusterRoleBindings().Delete(q.Name(), metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s cluster role binding: %v", q.Name(), err)
	}
	for _, crdName := range q.CRDNames() {
		err = q.extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crdName, metav1.NewDeleteOptions(1))
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete %s crd: %v", q.Name(), err)
		}
	}

	log.Logf("%s teardown suite successful", q.Name())
	return nil
}

func (q *QdrOperator) Image() string {
	return q.image
}

func (q *QdrOperator) CRDNames() []string {
	return []string{"interconnects.interconnectedcloud.github.io"}
}

func (q *QdrOperator) GroupName() string {
	return "interconnectedcloud.github.io"
}

func (q *QdrOperator) APIVersion() string {
	return q.apiVersion
}

func (q *QdrOperator) Name() string {
	return q.operatorName
}

func (q *QdrOperator) Interface() interface{} {
	return q.qdrClient
}

func int32Ptr(i int32) *int32 { return &i }
