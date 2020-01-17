package amqp

import (
	"time"
)

const (
	TimeoutDefaultSecs   int = 60
	TimeoutInterruptSecs int = 60
	Poll                     = time.Duration(5) * time.Second
)

type Client interface {
	Deploy() error
	Status() ClientStatus
	Running() bool
	Interrupt()
	Wait() ClientStatus
	Result() ResultData
}

//
// Data types for AMQP clients
//
type Param struct {
	Name, Value string
}

type Message struct {
	Address       string
	Content       string
	ContentSHA1   string
	Id            string
	CorrelationId string
	ReplyTo       string
	Expiration    int
	Priority      int
	Ttl           int
	UserId        string
}

type ResultData struct {
	Messages  []Message
	Delivered int
	Released  int
	Modified  int
}

type ClientStatus int

const (
	Starting ClientStatus = iota
	Running
	Success
	Error
	Timeout
	Interrupted
	Unknown
)

// ClientStatusIn returns true if the given "status" is present in the status slice
func ClientStatusIn(status ClientStatus, statuses ...ClientStatus) bool {
	for _, v := range statuses {
		if v == status {
			return true
		}
	}
	return false
}
