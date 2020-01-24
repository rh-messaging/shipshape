package qeclients

// AmqpQEClientImpl specifies the available Amqp QE Clients
type AmqpQEClientImpl int

const (
	Python AmqpQEClientImpl = iota
	Java
	NodeJS
	Timeout int = 60
)

var (
	QEClientImageMap = map[AmqpQEClientImpl]AmqpQEClientImplInfo{
		Python: {
			Name:    "cli-proton-python",
			Image:   "docker.io/rhmessagingqe/cli-proton-python:latest",
			CommandSender: "cli-proton-python-sender",
			CommandReceiver: "cli-proton-python-receiver",
		},
		Java: {
			Name:    "cli-qpid-java",
			Image:   "docker.io/rhmessagingqe/cli-java:latest",
			CommandSender: "cli-qpid-sender",
			CommandReceiver: "cli-qpid-receiver",
		},
		NodeJS: {
			Name:    "cli-rhea-nodejs",
			Image:   "docker.io/rhmessagingqe/cli-rhea:centos7",
			CommandSender: "cli-rhea-sender",
			CommandReceiver: "cli-rhea-receiver",
		},
	}
)

type AmqpQEClientImplInfo struct {
	Name            string
	Image           string
	CommandSender   string
	CommandReceiver string
}

// Common builder properties and methods to be reused by sender/receiver builders
type AmqpQEClientBuilderCommon struct {
	customImage      string

	MessageCount     int
}

func (a *AmqpQEClientBuilderCommon) Messages(count int) *AmqpQEClientBuilderCommon {
	a.MessageCount = count
	return a
}

func (a *AmqpQEClientBuilderCommon) CustomImage(image string) *AmqpQEClientBuilderCommon {
	a.customImage = image
	return a
}
