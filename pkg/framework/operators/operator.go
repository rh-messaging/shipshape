package operators

import (
	restclient "k8s.io/client-go/rest"
)

type OperatorSetupBuilder interface {
	NewForConfig(namespace string,
		restConfig *restclient.Config,
		operatorConfig OperatorConfig) (OperatorSetup, error)

}

type OperatorConfig interface {
	ApiVersion() string
	OperatorName() string
	YamlUrls() []string
	KeepCRD() bool
}

type OperatorSetup interface {
	Interface() interface{}
	Namespace() string
	Image() string
	Name() string
	CRDNames() []string
	GroupName() string
	APIVersion() string
	Setup() error
	TeardownEach() error
	TeardownSuite() error
}
