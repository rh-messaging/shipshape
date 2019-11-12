package qeclients

import (
	"github.com/fgiorgetti/shipshape/pkg/api/client/amqp"
	"github.com/fgiorgetti/shipshape/pkg/framework"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
	"sync"
)

// AmqpPythonReceiver cli-proton-python-receiver
type AmqpPythonReceiver struct {
	AmqpClientCommon
}

// AmqpPythonReceiverBuilder can be used to produce a cli-proton-python-receiver
type AmqpPythonReceiverBuilder struct {
	pythonReceiver *AmqpPythonReceiver
}

func (a *AmqpPythonReceiverBuilder) New(name string, data framework.ContextData, url string) amqp.ReceiverBuilder {
	a.pythonReceiver = &AmqpPythonReceiver{
		AmqpClientCommon: AmqpClientCommon{
			Context: data,
			Name:    name,
			Url:     url,
			Mutex:   sync.Mutex{},
		}}
	return a
}

func (a *AmqpPythonReceiverBuilder) Messages(count int) amqp.ReceiverBuilder {
	a.pythonReceiver.Messages = count
	return a
}

func (a *AmqpPythonReceiverBuilder) Timeout(timeout int) amqp.ReceiverBuilder {
	a.pythonReceiver.Timeout = timeout
	return a
}

func (a *AmqpPythonReceiverBuilder) Param(name string, value string) amqp.ReceiverBuilder {
	if a.pythonReceiver.Params == nil {
		a.pythonReceiver.Params = make([]amqp.Param, 0)
	}
	a.pythonReceiver.Params = append(a.pythonReceiver.Params, amqp.Param{Name: name, Value: value})
	return a
}

func (a *AmqpPythonReceiverBuilder) Build() (amqp.Client, error) {
	//TODO Add some validation to ensure all required have been populated
	// and return an error instead of nil
	s := a.pythonReceiver
	s.Pod = &v1.Pod{
		ObjectMeta: v12.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Context.Namespace,
			Labels: map[string]string{
				"amqp-client": "cli-proton-python-receiver",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    s.Name,
					Image:   "docker.io/rhmessagingqe/cli-proton-python:latest",
					Command: []string{"cli-proton-python-receiver"},
					Args: []string{
						"--count",
						strconv.Itoa(s.Messages),
						"--timeout",
						strconv.Itoa(s.Timeout),
						"--broker-url",
						s.Url,
						"--log-msgs",
						"json",
					},
					ImagePullPolicy: "Always",
				},
			},
			RestartPolicy: "Never",
		},
		Status: v1.PodStatus{},
	}
	return a.pythonReceiver, nil
}
