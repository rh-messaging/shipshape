package qeclients

//
// Parser for QEClients exclusive for common arguments between sender and receiver
//

import (
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp/qeclients/argparser/java"
	"strconv"
)

// parseUrl Parses URL defined at the AmqpQEClientCommon instance
// and returns arguments list based on the implementation being used
func parseUrl(client *AmqpQEClientCommon) []string {
	switch client.Implementation {
	case Python:
		return []string{"--broker-url", client.Url}
	default:
		return java.Url(client.Url)
	}
}

// parseCount returns the arguments for message count
func parseCount(count int) []string {
	return []string{"--count", strconv.Itoa(count)}
}

// parseTimeout returns the arguments for timeout
func parseTimeout(secs int) []string {
	return []string{"--timeout", strconv.Itoa(secs)}
}
