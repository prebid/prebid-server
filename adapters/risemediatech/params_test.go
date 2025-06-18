package risemediatech

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	valid := `{"bidfloor": 1.2, "mimes": ["video/mp4"], "minduration": 5}`
	var extImp openrtb_ext.ExtImpRiseMediaTech
	if err := json.Unmarshal([]byte(valid), &extImp); err != nil {
		t.Errorf("Valid params should not throw error: %v", err)
	}
}
