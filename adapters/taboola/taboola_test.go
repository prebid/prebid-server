package taboola

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTaboola, config.Adapter{
		Endpoint: "http://{{.MediaType}}.whatever.com/{{.GvlID}}/{{.PublisherID}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 12, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "taboolatest", bidder)
}

func TestMakeRequestExt(t *testing.T) {
	t.Run("merges pageType into existing ext without dropping other fields", func(t *testing.T) {
		existingExt := json.RawMessage(`{"prebid":{"integration":"pbjs"}}`)

		result, err := makeRequestExt("article", existingExt)

		assert.NoError(t, err)
		assert.JSONEq(t, `{"prebid":{"integration":"pbjs"},"pageType":"article"}`, string(result))
	})

	t.Run("sets pageType when existing ext is empty", func(t *testing.T) {
		result, err := makeRequestExt("article", nil)

		assert.NoError(t, err)
		assert.JSONEq(t, `{"pageType":"article"}`, string(result))
	})

	t.Run("returns error when existing ext is invalid JSON", func(t *testing.T) {
		invalidExt := json.RawMessage(`not-a-json-object`)

		result, err := makeRequestExt("article", invalidExt)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not unmarshal request ext")
		assert.Nil(t, result)
	})
}

func TestEmptyExternalUrl(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderTaboola, config.Adapter{
		Endpoint: "http://whatever.com"}, config.Server{})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	bidderTaboola := bidder.(*adapter)

	assert.Equal(t, "", bidderTaboola.gvlID)
}
