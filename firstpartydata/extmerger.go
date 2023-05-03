package firstpartydata

import (
	"encoding/json"

	"github.com/prebid/prebid-server/util/sliceutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
)

type extMerger struct {
	ext      *json.RawMessage
	snapshot json.RawMessage
}

func (e *extMerger) Track(ext *json.RawMessage) {
	e.ext = ext
	e.snapshot = sliceutil.Clone(*ext)
}

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
		return err
	}

	*e.ext = merged
	return nil
}
