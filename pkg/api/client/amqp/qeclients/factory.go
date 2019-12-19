package qeclients

import (
	"fmt"
	"github.com/rh-messaging/shipshape/pkg/framework"
)

// AmqpQEClientImpl specifies the available Amqp QE Clients
type AmqpQEClientImpl int

const (
	Python AmqpQEClientImpl = iota
	Java
	NodeJS
	Timeout int = 60
)

type AmqpQEClientImplInfo struct {
	Name            string
	Image           string
	CommandSender   string
	CommandReceiver string
}

var (
	QEClientImageMap = map[AmqpQEClientImpl]AmqpQEClientImplInfo{
		Python: {
			Name:    "cli-proton-python",
			Image:   "docker.io/rhmessagingqe/cli-proton-python:latest",
			CommandSender: "cli-proton-python-sender",
			CommandReceiver: "cli-proton-python-receiver",
		},
	}
)

func sampleClient() {

	ctx := framework.ContextData{}

	// Prepare my python sender
	sb := NewSenderBuilder("sender-python-1", Python, ctx, "amqp://my.sample.url:5672/myAddress")
	sb.MessageContentFromFile("messaging-files", "large-messages.txt")
	sb.Messages(100)
	sender, _ := sb.Build()
	err := sender.Deploy()

	fmt.Print(err)


}
