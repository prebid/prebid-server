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
		name           string
		bidRequest     openrtb2.BidRequest
		impIdToBidders map[string]map[string]json.RawMessage
		extImpPrebid   openrtb_ext.ExtImpPrebid
		expectErr      bool
		expectData     openrtb_ext.ExtImpPrebid
	}{
		{
			name:           "One imp with 2 bidders, should be changed to one bidder",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderA": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"param1": "value1"}`),
			}},
		},
		{
			name:           "One imp with 2 bidders, overwrite all bidders in imp",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpA": {"bidderC": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
				"bidderB": json.RawMessage(`{"paramB": "valueB"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderC": json.RawMessage(`{"param1": "value1"}`),
			}},
		},
		{
			name:           "No bidders in impIdToBidders",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			impIdToBidders: map[string]map[string]json.RawMessage{},
			extImpPrebid: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
			expectErr: false,
			expectData: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
				"bidderA": json.RawMessage(`{"paramA": "valueA"}`),
			}},
		},
		{
			name:           "One imp with 1 bidder, imp not found",
			bidRequest:     openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "ImpA"}}},
			impIdToBidders: map[string]map[string]json.RawMessage{"ImpABC": {"bidderC": json.RawMessage(`{"param1": "value1"}`)}},
			extImpPrebid: openrtb_ext.ExtImpPrebid{Bidder: map[string]json.RawMessage{
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
			brw := openrtb_ext.RequestWrapper{BidRequest: &tt.bidRequest}
			impExt, err := brw.GetImp()[0].GetImpExt()
			assert.NoError(t, err)

			impExt.SetPrebid(&tt.extImpPrebid)
			payload := BidderRequestPayload{Bidder: "appnexus", Request: &brw}

			cbr := ChangeSetBidderRequest[BidderRequestPayload]{
				changeSet: &ChangeSet[BidderRequestPayload]{},
			}
			cbr.Bidders().Update(tt.impIdToBidders)

			for _, mut := range cbr.changeSet.Mutations() {
				_, err := mut.Apply(payload)
				assert.NoError(t, err)
			}

			impExtRes, err := payload.Request.GetImp()[0].GetImpExt()
			assert.NoError(t, err)
			assert.Equal(t, &tt.expectData, impExtRes.GetPrebid(), "Bidder data should match expected")

		})
	}

}
