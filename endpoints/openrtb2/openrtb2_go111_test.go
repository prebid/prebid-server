// +build go1.11

package openrtb2

import (
	"testing"

	"github.com/buger/jsonparser"
)

func getMessage(t *testing.T, example []byte) []byte {
	// Hack to get tests passing in go1.11, see: https://github.com/golang/go/issues/27275
	// todo: remove this hack when go1.11.1 is released
	if value, _ := jsonparser.GetString(example, "message_go1.11"); value != "" {
		return []byte(value)
	}
	if value, err := jsonparser.GetString(example, "message"); err != nil {
		t.Fatalf("Error parsing root.message from request: %v.", err)
	} else {
		return []byte(value)
	}
	return nil
}
