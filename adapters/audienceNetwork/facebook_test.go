package audienceNetwork

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
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
	bidder, buildErr := Builder(openrtb_ext.BidderAudienceNetwork, config.Adapter{
		Endpoint:   "https://an.facebook.com/placementbid.ortb",
		PlatformID: "test-platform-id",
		AppSecret:  "test-app-secret",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "audienceNetworktest", bidder)
}

func TestMakeTimeoutNoticeApp(t *testing.T) {
	req := adapters.RequestData{
		Body: []byte(`{"id":"1234","imp":[{"id":"1234"}],"app":{"publisher":{"id":"5678"}}}`),
	}
	bidder, buildErr := Builder(openrtb_ext.BidderAudienceNetwork, config.Adapter{
		Endpoint:   "https://an.facebook.com/placementbid.ortb",
		PlatformID: "test-platform-id",
		AppSecret:  "test-app-secret",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tb, ok := bidder.(adapters.TimeoutBidder)
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
	bidder, buildErr := Builder(openrtb_ext.BidderAudienceNetwork, config.Adapter{
		Endpoint:   "https://an.facebook.com/placementbid.ortb",
		PlatformID: "test-platform-id",
		AppSecret:  "test-app-secret",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	tb, ok := bidder.(adapters.TimeoutBidder)
	if !ok {
		t.Error("Facebook adapter is not a TimeoutAdapter")
	}

	toReq, err := tb.MakeTimeoutNotification(&req)
	assert.Empty(t, toReq.Uri, "Facebook MakeTimeoutNotification() did not return nil", err)
	assert.NotNil(t, err, "Facebook MakeTimeoutNotification() did not return an error")

}

func TestNewFacebookBidderMissingPlatformID(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderAudienceNetwork, config.Adapter{
		Endpoint:  "https://an.facebook.com/placementbid.ortb",
		AppSecret: "test-app-secret",
	})

	assert.Empty(t, bidder)
	assert.EqualError(t, err, "PartnerID is not configured. Did you set adapters.facebook.platform_id in the app config?")
}

func TestNewFacebookBidderMissingAppSecret(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderAudienceNetwork, config.Adapter{
		Endpoint:   "https://an.facebook.com/placementbid.ortb",
		PlatformID: "test-platform-id",
	})

	assert.Empty(t, bidder)
	assert.EqualError(t, err, "AppSecret is not configured. Did you set adapters.facebook.app_secret in the app config?")
}
