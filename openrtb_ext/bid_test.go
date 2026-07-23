package openrtb_ext

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBidderKey(t *testing.T) {
	apnKey := PbKey.BidderKey("hb", BidderAppnexus, 50)
	if apnKey != "hb_pb_appnexus" {
		t.Errorf("Bad resolved targeting key. Expected hb_pb_appnexus, got %s", apnKey)
	}
}

func TestTruncatedKey(t *testing.T) {
	apnKey := PbKey.BidderKey("hb", BidderAppnexus, 8)
	if apnKey != "hb_pb_ap" {
		t.Errorf("Bad truncated targeting key. Expected hb_pb_ap, got %s", apnKey)
	}
}

func TestTruncateKey(t *testing.T) {
	testCases := []struct {
		description          string
		givenMaxLength       int
		givenTargetingKey    TargetingKey
		expectedTargetingKey string
	}{
		{
			description:          "Targeting key is smaller than max length, expect targeting key to stay the same",
			givenMaxLength:       15,
			givenTargetingKey:    TargetingKey("_bidder_key"),
			expectedTargetingKey: "hb_bidder_key",
		},
		{
			description:          "Targeting key is larger than max length, expect targeting key to be truncated",
			givenMaxLength:       9,
			givenTargetingKey:    TargetingKey("_bidder_key"),
			expectedTargetingKey: "hb_bidder",
		},
		{
			description:          "Max length isn't greater than zero, expect targeting key to not be truncated",
			givenMaxLength:       0,
			givenTargetingKey:    TargetingKey("_bidder_key"),
			expectedTargetingKey: "hb_bidder_key",
		},
	}

	for _, test := range testCases {
		truncatedKey := test.givenTargetingKey.TruncateKey("hb", test.givenMaxLength)
		assert.Equalf(t, test.expectedTargetingKey, truncatedKey, "The Targeting Key is incorrect: %s\n", test.description)
	}
}

func TestBidParsing(t *testing.T) {
	assertBidParse(t, "banner", BidTypeBanner)
	assertBidParse(t, "video", BidTypeVideo)
	assertBidParse(t, "audio", BidTypeAudio)
	assertBidParse(t, "native", BidTypeNative)
	parsed, err := ParseBidType("unknown")
	if err == nil {
		t.Errorf("ParseBidType did not return the expected error.")
	}
	if parsed != "" {
		t.Errorf("ParseBidType should return an empty string on error. Instead got %s", parsed)
	}
}

func assertBidParse(t *testing.T, s string, bidType BidType) {
	t.Helper()

	parsed, err := ParseBidType(s)
	if err != nil {
		t.Errorf("Bid parsing failed with error: %v", err)
	}
	if parsed != bidType {
		t.Errorf("Bid types did not match. Expected %s, got %s", bidType, parsed)
	}
}
