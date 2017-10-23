package test

import (
	"net/http/httptest"
	"github.com/mxmCherry/openrtb"
	"fmt"
	"strings"
	"testing"
)

/**
 * Represents a scaffolded OpenRTB service.
 */
type OrtbMockService struct {
	Server         *httptest.Server
	LastBidRequest *openrtb.BidRequest
}

/**
 * Produces a map of TagIds, based on a comma separated strings. The map
 * contains the list of tags to bid on.
 */
func BidOnTags(tags string) map[string]bool {
	values := strings.Split(tags, ",")
	set := make(map[string]bool)
	for _, tag := range values {
		set[tag] = true
	}
	return set
}

/**
 * Produces a sample bid based on params given.
 */
func SampleBid(width int, height int, impId string, index int) openrtb.Bid {
	return openrtb.Bid{
		ID:    "Bid-123",
		ImpID: fmt.Sprintf("div-adunit-%d", index),
		Price: 2.1,
		AdM:   "<div>This is an Ad</div>",
		CrID:  "Cr-234",
		W:     uint64(width),
		H:     uint64(height),
	}
}

/**
 * Helper function to assert string equals.
 */
func VerifyStringValue(value string, expected string, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%s expected, got %s", expected, value))
	}
}

/**
 * Helper function to assert Int equals.
 */
func VerifyIntValue(value int, expected int, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%d expected, got %d", expected, value))
	}
}