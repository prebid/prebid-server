package bidstack

import (
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderBidstack, config.Adapter{Endpoint: "http://mock-adserver.url"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "bidstacktest", bidder)
}

func Test_prepareHeaders(t *testing.T) {
	publisherID := "12345"
	happyHeaders := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + publisherID},
	}

	t.Run("happy", func(t *testing.T) {
		expected := happyHeaders
		actual, err := prepareHeaders(&openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidderparams":{"publisherId":"` + publisherID + `"}}}`)})

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("no extensions", func(t *testing.T) {
		expected := (http.Header)(nil)
		actual, err := prepareHeaders(&openrtb2.BidRequest{})

		assert.Equal(t, ErrNoPublisherID, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("no publisher ID key in extensions", func(t *testing.T) {
		expected := (http.Header)(nil)
		actual, err := prepareHeaders(&openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidderparams":{}}}`)})

		assert.Equal(t, ErrNoPublisherID, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("empty publisher ID value in extensions", func(t *testing.T) {
		expected := (http.Header)(nil)
		actual, err := prepareHeaders(&openrtb2.BidRequest{Ext: []byte(`{"prebid":{"bidderparams":{"publisherId":""}}}`)})

		assert.Equal(t, ErrNoPublisherID, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("malformed extensions", func(t *testing.T) {
		expected := (http.Header)(nil)
		actual, err := prepareHeaders(&openrtb2.BidRequest{Ext: []byte(`{`)})

		assert.EqualError(t, err, "extract bidder params: error decoding Request.ext : unexpected end of JSON input")
		assert.Equal(t, expected, actual)
	})
}
