package qeclients

import (
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"sync"
)

const (
	MountPath = "/opt/messaging-files"
)

type AmqpQESenderBuilder struct {
	*AmqpQEClientBuilderCommon
	sender                    *AmqpQEClientCommon
	ContentConfigMap          string
	MessageContent            string
	MessageContentFromFileKey string
}

func NewSenderBuilder(name string, impl AmqpQEClientImpl, data framework.ContextData, url string) *AmqpQESenderBuilder {
	sb := new(AmqpQESenderBuilder)
	sb.AmqpQEClientBuilderCommon = &AmqpQEClientBuilderCommon{}
	sb.sender = &AmqpQEClientCommon{
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
	return sb
}

func (a *AmqpQESenderBuilder) Count(count int) *AmqpQESenderBuilder {
	a.MessageCount = count
	return a
}

func (a *AmqpQESenderBuilder) Timeout(timeout int) *AmqpQESenderBuilder {
	a.sender.Timeout = timeout
	return a
}

func (a *AmqpQESenderBuilder) Content(content string) *AmqpQESenderBuilder {
	a.MessageContent = content
	return a
}

// MessageContentFromFile uses the given config map name and just the filename reference,
// inside your configmap Data (key for the file)
func (a *AmqpQESenderBuilder) MessageContentFromFile(configMapName string, filenameKey string) *AmqpQESenderBuilder {
	a.MessageContentFromFileKey = filenameKey
	a.ContentConfigMap = configMapName
	return a
}

func (a *AmqpQESenderBuilder) Build() (*AmqpQEClientCommon, error) {
	// Preparing Pod, Container (commands and args), Volumes and etc
	podBuilder := framework.NewPodBuilder(a.sender.Name, a.sender.Context.Namespace)
	podBuilder.AddLabel("amqp-client-impl", QEClientImageMap[a.sender.Implementation].Name)
	podBuilder.RestartPolicy("Never")

	// Adding VolumeSource for provided configMap
	if a.ContentConfigMap != "" {
		podBuilder.AddConfigMapVolumeSource(a.ContentConfigMap, a.ContentConfigMap)
	}

	//
	// Helps building the container for sender pod
	//
	//
	// Helps building the container for sender pod
	//
	image := QEClientImageMap[a.sender.Implementation].Image
	if a.customImage != "" {
		image = a.customImage
	}

	cBuilder := framework.NewContainerBuilder(a.sender.Name, image)
	if a.customCommand == "" {
		cBuilder.WithCommands(QEClientImageMap[a.sender.Implementation].CommandSender)
	} else {
		cBuilder.WithCommands(a.customCommand)
	}

	//
	// Adds args (may vary from one implementation to another)
	//

	//
	// Common options first
	//

	// URL
	cBuilder.AddArgs(parseUrl(a.sender)...)

	// Message count
	cBuilder.AddArgs(parseCount(a.MessageCount)...)

	// Timeout
	cBuilder.AddArgs(parseTimeout(a.sender.Timeout)...)

	//
	// Sender specific options
	//

	// Source for message content (file or arg)
	if a.MessageContentFromFileKey != "" {
		cBuilder.AddVolumeMountConfigMapData(a.ContentConfigMap, MountPath, true)
		cBuilder.AddArgs("--msg-content-from-file", MountPath+"/"+a.MessageContentFromFileKey)
	} else {
		cBuilder.AddArgs("--msg-content", a.MessageContent)
	}

	// Static options
	cBuilder.AddArgs("--log-msgs", "json")
	if a.customCommand == "" || a.customCommand == "cli-qpid-sender" {
		cBuilder.AddArgs("--on-release", "retry")
	}

	// Retrieving container and adding to pod
	c := cBuilder.Build()
	podBuilder.AddContainer(c)
	pod := podBuilder.Build()
	a.sender.Pod = pod

	return a.sender, nil
}
