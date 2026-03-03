package appnexus

import (
	"net/url"
	"testing"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderAppNexus, _ := bidder.(*adapter)
	bidderAppNexus.randomGenerator = FakeRandomNumberGenerator{Number: 10}

	adapterstest.RunJSONBidderTest(t, "appnexustest", bidder)
}

func TestAppendMemberID(t *testing.T) {
	uri, err := url.Parse("http://ib.adnxs.com/openrtb2?query_param=true")
	require.NoError(t, err, "Failed to parse URI with query string")

	uriWithMember := appendMemberId(*uri, "102")
	expected := "http://ib.adnxs.com/openrtb2?member_id=102&query_param=true"
	assert.Equal(t, expected, uriWithMember.String(), "Failed to append member id to URI with query string")
}

func TestBuilderWithPlatformID(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAppnexus, config.Adapter{
		Endpoint: "http://ib.adnxs.com/openrtb2", PlatformID: "3"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.NoError(t, buildErr)
	require.NotNil(t, bidder)
	assert.Equal(t, 3, (*bidder.(*adapter)).hbSource)
}

type FakeRandomNumberGenerator struct {
	Number int64
}

func (f FakeRandomNumberGenerator) GenerateInt63() int64 {
	return f.Number
}
func (f FakeRandomNumberGenerator) Intn(n int) int {
	return int(f.Number)
}
