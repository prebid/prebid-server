package nobid

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	nobidAdapter := NewNoBidBidder("http://ads.servenobid.com/ortb_adreq?tek=pbs")
	adapterstest.RunJSONBidderTest(t, "nobidtest", nobidAdapter)
}
