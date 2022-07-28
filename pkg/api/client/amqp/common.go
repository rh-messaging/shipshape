package amqp

import (
	"context"
	"sync"
	"time"

	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Common partial implementation for Clients running in Pods/Containers
// Result() must be implemented by concrete client implementations
type AmqpClientCommon struct {
	Context     framework.ContextData
	Name        string
	Url         string
	Timeout     int
	Params      []Param
	Pod         *v1.Pod
	TimedOut    bool
	Interrupted bool
	FinalResult *ResultData
	Mutex       sync.Mutex
}

func (a *AmqpClientCommon) Deploy() error {
	_, err := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).Create(context.TODO(), a.Pod, metav1.CreateOptions{})
	return err
}

func (a *AmqpClientCommon) Status() ClientStatus {

	// If user action related condition, do not query Kube
	if a.TimedOut {
		return Timeout
	} else if a.Interrupted {
		return Interrupted
	}

	pod, err := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).Get(context.TODO(), a.Pod.Name, metav1.GetOptions{})
	// Errors might happen and we cannot simply assert it will be nil,
	// or it will cause a panic.
	if err != nil {
		log.Logf("error getting pod status: %s", err)
		return Unknown
	}
	if pod == nil {
		log.Logf("error getting pod status (pod is nil): %s", err)
		return Unknown
	}

	switch pod.Status.Phase {
	case v1.PodPending:
		return Starting
	case v1.PodRunning:
		return Running
	case v1.PodSucceeded:
		return Success
	case v1.PodFailed:
		return Error
	case v1.PodUnknown:
		return Unknown
	default:
		return Unknown
	}
}

func (a *AmqpClientCommon) Running() bool {
	return a.Status() == Starting || a.Status() == Running
}

func (a *AmqpClientCommon) Interrupt() {
	a.Mutex.Lock()
	defer a.Mutex.Unlock()

	if a.Interrupted {
		return
	}

	timeout := int64(TimeoutInterruptSecs)
	err := a.Context.Clients.KubeClient.CoreV1().Pods(a.Context.Namespace).Delete(context.TODO(), a.Pod.Name, metav1.DeleteOptions{GracePeriodSeconds: &timeout})
	gomega.Expect(err).To(gomega.BeNil())

	a.Interrupted = true
}

// Wait Waits for client to complete running (successfully or not), until pre-defined client's timeout.
func (a *AmqpClientCommon) Wait() ClientStatus {
	return a.WaitFor(a.Timeout)
}

// WaitFor Waits for client to complete running (successfully or not), until given timeout.
func (a *AmqpClientCommon) WaitFor(secs int) ClientStatus {
	return a.WaitForStatus(secs, Success, Error, Timeout, Interrupted)
}

// WaitForStatus Waits till client status matches one of the given statuses or till it times out
func (a *AmqpClientCommon) WaitForStatus(secs int, statuses ...ClientStatus) ClientStatus {
	// Wait timeout
	timeout := time.Duration(secs) * time.Second

	// Channel to notify when status
	result := make(chan ClientStatus, 1)
	go func() {
		for t := time.Now(); time.Since(t) < timeout; time.Sleep(Poll) {
			curStatus := a.Status()

			if ClientStatusIn(curStatus, statuses...) {
				result <- curStatus
				return
			}
		}
	}()

	select {
	case res := <-result:
		return res
	case <-time.After(time.Duration(secs) * time.Second):
		return Timeout
	}
}
