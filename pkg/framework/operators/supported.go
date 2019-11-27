package operators

type OperatorType int

const (
	OperatorTypeQdr OperatorType = iota
)

var (
	SupportedOperators = map[OperatorType]OperatorSetupBuilder{
		OperatorTypeQdr: &QdrOperatorBuilder{BaseOperatorBuilder{
			image:        "quay.io/interconnectedcloud/qdr-operator",
			operatorName: "qdr-operator",
			keepCdrs:     true,
			apiVersion:   "v1alpha1",
		}},
	}
)
