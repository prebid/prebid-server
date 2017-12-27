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
