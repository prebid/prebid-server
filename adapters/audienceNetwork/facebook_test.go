package audienceNetwork

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/adapters/adapterstest"
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
	adapterstest.RunJSONBidderTest(t, "audienceNetworktest", NewFacebookBidder(nil, "test-platform-id", "test-app-secret"))
}
