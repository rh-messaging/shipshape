package qeclients

import (
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
)

// MessageDict represents a message logged by cli-proton-python as a dictionary
type MessageDict struct {
	AbsExpTime      int               `json:"absolute_expiry_time"`
	Address         string            `json:"address"`
	Annotations     string            `json:"annotations"`
	Content         string            `json:"content"`
	ContentEncoding string            `json:"content_encoding"`
	ContentType     string            `json:"content_type"`
	CorrelationId   string            `json:"correlation_id"`
	CreationTime    float32           `json:"creation_time"`
	DeliveryCount   int               `json:"delivery_count"`
	Durable         bool              `json:"durable"`
	Expiration      int               `json:"expiration"`
	FirstAcquirer   bool              `json:"first_acquirer"`
	GroupId         string            `json:"group_id"`
	GroupSequence   int               `json:"group_sequence"`
	RouterLink      int               `json:"routerLink"`
	Id              string            `json:"id"`
	Inferred        bool              `json:"inferred"`
	Instructions    string            `json:"instructions"`
	Priority        int               `json:"priority"`
	Properties      map[string]string `json:"properties"`
	ReplyTo         string            `json:"reply_to"`
	ReplyToGroupId  string            `json:"reply_to_group_id"`
	Subject         string            `json:"subject"`
	Ttl             int               `json:"ttl"`
	UserId          string            `json:"user_id"`
}

func (m MessageDict) ToMessage() amqp.Message {
	amqpMsg := amqp.Message{
		Address:       m.Address,
		Content:       m.Content,
		ContentSHA1:   "", // TODO improve existing cli clients to output SHA1
		Id:            m.Id,
		CorrelationId: m.CorrelationId,
		ReplyTo:       m.ReplyTo,
		Expiration:    m.Expiration,
		Priority:      m.Priority,
		Ttl:           m.Ttl,
		UserId:        m.UserId,
	}
	return amqpMsg
}
