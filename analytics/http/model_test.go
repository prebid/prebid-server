package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestCheckForNil(t *testing.T) {
	now := time.Now()
	_, err := serializeAuctionObject(nil, now)
	assert.Error(t, err)

	_, err = serializeVideoObject(nil, now)
	assert.Error(t, err)

	_, err = serializeSetUIDObject(nil, now)
	assert.Error(t, err)

	_, err = serializeNotificationEvent(nil, now)
	assert.Error(t, err)

	_, err = serializeCookieSyncObject(nil, now)
	assert.Error(t, err)

	_, err = serializeAmpObject(nil, now)
	assert.Error(t, err)
}

func TestSerializeAuctionObject(t *testing.T) {
	data, err := serializeAuctionObject(&analytics.AuctionObject{
		Status: http.StatusOK,
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "some-id",
			},
		},
	}, time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"auction\"")
	assert.Contains(t, string(data), "createdAt")
}

func TestSerializeVideoObject(t *testing.T) {
	data, err := serializeVideoObject(&analytics.VideoObject{
		Status: http.StatusOK,
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{
				ID: "some-id",
			},
		},
	}, time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"video\"")
	assert.Contains(t, string(data), "createdAt")
}

func TestSerializeSetUIDObject(t *testing.T) {
	data, err := serializeSetUIDObject(&analytics.SetUIDObject{
		Status: http.StatusOK,
	}, time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"setuid\"")
	assert.Contains(t, string(data), "createdAt")
}

func TestSerializeNotificationEvent(t *testing.T) {
	data, err := serializeNotificationEvent(&analytics.NotificationEvent{
		Request: &analytics.EventRequest{
			Bidder: "bidder",
		},
		Account: &config.Account{
			ID: "id",
		},
	}, time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"notification\"")
	assert.Contains(t, string(data), "createdAt")
}

func TestSerializeCookieSyncObject(t *testing.T) {
	data, err := serializeCookieSyncObject(&analytics.CookieSyncObject{
		Status:       http.StatusOK,
		BidderStatus: []*analytics.CookieSyncBidder{},
	}, time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"cookiesync\"")
	assert.Contains(t, string(data), "createdAt")
}

func TestSerializeAmpObject(t *testing.T) {
	data, err := serializeAmpObject(&analytics.AmpObject{
		Status: http.StatusOK,
	}, time.Now())
	assert.NoError(t, err)
	assert.Contains(t, string(data), "\"type\":\"amp\"")
	assert.Contains(t, string(data), "createdAt")
}
