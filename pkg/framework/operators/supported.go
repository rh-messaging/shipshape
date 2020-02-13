package operators

type OperatorType int

const (
	OperatorTypeQdr OperatorType = iota
	OperatorTypeBroker OperatorType = iota
)

var (
	SupportedOperators = map[OperatorType]OperatorSetupBuilder{
		OperatorTypeQdr: &QdrOperatorBuilder{BaseOperatorBuilder{
			image:        "quay.io/interconnectedcloud/qdr-operator",
			operatorName: "qdr-operator",
			keepCdrs:     true,
			apiVersion:   "v1alpha1",
		}},
		OperatorTypeBroker: &BrokerOperatorBuilder {BaseOperatorBuilder{
			image: "quay.io/artemiscloud/activemq-artemis-operator:latest",
			operatorName: "activemq-artemis-operator",
			keepCdrs: true,
			apiVersion: "v1alpha1",
		}},
	}
)
