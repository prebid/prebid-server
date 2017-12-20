package openrtb_ext

import "testing"

func TestBidderKey(t *testing.T) {
	apnKey := HbpbConstantKey.BidderKey(BidderAppnexus, 50)
	if apnKey != "hb_pb_appnexus" {
		t.Errorf("Bad resolved targeting key. Expected hb_pb_appnexus, got %s", apnKey)
	}
}

func TestTruncatedKey(t *testing.T) {
	apnKey := HbpbConstantKey.BidderKey(BidderAppnexus, 8)
	if apnKey != "hb_pb_ap" {
		t.Errorf("Bad truncated targeting key. Expected hb_pb_ap, got %s", apnKey)
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
