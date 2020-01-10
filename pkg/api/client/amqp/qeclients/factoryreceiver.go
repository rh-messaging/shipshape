package qeclients

import (
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"strconv"
	"sync"
)

type AmqpQEReceiverBuilder struct {
	receiver     *AmqpQEClientCommon
	MessageCount int
}

func NewReceiverBuilder(name string, impl AmqpQEClientImpl, data framework.ContextData, url string) *AmqpQEReceiverBuilder {
	rb := new(AmqpQEReceiverBuilder)
	rb.receiver = &AmqpQEClientCommon{
		AmqpClientCommon: amqp.AmqpClientCommon{
			Context: data,
			Name:    name,
			Url:     url,
			Timeout: Timeout,
			Params:  []amqp.Param{},
			Mutex:   sync.Mutex{},
		},
		Implementation: impl,
	}
	return rb
}

func (a *AmqpQEReceiverBuilder) Timeout(timeout int) *AmqpQEReceiverBuilder {
	a.receiver.Timeout = timeout
	return a
}

func (a *AmqpQEReceiverBuilder) Messages(count int) *AmqpQEReceiverBuilder {
	a.MessageCount = count
	return a
}

func (a *AmqpQEReceiverBuilder) addSpecificImplementationOptions(cBuilder *framework.ContainerBuilder) {
	switch a.receiver.Implementation {
	// URL
	case MultipleReceiversPython:
		{
			cBuilder.AddArgs("--address", a.receiver.Url)
			cBuilder.AddArgs("--connections", "100") //total connections
			cBuilder.AddArgs("--links", "500")       //total links per connection
		}
	default:
		{
			cBuilder.AddArgs("--broker-url", a.receiver.Url)
			cBuilder.AddArgs("--count", strconv.Itoa(a.MessageCount))
			cBuilder.AddArgs("--timeout", strconv.Itoa(a.receiver.Timeout))
			cBuilder.AddArgs("--log-msgs", "json")
		}
	}
}

func (a *AmqpQEReceiverBuilder) Build() (*AmqpQEClientCommon, error) {
	// Preparing Pod, Container (commands and args) and etc
	podBuilder := framework.NewPodBuilder(a.receiver.Name, a.receiver.Context.Namespace)
	podBuilder.AddLabel("amqp-client-impl", QEClientImageMap[a.receiver.Implementation].Name)
	podBuilder.RestartPolicy("Never")

	//
	// Helps building the container for sender pod
	//
	cBuilder := framework.NewContainerBuilder(a.receiver.Name, QEClientImageMap[a.receiver.Implementation].Image)
	cBuilder.WithCommands(QEClientImageMap[a.receiver.Implementation].CommandReceiver)

	a.addSpecificImplementationOptions(cBuilder)

	// Retrieving container and adding to pod
	c := cBuilder.Build()
	podBuilder.AddContainer(c)
	pod := podBuilder.Build()
	a.receiver.Pod = pod

	return a.receiver, nil
}
