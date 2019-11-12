package qeclients

import (
	"github.com/fgiorgetti/shipshape/pkg/api/client/amqp"
	"github.com/fgiorgetti/shipshape/pkg/framework"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"sync"
)

// AmqpPythonSender amqp Client implementation
type AmqpPythonSender struct {
	AmqpClientCommon
	Content string
}

// AmqpPythonSenderBuilder amqp SenderBuilder implementation
type AmqpPythonSenderBuilder struct {
	pythonSender *AmqpPythonSender
}

func (a *AmqpPythonSenderBuilder) New(name string, data framework.ContextData, url string) amqp.SenderBuider {
	a.pythonSender = &AmqpPythonSender{
		AmqpClientCommon: AmqpClientCommon{
			Name:    name,
			Context: data,
			Url:     url,
			Mutex:   sync.Mutex{},
			Timeout: amqp.TimeoutDefaultSecs,
		},
	}
	return a
}

func (a *AmqpPythonSenderBuilder) Messages(count int) amqp.SenderBuider {
	a.pythonSender.Messages = count
	return a
}

func (a *AmqpPythonSenderBuilder) Timeout(timeout int) amqp.SenderBuider {
	a.pythonSender.Timeout = timeout
	return a
}

func (a *AmqpPythonSenderBuilder) Param(name string, value string) amqp.SenderBuider {
	if a.pythonSender.Params == nil {
		a.pythonSender.Params = make([]amqp.Param, 0)
	}
	a.pythonSender.Params = append(a.pythonSender.Params, amqp.Param{Name: name, Value: value})
	return a
}

func (a *AmqpPythonSenderBuilder) MessageContent(content string) amqp.SenderBuider {
	a.pythonSender.Content = content
	return a
}

func (a *AmqpPythonSenderBuilder) Build() (amqp.Client, error) {
	//TODO Add some validation to ensure all required have been populated
	// and return an error instead of nil
	s := a.pythonSender
	s.Pod = &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Context.Namespace,
			Labels: map[string]string{
				"amqp-client": "cli-proton-python-sender",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    s.Name,
					Image:   "docker.io/rhmessagingqe/cli-proton-python:latest",
					Command: []string{"cli-proton-python-sender"},
					Args: []string{
						"--count",
						strconv.Itoa(s.Messages),
						"--timeout",
						strconv.Itoa(s.Timeout),
						"--broker-url",
						s.Url,
						"--msg-content",
						s.Content,
						"--log-msgs",
						"json",
						"--on-release",
						"retry",
					},
					ImagePullPolicy: "Always",
				},
			},
			RestartPolicy: "Never",
		},
		Status: v1.PodStatus{},
	}
	return a.pythonSender, nil
}
