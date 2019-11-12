package qeclients

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/fgiorgetti/shipshape/pkg/api/client/amqp"
	"github.com/fgiorgetti/shipshape/pkg/framework"
	"github.com/onsi/gomega"
	"io"
	"k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sync"
	"time"
)

// Common implementation for Clients running in containers
type AmqpClientCommon struct {
	Context     framework.ContextData
	Name        string
	Url         string
	Messages    int
	Timeout     int
	Params      []amqp.Param
	Pod         *v1.Pod
	Timedout    bool
	Interrupted bool
	finalResult *amqp.ResultData
	Mutex       sync.Mutex
}

func (a *AmqpClientCommon) Deploy() error {
	_, err := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).Create(a.Pod)
	return err
}

func (a *AmqpClientCommon) Status() amqp.ClientStatus {

	// If user action related condition, do not query Kube
	if a.Timedout {
		return amqp.Timeout
	} else if a.Interrupted {
		return amqp.Interrupted
	}

	pod, err := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).Get(a.Pod.Name, v12.GetOptions{})
	gomega.Expect(err).To(gomega.BeNil())
	gomega.Expect(pod).NotTo(gomega.BeNil())

	switch pod.Status.Phase {
	case v1.PodPending:
		return amqp.Starting
	case v1.PodRunning:
		return amqp.Running
	case v1.PodSucceeded:
		return amqp.Success
	case v1.PodFailed:
		return amqp.Error
	case v1.PodUnknown:
		return amqp.Unknown
	default:
		return amqp.Unknown
	}
}

func (a *AmqpClientCommon) Running() bool {
	return a.Status() == amqp.Starting || a.Status() == amqp.Running
}

func (a *AmqpClientCommon) Interrupt() {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	if a.Interrupted {
		return
	}

	timeout := int64(amqp.TimeoutInterruptSecs)
	err := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).Delete(a.Pod.Name, &v12.DeleteOptions{GracePeriodSeconds: &timeout})
	gomega.Expect(err).To(gomega.BeNil())

	a.Interrupted = true
}

func (a *AmqpClientCommon) Result() amqp.ResultData {

	// If client is not longer running and finalResult already set, return it
	if a.finalResult != nil {
		return *a.finalResult
	}

	request := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).GetLogs(a.Pod.Name, &v1.PodLogOptions{})
	logs, err := request.Stream()
	gomega.Expect(err).To(gomega.BeNil())

	// Close when done reading
	defer logs.Close()

	// Reading logs into buf
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, logs)
	gomega.Expect(err).To(gomega.BeNil())

	// Allows reading line by line
	reader := bufio.NewReader(buf)

	// Unmarshalling message dict
	var messages []MessageDict

	// Iterate through lines
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		gomega.Expect(err).To(gomega.BeNil())

		var msg MessageDict
		err = json.Unmarshal([]byte(line), &msg)
		gomega.Expect(err).To(gomega.BeNil())
		messages = append(messages, msg)
	}

	// Generating result data
	result := amqp.ResultData{
		Messages:  make([]amqp.Message, 0),
		Delivered: len(messages),
	}
	for _, message := range messages {
		result.Messages = append(result.Messages, message.ToMessage())
	}

	// Locking to set finalResults
	a.Mutex.Lock()
	defer a.Mutex.Unlock()
	if !a.Running() && a.finalResult == nil {
		a.finalResult = &result
	}

	return result
}

// Wait Waits for client to complete running (successfully or not), until pre-defined client's timeout.
func (a *AmqpClientCommon) Wait() amqp.ClientStatus {
	return a.WaitFor(a.Timeout)
}

// WaitFor Waits for client to complete running (successfully or not), until given timeout.
func (a *AmqpClientCommon) WaitFor(secs int) amqp.ClientStatus {
	return a.WaitForStatus(secs, amqp.Success, amqp.Error, amqp.Timeout, amqp.Interrupted)
}

// WaitForStatus Waits till client status matches one of the given statuses or till it times out
func (a *AmqpClientCommon) WaitForStatus(secs int, statuses ...amqp.ClientStatus) amqp.ClientStatus {
	// Wait timeout
	timeout := time.Duration(secs) * time.Second

	// Channel to notify when status
	result := make(chan amqp.ClientStatus, 1)
	go func() {
		for t := time.Now(); time.Since(t) < timeout; time.Sleep(amqp.Poll) {
			curStatus := a.Status()

			if amqp.ClientStatusIn(curStatus, statuses...) {
				result <- curStatus
				return
			}
		}
	}()

	select {
	case res := <-result:
		return res
	case <-time.After(time.Duration(secs) * time.Second):
		return amqp.Timeout
	}
}
