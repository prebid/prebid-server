package lemmadigital

import (
	"testing"

	"github.com/prebid/prebid-server/v2/adapters/adapterstest"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderLemmadigital,
		config.Adapter{
			Endpoint:         "https://{{.Host}}.ads.lemmatechnologies.com/lemma/servad?src=prebid&pid={{.PublisherID}}&aid={{.AdUnit}}",
			ExtraAdapterInfo: "{\"host\":\"sg\"}",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "lemmadigitaltest", bidder)

	// test the dooh config
	usbidder, buildErr := Builder(
		openrtb_ext.BidderLemmadigital,
		config.Adapter{
			Endpoint:         "https://{{.Host}}.ads.lemmatechnologies.com/lemma/servad?src=prebid&pid={{.PublisherID}}&aid={{.AdUnit}}",
			ExtraAdapterInfo: "{\"host\":\"uses\"}",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunSingleJSONBidderTest(t, usbidder, "lemmadigitaltest/exemplary/dooh_video_us", true)
}

func TestExtraInfoDefault(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderLemmadigital,
		config.Adapter{
			Endpoint:         "https://{{.Host}}.ads.lemmatechnologies.com/lemma/servad?src=prebid&pid={{.PublisherID}}&aid={{.AdUnit}}",
			ExtraAdapterInfo: "",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}
}

func TestExtraInfoInvalid(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderLemmadigital,
		config.Adapter{
			Endpoint:         "https://{{.Host}}.ads.lemmatechnologies.com/lemma/servad?src=prebid&pid={{.PublisherID}}&aid={{.AdUnit}}",
			ExtraAdapterInfo: "{\"host\":\"antarctica\"}",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	assert.Error(t, buildErr)
}

func TestDoohHostInvalid(t *testing.T) {
	_, buildErr := Builder(
		openrtb_ext.BidderLemmadigital,
		config.Adapter{
			Endpoint:         "https://{{.Host}}.ads.lemmatechnologies.com/lemma/servad?src=prebid&pid={{.PublisherID}}&aid={{.AdUnit}}",
			ExtraAdapterInfo: "{\"host\":\"uses\",\"dooh_host\":\"usws\"}",
		},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	assert.Error(t, buildErr)
}
