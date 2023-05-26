package firstpartydata

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/prebid/prebid-server/util/sliceutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
)

var (
	ErrBadRequest = fmt.Errorf("invalid request ext")
	ErrBadFPD     = fmt.Errorf("invalid first party data ext")
)

// extMerger assists in tracking and merging changes to extension json after
// unmarshalling override json on top of an existing OpenRTB object.
type extMerger struct {
	// wow! It's really tricky to understand e.ext will change after unmarshal one level upper because it's a pointer.
	// this is applicable for all sub-extensions, maybe add a comment about it?
	ext      *json.RawMessage
	snapshot json.RawMessage
}

// Track saves a copy of the extension json and stores a reference to the extension
// object for comparison later in the Merge call.
func (e *extMerger) Track(ext *json.RawMessage) {
	e.ext = ext
	e.snapshot = sliceutil.Clone(*ext)
}

// Merge applies a json merge of the stored extension snapshot on top of the current
// json of the tracked extension object.
func (e extMerger) Merge() error {
	if e.ext == nil {
		return nil
	}

	if len(e.snapshot) == 0 {
		return nil
	}

	if len(*e.ext) == 0 {
		*e.ext = e.snapshot
		return nil
	}

	merged, err := jsonpatch.MergePatch(e.snapshot, *e.ext)
	if err != nil {
		if errors.Is(err, jsonpatch.ErrBadJSONDoc) {
			return ErrBadRequest
		} else if errors.Is(err, jsonpatch.ErrBadJSONPatch) {
			return ErrBadFPD
		}
		return err
	}

	*e.ext = merged
	return nil
}
