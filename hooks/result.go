package hooks

import (
	"encoding/json"

	"github.com/prebid/openrtb/v17/openrtb2"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
)

type ExecutionResult []json.RawMessage

func (r *ExecutionResult) Push(data json.RawMessage) {
	*r = append(*r, data)
}

func (r *ExecutionResult) EnrichResponse(response *openrtb2.BidResponse) (
	resolvedResponse *openrtb2.BidResponse,
	err error,
) {
	patch := json.RawMessage(`{}`)
	for _, data := range *r {
		patch, err = jsonpatch.MergeMergePatches(patch, data)
		if err != nil {
			return response, err
		}
	}

	response.Ext, err = jsonpatch.MergePatch(response.Ext, patch)

	return response, err
}
