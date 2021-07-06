package privacy

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestNilWriter(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID:  "anyID",
		Ext: json.RawMessage(`{"anyJson":"anyValue"}`),
	}
	expectedRequest := &openrtb2.BidRequest{
		ID:  "anyID",
		Ext: json.RawMessage(`{"anyJson":"anyValue"}`),
	}

	nilWriter := &NilPolicyWriter{}
	nilWriter.Write(request)

	assert.Equal(t, expectedRequest, request)
}
