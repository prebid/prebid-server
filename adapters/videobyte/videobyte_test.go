package videobyte

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "videobytetest", &VideoByteAdapter{endpoint: "https://mock.videobyte.com"})
}
