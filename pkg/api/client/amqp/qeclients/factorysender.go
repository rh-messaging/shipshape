package qeclients

import (
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"strconv"
	"sync"
)

const (
	MountPath = "/opt/messaging-files"
)

type AmqpQESenderBuilder struct {
	sender           *AmqpQESender
	contentConfigMap string
}

func NewSenderBuilder(name string, impl AmqpQEClientImpl, data framework.ContextData, url string) *AmqpQESenderBuilder {
	sb := new(AmqpQESenderBuilder)
	sb.sender = &AmqpQESender{
		AmqpQEClientCommon: AmqpQEClientCommon{
			AmqpClientCommon: amqp.AmqpClientCommon{
				Context: data,
				Name:    name,
				Url:     url,
				Timeout: Timeout,
				Params:  []amqp.Param{},
				Mutex:   sync.Mutex{},
			},
			Implementation: impl,
		},
	}
	return sb
}

func (a *AmqpQESenderBuilder) Timeout(timeout int) *AmqpQESenderBuilder {
	a.sender.Timeout = timeout
	return a
}

func (a *AmqpQESenderBuilder) Messages(count int) *AmqpQESenderBuilder {
	a.sender.MessageCount = count
	return a
}

func (a *AmqpQESenderBuilder) MessageContent(content string) *AmqpQESenderBuilder {
	a.sender.MessageContent = content
	return a
}

// MessageContentFromFile uses the given config map name and just the filename reference,
// inside your configmap Data (key for the file)
func (a *AmqpQESenderBuilder) MessageContentFromFile(configMapName string, filenameKey string) *AmqpQESenderBuilder {
	a.sender.MessageContentFromFile = filenameKey
	a.contentConfigMap = configMapName
	return a
}

func (a *AmqpQESenderBuilder) Build() (*AmqpQESender, error) {
	// Preparing Pod, Container (commands and args), Volumes and etc
	podBuilder := framework.NewPodBuilder(a.sender.Name, a.sender.Context.Namespace)
	podBuilder.AddLabel("amqp-client-impl", QEClientImageMap[a.sender.Implementation].Name)
	podBuilder.RestartPolicy("Never")

	// Adding VolumeSource for provided configMap
	if a.contentConfigMap != "" {
		podBuilder.AddConfigMapVolumeSource(a.contentConfigMap, a.contentConfigMap)
	}

	//
	// Helps building the container for sender pod
	//
	cBuilder := framework.NewContainerBuilder(a.sender.Name, QEClientImageMap[a.sender.Implementation].Image)
	cBuilder.WithCommands(QEClientImageMap[a.sender.Implementation].CommandSender)

	//
	// Adds args (may vary from one implementation to another)
	//

	// URL
	cBuilder.AddArgs("--broker-url", a.sender.Url)

	// Message count
	cBuilder.AddArgs("--count", strconv.Itoa(a.sender.MessageCount))

	// Timeout
	cBuilder.AddArgs("--timeout", strconv.Itoa(a.sender.Timeout))

	// Source for message content (file or arg)
	if a.sender.MessageContentFromFile != "" {
		cBuilder.AddVolumeMountConfigMapData(a.contentConfigMap, MountPath, true)
		cBuilder.AddArgs("--msg-content-from-file", MountPath + "/" + a.sender.MessageContentFromFile)
	} else {
		cBuilder.AddArgs("--msg-content", a.sender.MessageContent)
	}

	// Static options
	cBuilder.AddArgs("--log-msgs", "json", "--on-release", "retry")

	// Retrieving container and adding to pod
	c := cBuilder.Build()
	podBuilder.AddContainer(c)
	pod := podBuilder.Build()
	a.sender.Pod = pod

	return a.sender, nil
}
