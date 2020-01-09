package qeclients

import (
	"bufio"
	"bytes"
	"encoding/json"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"io"
	v1 "k8s.io/api/core/v1"
)

type AmqpQEClientCommon struct {
	amqp.AmqpClientCommon
	Implementation AmqpQEClientImpl
}

// Result common implementation for QE Clients
func (a *AmqpQEClientCommon) Result() amqp.ResultData {

	// If client is not longer running and finalResult already set, return it
	if a.FinalResult != nil {
		return *a.FinalResult
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

	// Iterate through lines\
	outer: for {
		var line, partLine []byte
		var fullLine = true

		// ReadLine may not return the full line when it exceeds 4096 bytes,
		// so we need to keep reading till fullLine is false or eof is found
		for fullLine {
			partLine, fullLine, err = reader.ReadLine()
			line = append(line, partLine...)
			if err == io.EOF {
				break outer
			}
			gomega.Expect(err).To(gomega.BeNil())
		}

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
	if !a.Running() && a.FinalResult == nil {
		a.FinalResult = &result
	}

	return result
}
