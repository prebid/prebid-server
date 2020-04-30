package adkernel

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "adkerneltest", NewAdkernelAdapter("http://{{.Host}}/hb?zone={{.ZoneID}}"))
}
