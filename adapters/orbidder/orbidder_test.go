package orbidder

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUnmarshalOrbidderExtImp(t *testing.T) {
	ext := json.RawMessage(`{"accountId":"orbidder-test", "placementId":"center-banner", "bidfloor": 0.1}`)
	impExt := new(openrtb_ext.ExtImpOrbidder)

	assert.NoError(t, json.Unmarshal(ext, impExt))
	assert.Equal(t, &openrtb_ext.ExtImpOrbidder{
		AccountId:   "orbidder-test",
		PlacementId: "center-banner",
		BidFloor:    0.1,
	}, impExt)
}
