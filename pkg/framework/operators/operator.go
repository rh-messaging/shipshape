package operators

import (
	"k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type OperatorSetupBuilder interface {
	NewBuilder(restConfig *rest.Config, rawConfig *clientcmdapi.Config) OperatorSetupBuilder
	WithNamespace(namespace string) OperatorSetupBuilder
	WithImage(image string) OperatorSetupBuilder
	WithCommand(command string) OperatorSetupBuilder
	WithYamls(yamls []string) OperatorSetupBuilder
	AddYaml(yaml string) OperatorSetupBuilder
	WithOperatorName(name string) OperatorSetupBuilder
	KeepCdr(keepCdrs bool) OperatorSetupBuilder
	WithApiVersion(apiVersion string) OperatorSetupBuilder
	Build() (OperatorSetup, error)
	OperatorType() OperatorType
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
