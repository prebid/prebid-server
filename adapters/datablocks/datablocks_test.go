package datablocks

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "datablockstest", NewDatablocksBidder("http://{{.Host}}/openrtb2?sid={{.SourceId}}"))
}
