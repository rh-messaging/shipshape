package operators

import (
	restclient "k8s.io/client-go/rest"
)

type OperatorSetupBuilder interface {
	NewForConfig(namespace string,
		restConfig *restclient.Config,
		operatorConfig OperatorConfig) (OperatorDescription, error)

}

type OperatorConfig interface {
	ApiVersion() string
	OperatorName() string
	YamlUrls() []string
	KeepCRD() bool
}

type OperatorDescription interface {
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
