package java

import (
	"reflect"
	"testing"
)

// TestUrl validates the Url function for the cli-java parser
func TestUrl(t *testing.T) {
	// Valid URL
	args := Url("amqp://user:pass@127.0.0.1:5672/my/address/name")
	expArgs := []string{"--broker", "amqp://user:pass@127.0.0.1:5672", "--address", "/my/address/name"}
	if !reflect.DeepEqual(expArgs, args) {
		t.Errorf ("URL, got: %v, expected: %v", args, expArgs)
	}

	// Invalid URL
	assertPanic(t, "user@pass:host:5672/addresspart1/addresspart2")
	assertPanic(t, "amqp://user:pass@127.0.0.1 5672/my/address/name")
}

// assertPanic ensures that the provided url
// will cause the Url() function to panic
func assertPanic(t *testing.T, url string) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Url() function was expected to panic using url: %s", url)
		}
	}()
	Url(url)
}
