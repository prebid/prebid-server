package adfusion

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderAdfusion,
		config.Adapter{Endpoint: "http://test.adfusion.example/auction"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if err != nil {
		t.Fatalf("Builder returned unexpected error %v", err)
	}

	adapterstest.RunJSONBidderTest(t, "adfusiontest", bidder)
}

func TestAppendAccountIdToExtPreservesUnrelatedNumbers(t *testing.T) {
	request := &openrtb2.BidRequest{
		Ext: []byte(`{"data":{"lineItemId":9007199254740993}}`),
	}

	err := appendAccountIdToExt(request, 55)

	assert.NoError(t, err)
	assert.Contains(t, string(request.Ext), `"lineItemId":9007199254740993`)
}
