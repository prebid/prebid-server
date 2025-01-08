package agma

import (
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestCheckForNil(t *testing.T) {
	code := "test"
	_, err := serializeAnayltics(nil, EventTypeAuction, code, time.Now())
	assert.Error(t, err)
}

func TestSerializeAuctionObject(t *testing.T) {
	data, err := serializeAnayltics(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
		},
	}, EventTypeAuction, "test", time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"auction\"")
}

func TestSerializeVideoObject(t *testing.T) {
	data, err := serializeAnayltics(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
		},
	}, EventTypeVideo, "test", time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"video\"")
}

func TestSerializeAmpObject(t *testing.T) {
	data, err := serializeAnayltics(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
		},
	}, EventTypeAmp, "test", time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"amp\"")
}
