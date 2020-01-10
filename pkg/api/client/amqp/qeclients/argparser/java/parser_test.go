package java

import (
	"reflect"
	"testing"
)

// TestUrl validates the Url function for the cli-java parser
func TestUrl(t *testing.T) {
	args := Url("amqp://user:pass@127.0.0.1:5672/my/address/name")
	expArgs := []string{"--broker", "amqp://user:pass@127.0.0.1:5672", "--address", "/my/address/name"}
	if !reflect.DeepEqual(expArgs, args) {
		t.Errorf ("URL, got: %v, expected: %v", args, expArgs)
	}
}
