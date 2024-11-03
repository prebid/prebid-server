package mediasquare

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestPtrInt8ToBool(t *testing.T) {
	// map[tests-inputs]test-expected-results
	tests := map[int8]bool{
		0:  false,
		1:  true,
		42: false,
	}
	for value, expected := range tests {
		assert.Equal(t, expected, ptrInt8ToBool(&value), fmt.Sprintf("ptrInt8ToBool >> value: (int8)=%d.", value))
	}
	assert.Equal(t, false, ptrInt8ToBool(nil), "ptrInt8ToBool >> value(nil)")
}

func TestMethodsType(t *testing.T) {
	tests := []struct {
		// tests inputs
		resp msqResponseBids
		// tests expected-results
		bidType openrtb_ext.BidType
		mType   openrtb2.MarkupType
	}{
		{
			resp:    msqResponseBids{Native: &msqResponseBidsNative{ClickUrl: "not-nil"}},
			bidType: "native",
			mType:   openrtb2.MarkupNative,
		},
		{
			resp:    msqResponseBids{Video: &msqResponseBidsVideo{Xml: "not-nil"}},
			bidType: "video",
			mType:   openrtb2.MarkupVideo,
		},
		{
			resp:    msqResponseBids{ID: "not-nil"},
			bidType: "banner",
			mType:   openrtb2.MarkupBanner,
		},
	}
	for testIndex, test := range tests {
		assert.Equal(t, test.bidType, test.resp.bidType(), "bidType >> testIndex:", testIndex)
		assert.Equal(t, test.mType, test.resp.mType(), "mType >> testIndex:", testIndex)
	}
}

func TestLoadExtBid(t *testing.T) {
	tests := []struct {
		// tests inputs
		resp msqResponseBids
		// tests expected-results
		extBid openrtb_ext.ExtBid
		isOk   bool
	}{
		{
			resp:   msqResponseBids{},
			extBid: openrtb_ext.ExtBid{DSA: nil, Prebid: nil},
			isOk:   true,
		},
		{
			resp:   msqResponseBids{Dsa: openrtb_ext.ExtBidDSA{Behalf: "behalf"}},
			extBid: openrtb_ext.ExtBid{DSA: &openrtb_ext.ExtBidDSA{Behalf: "behalf"}},
			isOk:   true,
		},
		{
			resp:   msqResponseBids{Dsa: "lol"},
			extBid: openrtb_ext.ExtBid{},
			isOk:   false,
		},
		{
			resp:   msqResponseBids{Dsa: make(chan int)},
			extBid: openrtb_ext.ExtBid{},
			isOk:   false,
		},
	}

	for index, test := range tests {
		extBid, errs := test.resp.loadExtBid()
		assert.Equal(t, test.extBid.DSA, extBid.DSA, fmt.Sprintf("extBid.DSA >> index: %d.", index))
		assert.Equal(t, test.isOk, errs == nil, fmt.Sprintf("isOk >> index: %d.", index))
	}
}

func TestExtBidPrebidMeta(t *testing.T) {
	tests := []struct {
		// tests inputs
		resp msqResponseBids
		// tests expected-results
		adomains  []string
		mediatype string
		value     openrtb_ext.ExtBidPrebidMeta
	}{
		{
			resp:      msqResponseBids{ADomain: []string{"test-adomain-0", "test-adomain-1"}},
			adomains:  []string{"test-adomain-0", "test-adomain-1"},
			mediatype: "banner",
			value: openrtb_ext.ExtBidPrebidMeta{
				AdvertiserDomains: []string{"test-adomain-0", "test-adomain-1"},
				MediaType:         "banner",
			},
		},
		{
			resp:      msqResponseBids{},
			adomains:  nil,
			mediatype: "banner",
			value:     openrtb_ext.ExtBidPrebidMeta{MediaType: "banner"},
		},
		{
			resp:      msqResponseBids{Video: &msqResponseBidsVideo{Xml: "not-nil"}},
			adomains:  nil,
			mediatype: "video",
			value:     openrtb_ext.ExtBidPrebidMeta{MediaType: "video"},
		},
	}

	for index, test := range tests {
		result := test.resp.extBidPrebidMeta()
		assert.Equal(t, test.adomains, result.AdvertiserDomains, fmt.Sprintf("ADomains >> index: %d.", index))
		assert.Equal(t, test.mediatype, result.MediaType, fmt.Sprintf("MediaType >> index: %d.", index))
		assert.Equal(t, test.value, *result, fmt.Sprintf("ExactValue >> index: %d.", index))
	}
}

func TestExtBid(t *testing.T) {
	tests := []struct {
		// tests inputs
		resp msqResponseBids
		// tests expected-results
		raw json.RawMessage
	}{
		{
			resp: msqResponseBids{Dsa: openrtb_ext.ExtBidDSA{Behalf: "behalf"}},
			raw:  json.RawMessage([]byte(`{"dsa":{"behalf":"behalf"}}`)),
		},
		{
			resp: msqResponseBids{},
			raw:  nil,
		},
	}

	for index, test := range tests {
		assert.Equal(t, test.raw, test.resp.extBid(), fmt.Sprintf("Raw >> index: %d.", index))
	}
}
