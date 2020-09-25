package privacy

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

func TestNilWriter(t *testing.T) {
	request := &openrtb.BidRequest{
		ID:  "anyID",
		Ext: json.RawMessage(`{"anyJson":"anyValue"}`),
	}
	expectedRequest := &openrtb.BidRequest{
		ID:  "anyID",
		Ext: json.RawMessage(`{"anyJson":"anyValue"}`),
	}

	nilWriter := &NilPolicyWriter{}
	nilWriter.Write(request)

	assert.Equal(t, expectedRequest, request)
}
