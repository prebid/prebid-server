// +build !go1.11

package openrtb2

import (
	"testing"

	"github.com/buger/jsonparser"
)

func getMessage(t *testing.T, example []byte) []byte {
	if value, err := jsonparser.GetString(example, "message"); err != nil {
		t.Fatalf("Error parsing root.message from request: %v.", err)
	} else {
		return []byte(value)
	}
	return nil
}
