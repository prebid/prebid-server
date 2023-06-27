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

// extMerger tracks a JSON `ext` field within an OpenRTB request. The value of the
// `ext` field is expected to be modified when calling unmarshal on the same object
// and will later be updated when invoking Merge.
type extMerger struct {
	ext      *json.RawMessage // Pointer to the JSON `ext` field.
	snapshot json.RawMessage  // Copy of the original state of the JSON `ext` field.
}

// Track saves a copy of the JSON `ext` field and stores a reference to the extension
// object for comparison later in the Merge call.
func (e *extMerger) Track(ext *json.RawMessage) {
	e.ext = ext
	e.snapshot = sliceutil.Clone(*ext)
}

// Merge applies a JSON merge of the stored extension snapshot on top of the current
// JSON of the tracked extension object.
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
