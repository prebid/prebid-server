package adservertargeting

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestExtractAdServerTargeting(t *testing.T) {

	r := openrtb2.BidRequest{
		ID: "anyID",
		Imp: []openrtb2.Imp{
			{
				ID: "imp1", BidFloor: 10.00,
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1, "test": {"testUser": "user1"}}, "other": "otherImp", "bidder1": {"tagid": 111, "placementId": "test1"}}`),
			},
			{
				ID: "imp2", BidFloor: 20.00,
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1, "test": {"testUser": "user2"}}, "other": "otherImp", "bidder1": {"tagid": 222, "placementId": "test2"}}`),
			},
			{
				ID: "imp3", BidFloor: 30.00,
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1, "test": {"testUser": "user3"}}, "other": "otherImp", "bidder1": {"tagid": 333, "placementId": "test3"}}`),
			},
		},
		User: &openrtb2.User{
			ID:       "testUser",
			Yob:      2000,
			Keywords: "keywords",
		},
		Ext: json.RawMessage(reqExt),
	}

	rw := &openrtb_ext.RequestWrapper{BidRequest: &r}

	p := "https://www.test-url.com?ampkey=testAmpKey&data-override-height=400"
	u, _ := url.Parse(p)
	params := u.Query()
	reqBytes, err := jsonutil.Marshal(r)
	assert.NoError(t, err, "unexpected req marshal error")

	res, warnings := collect(rw, reqBytes, params)
	assert.Len(t, warnings, 2, "incorrect warnings")
	assert.Equal(t, "incorrect value type for path: imp.ext.prebid.test, value can only be string or number", warnings[0].Message, "incorrect warning")
	assert.Equal(t, "incorrect value type for path: ext.prebid.targeting.includebrandcategory, value can only be string or number", warnings[1].Message, "incorrect warning")

	assert.Len(t, res.RequestTargetingData, 5, "incorrect request targeting data length")
	assert.Len(t, res.ResponseTargetingData, 4, "incorrect response targeting data length")

	assert.Equal(t, res.RequestTargetingData["hb_amp_param"].SingleVal, json.RawMessage(`testAmpKey`), "incorrect requestTargetingData value for key: hb_amp_param")

	assert.Len(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId, 3, "incorrect requestTargetingData length for key: hb_req_imp_ext_bidder_param")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId["imp1"], []byte(`111`), "incorrect requestTargetingData value for key: hb_req_imp_ext_bidder_param.imp1")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId["imp2"], []byte(`222`), "incorrect requestTargetingData value for key: hb_req_imp_ext_bidder_param.imp2")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId["imp3"], []byte(`333`), "incorrect requestTargetingData value for key: hb_req_imp_ext_bidder_param.imp3")

	assert.Len(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId, 3, "incorrect requestTargetingData length for key: hb_req_imp_param")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId["imp1"], []byte(`10`), "incorrect requestTargetingData value for key: hb_req_imp_param.imp1")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId["imp2"], []byte(`20`), "incorrect requestTargetingData value for key: hb_req_imp_param.imp2")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId["imp3"], []byte(`30`), "incorrect requestTargetingData value for key: hb_req_imp_param.imp3")

	assert.Equal(t, res.RequestTargetingData["hb_req_user_param"].SingleVal, json.RawMessage(`2000`), "incorrect requestTargetingData value for key: hb_req_user_param")
	assert.Equal(t, res.RequestTargetingData["hb_static_thing"].SingleVal, json.RawMessage(`test-static-value`), "incorrect requestTargetingData value for key: hb_static_thing")

	assert.Equal(t, res.ResponseTargetingData[0].Key, "{{BIDDER}}_custom1", "incorrect ResponseTargetingData.Key")
	assert.True(t, res.ResponseTargetingData[0].HasMacro, "incorrect ResponseTargetingData.HasMacro")
	assert.Equal(t, res.ResponseTargetingData[0].Path, "seatbid.bid.ext.custom1", "incorrect ResponseTargetingData.Path")

	assert.Equal(t, res.ResponseTargetingData[1].Key, "custom2", "incorrect ResponseTargetingData.Key")
	assert.False(t, res.ResponseTargetingData[1].HasMacro, "incorrect ResponseTargetingData.HasMacro")
	assert.Equal(t, res.ResponseTargetingData[1].Path, "seatbid.bid.ext.custom2", "incorrect ResponseTargetingData.Path")

}

func TestResolveAdServerTargeting(t *testing.T) {
	nbr := openrtb3.NoBidReason(2)
	resp := &openrtb2.BidResponse{
		ID:         "testResponse",
		Cur:        "USD",
		BidID:      "testBidId",
		CustomData: "testCustomData",
		NBR:        &nbr,
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "appnexus",
				Bid: []openrtb2.Bid{
					{ID: "bidA1", ImpID: "imp1", Price: 10, Cat: []string{"cat11", "cat12"}, Ext: []byte(`{"prebid": {"foo": "bar1"}}`)},
					{ID: "bidA2", ImpID: "imp2", Price: 20, Cat: []string{"cat21", "cat22"}, Ext: []byte(`{"prebid": {"foo": "bar2"}}`)},
					{ID: "bidA3", ImpID: "imp3", Price: 30, Cat: []string{"cat31", "cat32"}, Ext: []byte(`{"prebid": {"foo": "bar3"}}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barApn"}}`),
			},
			{
				Seat: "rubicon",
				Bid: []openrtb2.Bid{
					{ID: "bidR1", ImpID: "imp1", Price: 11, Cat: []string{"cat111", "cat112"}, Ext: []byte(`{"prebid": {"foo": "bar11", "targeting":{"hb_amp_param":"testInputKey1", "testInput": 111}}}`)},
					{ID: "bidR2", ImpID: "imp2", Price: 22, Cat: []string{"cat221", "cat222"}, Ext: []byte(`{"prebid": {"foo": "bar22", "targeting":{"hb_amp_param":"testInputKey2", "testInput": 222}}}`)},
					{ID: "bidR3", ImpID: "imp3", Price: 33, Cat: []string{"cat331", "cat332"}, Ext: []byte(`{"prebid": {"foo": "bar33", "targeting":{"hb_amp_param":"testInputKey3", "testInput": 333}}}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barRubicon"}}`),
			},
		},
		Ext: []byte(`{"prebid": {"seatExt": "true"}}`),
	}

	adServerTargeting := &adServerTargetingData{
		RequestTargetingData: map[string]RequestTargetingData{
			"hb_amp_param": {SingleVal: json.RawMessage(`testAmpKey`), TargetingValueByImpId: nil},
			"hb_imp_param": {SingleVal: nil, TargetingValueByImpId: map[string][]byte{
				"imp1": []byte(`111`),
				"imp2": []byte(`222`),
				"imp3": []byte(`333`),
			},
			},
		},
		ResponseTargetingData: []ResponseTargetingData{
			{Key: "{{BIDDER}}_custom1", HasMacro: true, Path: "seatbid.bid.cat"},
			{Key: "custom2", HasMacro: false, Path: "seatbid.bid.ext.prebid.foo"},
			{Key: "custom3", HasMacro: false, Path: "seatbid.bid.price"},
			{Key: "custom4", HasMacro: false, Path: "seatbid.ext.testData.foo"},
			{Key: "custom5", HasMacro: false, Path: "cur"},
			{Key: "custom6", HasMacro: false, Path: "id"},
			{Key: "custom7", HasMacro: false, Path: "bidid"},
			{Key: "custom8", HasMacro: false, Path: "customdata"},
			{Key: "custom9", HasMacro: false, Path: "nbr"},
			{Key: "{{BIDDER}}_custom6", HasMacro: true, Path: "ext.prebid.seatExt"},
		},
	}
	bidResponse, warnings := resolve(adServerTargeting, resp, nil, nil)

	assert.Len(t, warnings, 6, "incorrect warnings number")
	assert.NotNil(t, bidResponse, "incorrect resolved targeting data")
	assert.Len(t, bidResponse.SeatBid, 2, "incorrect seat bids number")

	assert.Len(t, bidResponse.SeatBid[0].Bid, 3, "incorrect bids number for bidder : %", "appnexus")
	assert.Len(t, bidResponse.SeatBid[1].Bid, 3, "incorrect bids number for bidder : %", "rubicon")

	assert.JSONEq(t, seatBid0Bid0Ext, string(bidResponse.SeatBid[0].Bid[0].Ext), "incorrect ext for SeatBid[0].Bid[0]")
	assert.JSONEq(t, seatBid0Bid1Ext, string(bidResponse.SeatBid[0].Bid[1].Ext), "incorrect ext for SeatBid[0].Bid[1]")
	assert.JSONEq(t, seatBid0Bid2Ext, string(bidResponse.SeatBid[0].Bid[2].Ext), "incorrect ext for SeatBid[0].Bid[2]")
	assert.JSONEq(t, seatBid1Bid0Ext, string(bidResponse.SeatBid[1].Bid[0].Ext), "incorrect ext for SeatBid[1].Bid[0]")
	assert.JSONEq(t, seatBid1Bid1Ext, string(bidResponse.SeatBid[1].Bid[1].Ext), "incorrect ext for SeatBid[1].Bid[1]")
	assert.JSONEq(t, seatBid1Bid2Ext, string(bidResponse.SeatBid[1].Bid[2].Ext), "incorrect ext for SeatBid[1].Bid[2]")

}

func TestResolveAdServerTargetingForMultiBidAndOneImp(t *testing.T) {
	resp := &openrtb2.BidResponse{
		ID:  "testResponse",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "appnexus",
				Bid: []openrtb2.Bid{
					{ID: "bidA1", ImpID: "imp1", Price: 10, Cat: []string{"cat11", "cat12"}, Ext: []byte(`{"prebid": {"foo": "bar1"}}`)},
					{ID: "bidA2", ImpID: "imp1", Price: 20, Cat: []string{"cat21", "cat22"}, Ext: []byte(`{"prebid": {"foo": "bar2"}}`)},
					{ID: "bidA3", ImpID: "imp1", Price: 30, Cat: []string{"cat31", "cat32"}, Ext: []byte(`{"prebid": {"foo": "bar3"}}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barApn"}}`),
			},
		},
		Ext: []byte(`{"prebid": {"seatExt": "true"}}`),
	}

	adServerTargeting := &adServerTargetingData{
		RequestTargetingData: nil,
		ResponseTargetingData: []ResponseTargetingData{
			{Key: "custom_attribute", HasMacro: false, Path: "seatbid.bid.ext.prebid.foo"},
		},
	}
	truncateTargetingAttr := 11
	bidResponse, warnings := resolve(adServerTargeting, resp, nil, &truncateTargetingAttr)

	assert.Empty(t, warnings, "unexpected error")
	assert.Equal(t, bidResponse.ID, "testResponse", "incorrect")
	assert.Len(t, bidResponse.SeatBid, 1, "incorrect seat bids number")
	assert.Len(t, bidResponse.SeatBid[0].Bid, 3, "incorrect bids number")

	assert.JSONEq(t, bid0Ext, string(bidResponse.SeatBid[0].Bid[0].Ext), "incorrect ext for SeatBid[0].Bid[0]")
	assert.JSONEq(t, bid1Ext, string(bidResponse.SeatBid[0].Bid[1].Ext), "incorrect ext for SeatBid[0].Bid[1]")
	assert.JSONEq(t, bid2Ext, string(bidResponse.SeatBid[0].Bid[2].Ext), "incorrect ext for SeatBid[0].Bid[2]")

}

func TestProcessAdServerTargetingFull(t *testing.T) {

	r := openrtb2.BidRequest{
		ID: "anyID",
		Imp: []openrtb2.Imp{
			{
				ID: "imp1", BidFloor: 10.00,
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1, "test": {"testUser": "user1"}}, "other": "otherImp", "bidder1": {"tagid": 111, "placementId": "test1"}}`),
			},
			{
				ID: "imp2", BidFloor: 20.00,
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1, "test": {"testUser": "user2"}}, "other": "otherImp", "bidder1": {"tagid": 222, "placementId": "test2"}}`),
			},
			{
				ID: "imp3", BidFloor: 30.00,
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1, "test": {"testUser": "user3"}}, "other": "otherImp", "bidder1": {"tagid": 333, "placementId": "test3"}}`),
			},
		},
		User: &openrtb2.User{
			ID:       "testUser",
			Yob:      2000,
			Keywords: "keywords",
		},
		Ext: json.RawMessage(reqExt),
	}

	rw := &openrtb_ext.RequestWrapper{BidRequest: &r}

	p := "https://www.test-url.com?ampkey=testAmpKey&data-override-height=400"
	u, _ := url.Parse(p)
	params := u.Query()

	resp := &openrtb2.BidResponse{
		ID:  "testResponse",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "appnexus",
				Bid: []openrtb2.Bid{
					{ID: "bidA1", ImpID: "imp1", Price: 10, Cat: []string{"cat11", "cat12"}, Ext: []byte(`{"prebid": {"foo": "bar1"}, "custom1": 1111, "custom2": "a1111"}`)},
					{ID: "bidA2", ImpID: "imp2", Price: 20, Cat: []string{"cat21", "cat22"}, Ext: []byte(`{"prebid": {"foo": "bar2"}, "custom1": 2222, "custom2": "a2222"}`)},
					{ID: "bidA3", ImpID: "imp3", Price: 30, Cat: []string{"cat31", "cat32"}, Ext: []byte(`{"prebid": {"foo": "bar3"}, "custom1": 3333, "custom2": "a3333"}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barApn"}}`),
			},
			{
				Seat: "rubicon",
				Bid: []openrtb2.Bid{
					{ID: "bidR1", ImpID: "imp1", Price: 11, Cat: []string{"cat111", "cat112"}, Ext: []byte(`{"custom1": 4444, "custom2": "r4444", "prebid": {"foo": "bar11", "targeting":{"hb_amp_param":"testInputKey1", "testInput": 111}}}`)},
					{ID: "bidR2", ImpID: "imp2", Price: 22, Cat: []string{"cat221", "cat222"}, Ext: []byte(`{"custom1": 5555, "custom2": "r5555", "prebid": {"foo": "bar22", "targeting":{"hb_amp_param":"testInputKey2", "testInput": 222}}}`)},
					{ID: "bidR3", ImpID: "imp3", Price: 33, Cat: []string{"cat331", "cat332"}, Ext: []byte(`{"custom1": 6666, "custom2": "r6666", "prebid": {"foo": "bar33", "targeting":{"hb_amp_param":"testInputKey3", "testInput": 333}}}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barRubicon"}}`),
			},
		},
		Ext: []byte(`{"prebid": {"seatExt": "true"}}`),
	}

	bidResponseExt := &openrtb_ext.ExtBidResponse{Warnings: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)}

	reqBytes, err := jsonutil.Marshal(r)
	assert.NoError(t, err, "unexpected req marshal error")
	targetingKeyLen := 0
	resResp := Apply(rw, reqBytes, resp, params, bidResponseExt, &targetingKeyLen)
	assert.Len(t, resResp.SeatBid, 2, "incorrect response: seat bid number")
	assert.Len(t, bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral], 2, "incorrect warnings number")
	assert.Equal(t, "incorrect value type for path: imp.ext.prebid.test, value can only be string or number", bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral][0].Message, "incorrect warning")
	assert.Equal(t, "incorrect value type for path: ext.prebid.targeting.includebrandcategory, value can only be string or number", bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral][1].Message, "incorrect warning")

	apnBids := resResp.SeatBid[0].Bid
	rbcBids := resResp.SeatBid[1].Bid

	assert.Len(t, apnBids, 3, "Incorrect response: appnexus bids number")
	assert.Len(t, rbcBids, 3, "Incorrect response: rubicon bid number")

	assert.JSONEq(t, apnBid0Ext, string(apnBids[0].Ext), "incorrect ext for appnexus bid[0]")
	assert.JSONEq(t, apnBid1Ext, string(apnBids[1].Ext), "incorrect ext for appnexus bid[1]")
	assert.JSONEq(t, apnBid2Ext, string(apnBids[2].Ext), "incorrect ext for appnexus bid[2]")

	assert.JSONEq(t, rbcBid0Ext, string(rbcBids[0].Ext), "incorrect ext for rubicon bid[0]")
	assert.JSONEq(t, rbcBid1Ext, string(rbcBids[1].Ext), "incorrect ext for rubicon bid[1]")
	assert.JSONEq(t, rbcBid2Ext, string(rbcBids[2].Ext), "incorrect ext for rubicon bid[2]")

}

func TestProcessAdServerTargetingWarnings(t *testing.T) {

	r := openrtb2.BidRequest{
		ID: "anyID",
		Imp: []openrtb2.Imp{
			{
				ID:  "imp1",
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1 }, "other": "otherImp", "bidder1": {"placementId": "test1"}}`),
			},
			{
				ID:  "imp2",
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1}, "other": "otherImp", "bidder1": {"placementId": "test2"}}`),
			},
			{
				ID:  "imp3",
				Ext: json.RawMessage(`{"prebid": {"is_rewarded_inventory": 1}, "other": "otherImp", "bidder1": {"placementId": "test3"}}`),
			},
		},
		User: &openrtb2.User{
			ID:       "testUser",
			Keywords: "keywords",
		},
		Ext: json.RawMessage(reqExt),
	}

	rw := &openrtb_ext.RequestWrapper{BidRequest: &r}

	p := "https://www.test-url.com?data-override-height=400"
	u, _ := url.Parse(p)
	params := u.Query()

	resp := &openrtb2.BidResponse{
		ID: "testResponse",
		SeatBid: []openrtb2.SeatBid{
			{
				Seat: "appnexus",
				Bid: []openrtb2.Bid{
					{ID: "bidA1", ImpID: "imp1", Price: 10, Cat: []string{"cat11", "cat12"}, Ext: []byte(`{"prebid": {"foo": "bar1"}}`)},
					{ID: "bidA2", ImpID: "imp2", Price: 20, Cat: []string{"cat21", "cat22"}, Ext: []byte(`{"prebid": {"foo": "bar2"}}`)},
					{ID: "bidA3", ImpID: "imp3", Price: 30, Cat: []string{"cat31", "cat32"}, Ext: []byte(`{"prebid": {"foo": "bar3"}}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barApn"}}`),
			},
			{
				Seat: "rubicon",
				Bid: []openrtb2.Bid{
					{ID: "bidR1", ImpID: "imp1", Price: 11, Cat: []string{"cat111", "cat112"}, Ext: []byte(`{"prebid": {"foo": "bar11", "targeting":{"hb_amp_param":"testInputKey1", "testInput": 111}}}`)},
					{ID: "bidR2", ImpID: "imp2", Price: 22, Cat: []string{"cat221", "cat222"}, Ext: []byte(`{"prebid": {"foo": "bar22", "targeting":{"hb_amp_param":"testInputKey2", "testInput": 222}}}`)},
					{ID: "bidR3", ImpID: "imp3", Price: 33, Cat: []string{"cat331", "cat332"}, Ext: []byte(`{"prebid": {"foo": "bar33", "targeting":{"hb_amp_param":"testInputKey3", "testInput": 333}}}`)},
				},
				Ext: []byte(`{"testData": {"foo": "barRubicon"}}`),
			},
		},
		Ext: []byte(`{"prebid": {"seatExt": "true"}}`),
	}

	bidResponseExt := &openrtb_ext.ExtBidResponse{Warnings: make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage)}

	reqBytes, err := jsonutil.Marshal(r)
	assert.NoError(t, err, "unexpected req marshal error")
	resResp := Apply(rw, reqBytes, resp, params, bidResponseExt, nil)
	assert.Len(t, resResp.SeatBid, 2, "Incorrect response: seat bid number")

	apnBids := resResp.SeatBid[0].Bid
	rbcBids := resResp.SeatBid[1].Bid

	assert.Len(t, apnBids, 3, "Incorrect response: appnexus bids number")
	assert.Len(t, rbcBids, 3, "Incorrect response: rubicon bid number")

	warnings := bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral]
	assert.Len(t, warnings, 18, "Incorrect response: seat bid number")
	assert.Equal(t, "value not found for path: ext.prebid.amp.data.ampkey", warnings[0].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: imp.ext.prebid.test", warnings[1].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: imp.ext.bidder1.tagid", warnings[2].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: imp.bidfloor", warnings[3].Message, "Incorrect warning")
	assert.Equal(t, "incorrect value type for path: ext.prebid.targeting.includebrandcategory, value can only be string or number", warnings[4].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: user.yob", warnings[5].Message, "Incorrect warning")

	assert.Equal(t, "value not found for path: ext.custom1 for bidder: appnexus, bid id: bidA1", warnings[6].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom2 for bidder: appnexus, bid id: bidA1", warnings[7].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom1 for bidder: appnexus, bid id: bidA2", warnings[8].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom2 for bidder: appnexus, bid id: bidA2", warnings[9].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom1 for bidder: appnexus, bid id: bidA3", warnings[10].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom2 for bidder: appnexus, bid id: bidA3", warnings[11].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom1 for bidder: rubicon, bid id: bidR1", warnings[12].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom2 for bidder: rubicon, bid id: bidR1", warnings[13].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom1 for bidder: rubicon, bid id: bidR2", warnings[14].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom2 for bidder: rubicon, bid id: bidR2", warnings[15].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom1 for bidder: rubicon, bid id: bidR3", warnings[16].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: ext.custom2 for bidder: rubicon, bid id: bidR3", warnings[17].Message, "Incorrect warning")
}
