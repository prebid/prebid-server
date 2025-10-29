package revx_test

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/revx"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestMakeBids_NilExternalRequest(t *testing.T) {
	// Arrange
	bidder, buildErr := revx.Builder(openrtb_ext.BidderRevX, config.Adapter{
		Endpoint: "prebid-use.atomex.net/ag=PUB123",
	}, config.Server{
		ExternalUrl: "http://hosturl.com",
		GvlID:       375,
		DataCenter:  "2",
	})

	var internalReq *openrtb2.BidRequest
	var response *adapters.ResponseData
	if buildErr != nil {
		t.Logf("RevX Builder created successfully: %+v", bidder)
	}
	// Act
	bidderResponse, errs := bidder.MakeBids(internalReq, nil, response)

	// Assert
	assert.Nil(t, bidderResponse, "Expected bidderResponse to be nil when externalRequest is nil")
	assert.Nil(t, errs, "Expected errors to be nil when externalRequest is nil")
}
