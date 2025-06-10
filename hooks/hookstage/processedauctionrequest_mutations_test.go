package hookstage

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdatePrebidBidders(t *testing.T) {
	tests := []struct {
		name            string
		bidRequest      *openrtb2.BidRequest
		impIdToBidders  map[string]map[string]json.RawMessage
		extImpPrebid    *openrtb_ext.ExtImpPrebid
		expectErr       bool
		expectEmptyImps bool
		expectData      openrtb_ext.ExtImpPrebid
	}{

		{
			name:           "nil-req-imp",
			bidRequest:     &openrtb2.BidRequest{Imp: nil},
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderA": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectErr:       false,
			expectEmptyImps: true,
		},
		{
			name:           "empty-req-imp",
			bidRequest:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{}},
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderA": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectErr:       false,
			expectEmptyImps: true,
		},
		{
			name:           "nil-req-imp-ext-prebid",
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderA": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid:   nil,
			expectErr:      false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"param1": "value1"}`),
			},
			},
		},
		{
			name:           "one-req-imp-with-multiple-bidders-update-existing-bidder",
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderA": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"param1": "value1"}`),
			}},
		},
		{
			name:           "one-req-imp-with-multiple-bidders-update-new-bidder",
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderC": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderC": json.RawMessage(`{"param1": "value1"}`),
			}},
		},
		{
			name:           "empty-imp-map",
			impIdToBidders: map[string]map[string]json.RawMessage{},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
		},
		{
			name:           "one-req-imp-with-one-bidder-imp-not-found",
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpABC": {"bidderC": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: &openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}}
			if tt.bidRequest != nil {
				br = tt.bidRequest
			}
			brw := openrtb_ext.RequestWrapper{BidRequest: br}
			impWrapperArr := brw.GetImp()

			if len(impWrapperArr) > 0 {
				impExt, err := brw.GetImp()[0].GetImpExt()
				assert.NoError(t, err)
				impExt.SetPrebid(tt.extImpPrebid)
			}

			payload := ProcessedAuctionRequestPayload{Request: &brw}

			cpar := ChangeSetProcessedAuctionRequest[ProcessedAuctionRequestPayload]{
				changeSet: &ChangeSet[ProcessedAuctionRequestPayload]{},
			}
			cpar.Bidders().Update(tt.impIdToBidders)

			for _, mut := range cpar.changeSet.Mutations() {
				_, err := mut.Apply(payload)
				assert.NoError(t, err)
			}

			if tt.expectEmptyImps {
				assert.Empty(t, payload.Request.GetImp(), "Expected no imps in the request")
				return
			}

			impExtRes, err := payload.Request.GetImp()[0].GetImpExt()
			assert.NoError(t, err)
			assert.Equal(t, &tt.expectData, impExtRes.GetPrebid(), "Bidder data should match expected")

		})
	}

}
