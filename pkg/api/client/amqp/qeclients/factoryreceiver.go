package qeclients

import (
	"sync"

	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
)

type AmqpQEReceiverBuilder struct {
	*AmqpQEClientBuilderCommon
	receiver *AmqpQEClientCommon
}

func NewReceiverBuilder(name string, impl AmqpQEClientImpl, data framework.ContextData, url string) *AmqpQEReceiverBuilder {
	rb := new(AmqpQEReceiverBuilder)
	rb.AmqpQEClientBuilderCommon = &AmqpQEClientBuilderCommon{}
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

func (a *AmqpQEReceiverBuilder) WithCount(count int) *AmqpQEReceiverBuilder {
	a.MessageCount = count
	return a
}

func (a *AmqpQEReceiverBuilder) Build() (*AmqpQEClientCommon, error) {
	// Preparing Pod, Container (commands and args) and etc
	podBuilder := framework.NewPodBuilder(a.receiver.Name, a.receiver.Context.Namespace, a.receiver.Context.ServerVersion)
	podBuilder.AddLabel("amqp-client-impl", QEClientImageMap[a.receiver.Implementation].Name)
	podBuilder.RestartPolicy("Never")

	//
	// Helps building the container for sender pod
	//
	image := QEClientImageMap[a.receiver.Implementation].Image
	if a.customImage != "" {
		image = a.customImage
	}
	cBuilder := framework.NewContainerBuilder(a.receiver.Name, image)
	if a.customCommand == "" {
		cBuilder.WithCommands(QEClientImageMap[a.receiver.Implementation].CommandSender)
	} else {
		cBuilder.WithCommands(a.customCommand)
	}
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

	// Specific to cli-proton-python and cli-rhea
	impl := a.receiver.Implementation
	if impl == Python {
		cBuilder.AddArgs("--reactor-auto-accept")
	}

	// Retrieving container and adding to pod
	c := cBuilder.Build()
	podBuilder.AddContainer(c)
	pod := podBuilder.Build()
	a.receiver.Pod = pod

	return a.receiver, nil
}
