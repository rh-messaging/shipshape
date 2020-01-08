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
			image: "brew-pulp-docker01.web.prod.ext.phx2.redhat.com:8888/amq7/amq-broker-operator:0.9",
			operatorName: "activemq-artemis-operator",
			keepCdrs: true,
			apiVersion: "v1alpha1",
		}},
	}
)
