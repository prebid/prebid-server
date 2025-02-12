package bidstack

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidstack, config.Adapter{Endpoint: "http://mock-adserver.url"}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bidstacktest", bidder)
}

func TestPrepareHeaders(t *testing.T) {
	publisherID := "12345"
	expected := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + publisherID},
	}

	actual, err := prepareHeaders(&openrtb2.BidRequest{Imp: []openrtb2.Imp{
		{Ext: []byte(`{"bidder":{"publisherId":"` + publisherID + `"}}`)}},
	})

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestGetBidderExt(t *testing.T) {
	publisherID := "12345"
	expected := openrtb_ext.ImpExtBidstack{PublisherID: publisherID}

	actual, err := getBidderExt(openrtb2.Imp{
		Ext: []byte(`{"bidder":{"publisherId":"` + publisherID + `"}}`),
	})

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
