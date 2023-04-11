package appnexus

import (
	"testing"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
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

func TestMemberQueryParam(t *testing.T) {
	uriWithMember := appendMemberId("http://ib.adnxs.com/openrtb2?query_param=true", "102")
	expected := "http://ib.adnxs.com/openrtb2?query_param=true&member_id=102"
	if uriWithMember != expected {
		t.Errorf("appendMemberId() failed on URI with query string. Expected %s, got %s", expected, uriWithMember)
	}
}

// fakerandomNumberGenerator
type FakeRandomNumberGenerator struct {
	Number int64
}

func (f FakeRandomNumberGenerator) GenerateInt63() int64 {
	return f.Number
}
