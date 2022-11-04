package hookexecution

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*func TestPush(t *testing.T) {
	expectedResult := ExecutionResult{
		[]byte(`{"stage": "entrypoint", "result": "success"}`),
		[]byte(`{"stage": "rawauction", "result": "failed"}`),
	}

	result := ExecutionResult{}
	result.Push([]byte(`{"stage": "entrypoint", "result": "success"}`))
	result.Push([]byte(`{"stage": "rawauction", "result": "failed"}`))

	assert.Equal(t, expectedResult, result)
}*/

func TestEnrichResponse(t *testing.T) {
	bidResponse := &openrtb2.BidResponse{ID: "foo", Ext: []byte(`{"ext": {"prebid": {"foo": "bar"}}}`)}
	expectedResponse := json.RawMessage(`{"ext":{"prebid":{"foo":"bar","modules":{"entrypoint":"success","rawauction":"failed"}}}}`)

	stageOutcome := StageOutcome{
		Entity: hookstage.EntityHttpRequest,
		Stage:  hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				InvocationResults: []*HookOutcome{
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "foo"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate)},
						Errors:        nil,
						Warnings:      nil,
					},
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: param.foo, mutation type: %s", hookstage.MutationUpdate)},
						Errors:        nil,
						Warnings:      nil,
					},
				},
			},
			{
				InvocationResults: []*HookOutcome{
					{
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "baz"},
						Status:        StatusSuccess,
						Action:        ActionUpdate,
						Message:       "",
						DebugMessages: []string{
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
							fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
						},
						Errors:   nil,
						Warnings: nil,
					},
				},
			},
		},
	}

	err := EnrichResponse(bidResponse, []StageOutcome{stageOutcome})
	require.NoError(t, err, "Failed to enrich BidResponse with hook debug information: %s", err)
	assert.Equal(t, expectedResponse, bidResponse.Ext)
}
