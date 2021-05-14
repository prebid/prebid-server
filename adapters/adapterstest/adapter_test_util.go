package adapterstest

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"net/http"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

// OrtbMockService Represents a scaffolded OpenRTB service.
type OrtbMockService struct {
	Server          *httptest.Server
	LastBidRequest  *openrtb2.BidRequest
	LastHttpRequest *http.Request
}

// BidOnTags Produces a map of TagIds, based on a comma separated strings. The map
// contains the list of tags to bid on.
func BidOnTags(tags string) map[string]bool {
	values := strings.Split(tags, ",")
	set := make(map[string]bool)
	for _, tag := range values {
		set[tag] = true
	}
	return set
}

// SampleBid Produces a sample bid based on params given.
func SampleBid(width *int64, height *int64, impId string, index int) openrtb2.Bid {
	return openrtb2.Bid{
		ID:    "Bid-123",
		ImpID: fmt.Sprintf("div-adunit-%d", index),
		Price: 2.1,
		AdM:   "<div>This is an Ad</div>",
		CrID:  "Cr-234",
		W:     *width,
		H:     *height,
	}
}

// VerifyStringValue Helper function to assert string equals.
func VerifyStringValue(value string, expected string, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%s expected, got %s", expected, value))
	}
}

// VerifyIntValue Helper function to assert Int equals.
func VerifyIntValue(value int, expected int, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%d expected, got %d", expected, value))
	}
}

// VerifyBoolValue Helper function to assert bool equals.
func VerifyBoolValue(value bool, expected bool, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%v expected, got %v", expected, value))
	}
}

// VerifyBannerSize helper function to assert banner size
func VerifyBannerSize(banner *openrtb2.Banner, expectedWidth int, expectedHeight int, t *testing.T) {
	VerifyIntValue(int(*(banner.W)), expectedWidth, t)
	VerifyIntValue(int(*(banner.H)), expectedHeight, t)
}
