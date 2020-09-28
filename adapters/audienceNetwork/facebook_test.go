package audienceNetwork

import (
	"errors"
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/stretchr/testify/assert"
)

type tagInfo struct {
	code        string
	placementID string
	bid         float64
	content     string
	delay       time.Duration
	W           uint64
	H           uint64
	Instl       int8
}

type bidInfo struct {
	partnerID   int
	appID       string
	appSecret   string
	domain      string
	page        string
	publisherID string
	tags        []tagInfo
	deviceIP    string
	deviceUA    string
	buyerUID    string
}

var fbdata bidInfo

type FacebookExt struct {
	PlatformID int `json:"platformid"`
}

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "audienceNetworktest", NewFacebookBidder("test-platform-id", "test-app-secret"))
}

func TestMakeTimeoutNoticeApp(t *testing.T) {
	req := adapters.RequestData{
		Body: []byte(`{"id":"1234","imp":[{"id":"1234"}],"app":{"publisher":{"id":"5678"}}}`),
	}
	fba := NewFacebookBidder("test-platform-id", "test-app-secret")

	tb, ok := fba.(adapters.TimeoutBidder)
	if !ok {
		t.Error("Facebook adapter is not a TimeoutAdapter")
	}

	toReq, err := tb.MakeTimeoutNotification(&req)
	assert.Nil(t, err, "Facebook MakeTimeoutNotification() return an error %v", err)
	expectedUri := "https://www.facebook.com/audiencenetwork/nurl/?partner=test-platform-id&app=5678&auction=1234&ortb_loss_code=2"
	assert.Equal(t, expectedUri, toReq.Uri, "Facebook timeout notification not returning the expected URI.")
}

func TestMakeTimeoutNoticeBadRequest(t *testing.T) {
	req := adapters.RequestData{
		Body: []byte(`{"imp":[{{"id":"1234"}}`),
	}
	fba := NewFacebookBidder("test-platform-id", "test-app-secret")

	tb, ok := fba.(adapters.TimeoutBidder)
	if !ok {
		t.Error("Facebook adapter is not a TimeoutAdapter")
	}

	toReq, err := tb.MakeTimeoutNotification(&req)
	assert.Empty(t, toReq.Uri, "Facebook MakeTimeoutNotification() did not return nil", err)
	assert.NotNil(t, err, "Facebook MakeTimeoutNotification() did not return an error")

}

func TestNewFacebookBidderMissingPlatformID(t *testing.T) {
	result := NewFacebookBidder("", "anyAppSecret")

	expected := &adapters.MisconfiguredBidder{
		Name:  "audienceNetwork",
		Error: errors.New("Audience Network is not configured properly on this Prebid Server deploy. If you believe this should work, contact the company hosting the service and tell them to check their configuration."),
	}

	assert.Equal(t, expected, result)
}

func TestNewFacebookBidderMissingAppSecret(t *testing.T) {
	result := NewFacebookBidder("anyPlatformID", "")

	expected := &adapters.MisconfiguredBidder{
		Name:  "audienceNetwork",
		Error: errors.New("Audience Network is not configured properly on this Prebid Server deploy. If you believe this should work, contact the company hosting the service and tell them to check their configuration."),
	}

	assert.Equal(t, expected, result)
}
