package merge

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/ortb"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
)

func User(v *openrtb2.User, overrideJSON json.RawMessage) error {
	*v = *ortb.CloneUser(v)

	// Track EXTs
	// It's not necessary to track `ext` fields in array items because the array
	// items will be replaced entirely with the override JSON, so no merge is required.
	var ext, extGeo extMerger
	ext.Track(&v.Ext)
	if v.Geo != nil {
		extGeo.Track(&v.Geo.Ext)
	}

	// Merge
	if err := jsonutil.Unmarshal(overrideJSON, &v); err != nil {
		return err
	}

	// Merge EXTs
	if err := ext.Merge(); err != nil {
		return err
	}
	if err := extGeo.Merge(); err != nil {
		return err
	}

	return nil
}
