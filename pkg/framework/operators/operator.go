package operators

import (
	restclient "k8s.io/client-go/rest"
)

type OperatorSetupBuilder interface {
	NewForConfig(namespace string, restConfig *restclient.Config) (OperatorSetup, error)
}

type OperatorSetup interface {
	Interface() interface{}
	Namespace() string
	Image() string
	Name() string
	CRDName() string
	GroupName() string
	APIVersion() string
	Setup() error
	TeardownEach() error
	TeardownSuite() error
}
