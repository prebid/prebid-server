package bmtm

import (
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBmtm, config.Adapter{Endpoint: "https://example.com/api/pbs"})
	assert.NoError(t, buildErr, fmt.Sprintf("Builder returned unexpected error: %s", buildErr.Error()))
	adapterstest.RunJSONBidderTest(t, "brightmountainmediatest", bidder)
}
