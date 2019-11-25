package operators

type OperatorSetupBuilder interface {
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
