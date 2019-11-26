package operators

import (
	"fmt"
	qdrclientset "github.com/interconnectedcloud/qdr-operator/pkg/client/clientset/versioned"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
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
	log.Logf("Setting up Service Account")
	if err := q.SetupServiceAccount(); err != nil {
		return err
	}

	log.Logf("Setting up Role")
	if err := q.SetupRole(); err != nil {
		return err
	}

	log.Logf("Setting up Cluster Role")
	if err := q.SetupClusterRole(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	log.Logf("Setting up Role Binding")
	if err := q.SetupRoleBinding(); err != nil {
		return err
	}

	log.Logf("Setting up Cluster Role Binding")
	if err := q.SetupClusterRoleBinding(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	log.Logf("Setting up CRD")
	if err := q.SetupCRD(); err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	} else if err != nil {
		// In case CRD already exists, do not remove on clean up (to preserve original state)
		q.keepCRD = true

	}

	log.Logf("Setting up Operator Deployment")
	if err := q.SetupDeployment(); err != nil {
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

func (q *QdrOperator) SetupServiceAccount() error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: q.Name(),
		},
	}
	_, err := q.kubeClient.CoreV1().ServiceAccounts(q.Namespace()).Create(sa)
	if err != nil {
		return fmt.Errorf("create %s service account failed: %v", q.Name(), err)
	}
	return nil
}

func (q *QdrOperator) SetupRole() error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: q.Name(),
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
	_, err := q.kubeClient.RbacV1().Roles(q.Namespace()).Create(role)
	if err != nil {
		return fmt.Errorf("create qdr-operator role failed: %v", err)
	}
	return nil
}

func (q *QdrOperator) SetupClusterRole() error {
	crole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: q.Name(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err := q.kubeClient.RbacV1().ClusterRoles().Create(crole)
	if err != nil {
		return fmt.Errorf("create qdr-operator cluster role failed: %v", err)
	}
	return nil
}

func (q *QdrOperator) SetupRoleBinding() error {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: q.Name(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     q.Name(),
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      q.Name(),
				Namespace: q.Namespace(),
			},
		},
	}
	_, err := q.kubeClient.RbacV1().RoleBindings(q.Namespace()).Create(rb)
	if err != nil {
		return fmt.Errorf("create qdr-operator role binding failed: %v", err)
	}
	return nil
}

func (q *QdrOperator) SetupClusterRoleBinding() error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: q.Name(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     q.Name(),
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      q.Name(),
				Namespace: q.Namespace(),
			},
		},
	}
	_, err := q.kubeClient.RbacV1().ClusterRoleBindings().Create(crb)
	if err != nil {
		return fmt.Errorf("create qdr-operator cluster role binding failed: %v", err)
	}
	return nil
}

func (q *QdrOperator) SetupCRD() error {
	for _, crdName := range q.CRDNames() {
		crd := &apiextv1b1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crdName,
			},
			Spec: apiextv1b1.CustomResourceDefinitionSpec{
				Group: q.GroupName(),
				Names: apiextv1b1.CustomResourceDefinitionNames{
					Kind:     "Interconnect",
					ListKind: "InterconnectList",
					Plural:   "interconnects",
					Singular: "interconnect",
				},
				Scope:   "Namespaced",
				Version: q.APIVersion(),
			},
		}
		_, err := q.extClient.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
		if err != nil {
			return fmt.Errorf("create qdr-operator crd failed: %v", err)
		}
	}
	return nil
}

func (q *QdrOperator) SetupDeployment() error {
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: q.Name(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": q.Name(),
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": q.Name(),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: q.Name(),
					Containers: []corev1.Container{
						{
							Command:         []string{q.Name()},
							Name:            q.Name(),
							Image:           q.Image(),
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
									Value: q.Name(),
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
	_, err := q.kubeClient.AppsV1().Deployments(q.Namespace()).Create(dep)
	if err != nil {
		return fmt.Errorf("create qdr-operator deployment failed: %v", err)
	}
	return nil
}

func int32Ptr(i int32) *int32 { return &i }
