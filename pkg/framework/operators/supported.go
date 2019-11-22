package operators

type OperatorType int

const (
	OperatorTypeQdr OperatorType = iota
	OperatorTypeBase OperatorType = iota
)

var (
	SupportedOperators = map[OperatorType]OperatorSetupBuilder{
		OperatorTypeQdr: &QdrOperatorBuilder{},
		OperatorTypeBase: &BaseOperatorBuilder{},
	}
)
