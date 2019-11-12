package qeclients

import (
	"github.com/fgiorgetti/shipshape/pkg/api/client/amqp"
	"github.com/fgiorgetti/shipshape/pkg/framework"
)

// AmqpQEClientImpl specifies the available Amqp QE Clients
type AmqpQEClientImpl int

const (
	Python  AmqpQEClientImpl = iota
	Timeout int              = 60
)

// NewAmqpSender Builds a very basic Amqp Sender client using one of the
// available QE Clients.
// Message body will be truncated if it exceeds 256 bytes.
// TODO If Message content (body) is large, create a secret and mount it within the client Pod
//      Probably something to be done at the "topology setup".
func NewAmqpSender(impl AmqpQEClientImpl, name string, ctx framework.ContextData, url string, count int, body string) (amqp.Client, error) {
	// Generic Sender Builder
	var senderBuilder amqp.SenderBuider
	senderBuilder = NewSenderBuilder(impl)

	// Prepare the new basic sender
	senderBuilder.New(name, ctx, url)

	// Truncating message body in case it exceeds 256 bytes
	maxLength := 256
	if len(body) < maxLength {
		maxLength = len(body)
	}

	return senderBuilder.Timeout(Timeout).Messages(count).MessageContent(body[:maxLength]).Build()
}

func NewSenderBuilder(impl AmqpQEClientImpl) amqp.SenderBuider {
	var senderBuilder amqp.SenderBuider
	switch impl {
	case Python:
		fallthrough
	default:
		senderBuilder = new(AmqpPythonSenderBuilder)
	}
	return senderBuilder
}

// NewAmqpReceiver Builds a very basic Amqp Receiver client using one of the
// available QE clients.
func NewAmqpReceiver(impl AmqpQEClientImpl, name string, ctx framework.ContextData, url string, count int) (amqp.Client, error) {
	// Generic Sender Builder
	var receiverBuilder amqp.ReceiverBuilder
	receiverBuilder = NewReceiverBuilder(impl)

	// Prepare the new basic sender
	receiverBuilder.New(name, ctx, url)
	return receiverBuilder.Timeout(Timeout).Messages(count).Build()
}

func NewReceiverBuilder(impl AmqpQEClientImpl) amqp.ReceiverBuilder {
	var receiverBuilder amqp.ReceiverBuilder
	switch impl {
	case Python:
		fallthrough
	default:
		receiverBuilder = new(AmqpPythonReceiverBuilder)
	}
	return receiverBuilder
}
