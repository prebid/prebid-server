package orbidder

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
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

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderOrbidder, config.Adapter{
		Endpoint: "https://orbidder-test"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "orbiddertest", bidder)
}
