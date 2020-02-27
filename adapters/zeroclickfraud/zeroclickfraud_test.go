package zeroclickfraud

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "zeroclickfraudtest", NewZeroclickfraudBidder("http://{{.Host}}/openrtb2?sid={{.SourceId}}"))
}
