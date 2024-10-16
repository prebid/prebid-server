package mediasquare

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestGetContent(t *testing.T) {
	tests := []struct {
		resp     MsqResponse
		value    adapters.BidderResponse
		expected adapters.BidderResponse
	}{
		{
			resp:     MsqResponse{},
			value:    adapters.BidderResponse{Currency: ""},
			expected: adapters.BidderResponse{Currency: ""},
		},
		{
			resp: MsqResponse{
				Responses: []MsqResponseBids{
					{
						ID:       "id-ok",
						Ad:       "ad-ok",
						Cpm:      42.0,
						Currency: "currency-ok",
						Width:    42,
						Height:   42,
						Bidder:   "bidder-ok",
					},
				},
			},
			value: adapters.BidderResponse{Currency: ""},
			expected: adapters.BidderResponse{
				Currency: "currency-ok",
				Bids: []*adapters.TypedBid{
					{
						Bid: &openrtb2.Bid{
							AdM:   "ad-ok",
							ID:    "id-ok",
							Price: 42.0,
							W:     42,
							H:     42,
							MType: openrtb2.MarkupBanner,
						},
						BidMeta: &openrtb_ext.ExtBidPrebidMeta{MediaType: "banner"},
						BidType: "banner",
					},
				},
			},
		},
	}

	for index, test := range tests {
		test.resp.getContent(&test.value)
		expectedBytes, _ := json.Marshal(test.expected)
		valueBytes, _ := json.Marshal(test.value)
		assert.Equal(t, expectedBytes, valueBytes,
			fmt.Sprintf("getContent >> index: %d\nexpect:%s\nvalue:%s", index, string(expectedBytes), string(valueBytes)))
	}
}

func TestSetContent(t *testing.T) {
	tests := []struct {
		// tests inputs
		params MsqParametersCodes
		imp    openrtb2.Imp
		// tests expected-results
		ok bool
	}{
		{
			params: MsqParametersCodes{},
			imp:    openrtb2.Imp{},
			ok:     false,
		},
		{
			params: MsqParametersCodes{
				AdUnit:    "adunit-ok",
				AuctionId: "auctionid-ok",
				Code:      "code-ok",
				BidId:     "bidid-ok",
			},
			imp: openrtb2.Imp{
				ID: "imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 1, H: 2}, {W: 2, H: 1}},
					Ext:    json.RawMessage(`{"type":"raw-message-id"}`),
				},
				BidFloor: 0.8,
			},
			ok: true,
		},
		{
			params: MsqParametersCodes{},
			imp: openrtb2.Imp{
				ID: "imp-id",
				Video: &openrtb2.Video{
					MIMEs: []string{"MIMEs-ok"},
					W:     intAsPtrInt64(42), H: intAsPtrInt64(42),
					Ext: json.RawMessage(`{"h":42,"w":42}`),
				},
				BidFloor: 0.8,
			},
			ok: true,
		},
		{
			params: MsqParametersCodes{},
			imp: openrtb2.Imp{
				ID: "imp-id",
				Native: &openrtb2.Native{
					Ext: json.RawMessage(`{"sizes":[[42,42],[2,1],[1,1]]}`),
				},
				BidFloor: 0.8,
			},

			ok: true,
		},
		{
			params: MsqParametersCodes{},
			imp: openrtb2.Imp{
				ID: "imp-id",
				Banner: &openrtb2.Banner{
					W: intAsPtrInt64(42),
					H: intAsPtrInt64(42),
				},
				BidFloor: 0.8,
			},
			ok: true,
		},
	}

	for index, test := range tests {
		expected := test.params

		ok := test.params.setContent(test.imp)
		assert.Equal(t, test.ok, ok, fmt.Sprintf("ok >> index: %d", index))

		switch index {
		case 1:
			expected.Mediatypes.Banner = &MediaTypeBanner{Sizes: [][]*int{{intAsPtrInt(1), intAsPtrInt(2)}, {intAsPtrInt(2), intAsPtrInt(1)}}}
			expected.Floor = map[string]MsqFloor{"1x2": {Price: 0.8}, "2x1": {Price: 0.8}}
		case 2:
			expected.Floor = map[string]MsqFloor{"42x42": {Price: 0.8}, "*": {Price: 0.8}}
			expected.Mediatypes.Video = &MediaTypeVideo{Mimes: []string{"MIMEs-ok"}, H: intAsPtrInt(42), W: intAsPtrInt(42)}
		case 3:
			expected.Mediatypes.Native = &MediaTypeNative{Type: "native", Sizes: [][]int{{42, 42}, {2, 1}, {1, 1}}}
			expected.Floor = map[string]MsqFloor{"1x1": {Price: 0.8}, "42x42": {Price: 0.8}, "2x1": {Price: 0.8}, "*": {Price: 0.8}}
		case 4:
			expected.Mediatypes.Banner = &MediaTypeBanner{Sizes: [][]*int{{intAsPtrInt(42), intAsPtrInt(42)}}}
			expected.Floor = map[string]MsqFloor{"42x42": {Price: 0.8}}
		}

		expectedBytes, _ := json.Marshal(expected)
		paramsBytes, _ := json.Marshal(test.params)
		assert.Equal(t, string(expectedBytes), string(paramsBytes), fmt.Sprintf("assert >> index: %d", index))
	}
}

func intAsPtrInt(i int) *int {
	val := i
	return &val
}

func intAsPtrInt64(i int64) *int64 {
	val := i
	return &val
}
