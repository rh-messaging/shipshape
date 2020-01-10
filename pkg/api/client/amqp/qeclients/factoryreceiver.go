package qeclients

import (
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"sync"
)

type AmqpQEReceiverBuilder struct {
	receiver         *AmqpQEClientCommon
	MessageCount     int
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

	//
	// Adds args (may vary from one implementation to another)
	//

	// URL
	// Parsing URL
	cBuilder.AddArgs(parseUrl(a.receiver)...)

	// Message count
	cBuilder.AddArgs(parseCount(a.MessageCount)...)

	// Timeout
	cBuilder.AddArgs(parseTimeout(a.receiver.Timeout)...)

	// Static options
	cBuilder.AddArgs("--log-msgs", "json")

	// Retrieving container and adding to pod
	c := cBuilder.Build()
	podBuilder.AddContainer(c)
	pod := podBuilder.Build()
	a.receiver.Pod = pod

	return a.receiver, nil
}
