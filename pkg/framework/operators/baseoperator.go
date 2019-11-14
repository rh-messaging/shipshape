type BaseOperator struct {
	namespace string
	restConfig *rest.Config
	kubeClient clientset.Interface
	extClient  apiextension.Interface
	keepCRD    bool
	imageName string
}


func (b* BaseOperator) TeardownSuite(name string) error {
    
    if b.keepCRD {
		return nil
	}

	err := b.kubeClient.RbacV1().ClusterRoles().Delete(name, metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s cluster role: %v", name, err)
	}
	err = b.kubeClient.RbacV1().ClusterRoleBindings().Delete(name, metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s cluster role binding: %v", name, err)
	}
	//Custom Resource Definition teardown is up to children classes
	
	log.Logf("%s teardown suite successful", name)
	return nil
}

func (b* BaseOperator) TeardownEach(name string) error {
	err := b.kubeClient.CoreV1().ServiceAccounts(b.Namespace()).Delete(name, metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s service account: %v", name, err)
	}
	err = b.kubeClient.RbacV1().Roles(b.Namespace()).Delete(name, metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s role: %v", name, err)
	}
	err = b.kubeClient.RbacV1().RoleBindings(b.Namespace()).Delete(name, metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s role binding: %v", name, err)
	}
	err = b.kubeClient.AppsV1().Deployments(b.Namespace()).Delete(name, metav1.NewDeleteOptions(1))
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete %s deployment: %v", name, err)
	}

	log.Logf("%s teardown namespace successful", name)
	return nil
}


func (b* BaseOperator) SetupRole(name string) error {
    role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
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
		},
	}
	_, err := b.kubeClient.RbacV1().Roles(b.Namespace()).Create(role)
	return err
}

func (b* BaseOperator) SetupClusterRole(name string) error {
	crole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"apiextensions.k8s.io"},
				Resources: []string{"customresourcedefinitions"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	_, err := b.kubeClient.RbacV1().ClusterRoles().Create(crole)
	return err
}

func (b* BaseOperator) SetupRoleBinding(name string) error {
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: b.Namespace(),
			},
		},
	}
	_, err := b.kubeClient.RbacV1().RoleBindings(b.Namespace()).Create(rb)
	return err
}

func (b* BaseOperator) SetupClusterRoleBinding(name string) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
		Subjects: []rbacv1.Subject{
			{
				APIGroup:  "",
				Kind:      "ServiceAccount",
				Name:      name,
				Namespace: b.Namespace(),
			},
		},
	}
	_, err := q.kubeClient.RbacV1().ClusterRoleBindings().Create(crb)
	return err
}      

func (b* BaseOperator SetupDeployment(name string, 
                                      image string) error {
    dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"name": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: name,
					Containers: []corev1.Container{
						{
							Command:         []string{name},
							Name:            name,
							Image:           image,
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
									Value: name,
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
	_, err := q.kubeClient.AppsV1().Deployments(b.namespace).Create(dep)
    return err
}
