package qeclients

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
