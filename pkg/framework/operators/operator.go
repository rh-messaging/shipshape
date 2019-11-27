package operators

import "k8s.io/client-go/rest"

type OperatorSetupBuilder interface {
	NewBuilder(restConfig *rest.Config) OperatorSetupBuilder
	WithNamespace(namespace string) OperatorSetupBuilder
	WithImage(image string) OperatorSetupBuilder
	WithYamls(yamls []string) OperatorSetupBuilder
	AddYaml(yaml string) OperatorSetupBuilder
	WithOperatorName(name string) OperatorSetupBuilder
	KeepCdr(keepCdrs bool) OperatorSetupBuilder
	WithApiVersion(apiVersion string) OperatorSetupBuilder
	Build() (OperatorAccessor, error)
	OperatorType() OperatorType
}

type OperatorAccessor interface {
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
