package operators

type OperatorType int

const (
	OperatorTypeQdr OperatorType = iota
	OperatorTypeBroker OperatorType = iota
)

var (
	SupportedOperators = map[OperatorType]OperatorSetupBuilder{
		OperatorTypeQdr: &QdrOperatorBuilder{},
        OperatorTypeBroker: &BrokerOperatorBuilder{},
	}
)
