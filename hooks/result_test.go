package hooks

import (
	"encoding/json"
	"testing"

	"github.com/golang/glog"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestPush(t *testing.T) {
	expectedResult := ExecutionResult{
		[]byte(`{"stage": "entrypoint", "result": "success"}`),
		[]byte(`{"stage": "rawauction", "result": "failed"}`),
	}

	result := ExecutionResult{}
	result.Push([]byte(`{"stage": "entrypoint", "result": "success"}`))
	result.Push([]byte(`{"stage": "rawauction", "result": "failed"}`))

	assert.Equal(t, expectedResult, result)
}

func TestEnrichResponse(t *testing.T) {
	bidResponse := &openrtb2.BidResponse{ID: "foo", Ext: []byte(`{"ext": {"prebid": {"foo": "bar"}}}`)}
	expectedResponse := json.RawMessage(`{"ext":{"prebid":{"foo":"bar","modules":{"entrypoint":"success","rawauction":"failed"}}}}`)

	executionResult := ExecutionResult{
		[]byte(`{"ext": {"prebid": {"modules": {"entrypoint": "success"}}}}`),
		[]byte(`{"ext": {"prebid": {"modules": {"rawauction": "failed"}}}}`),
	}

	if resolvedResponse, err := executionResult.EnrichResponse(bidResponse); err == nil {
		bidResponse = resolvedResponse
	} else {
		glog.Errorf("Failed to enrich BidResponse with hook debug information: %s", err)
		t.Fatalf("Failed to enrich BidResponse with hook debug information: %s", err)
	}

	assert.Equal(t, expectedResponse, bidResponse.Ext)
}
