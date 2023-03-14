package adservertargeting

import (
	"encoding/json"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
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
	reqBytes, err := json.Marshal(r)
	assert.NoError(t, err, "unexpected req marshal error")

	res, warnings := collect(rw, reqBytes, params)
	assert.Empty(t, warnings, "unexpected warnings")

	assert.Len(t, res.RequestTargetingData, 7, "incorrect request targeting data length")
	assert.Len(t, res.ResponseTargetingData, 4, "incorrect response targeting data length")

	assert.Equal(t, res.RequestTargetingData["hb_amp_param"].SingleVal, json.RawMessage(`testAmpKey`), "incorrect requestTargetingData value for key: hb_amp_param")

	assert.Len(t, res.RequestTargetingData["hb_req_imp_ext_param"].TargetingValueByImpId, 3, "incorrect requestTargetingData length for key hb_req_imp_ext_param")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_param"].TargetingValueByImpId["imp1"], []byte(`{"testUser":"user1"}`), "incorrect requestTargetingData value for key: hb_req_imp_ext_param.imp1")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_param"].TargetingValueByImpId["imp2"], []byte(`{"testUser":"user2"}`), "incorrect requestTargetingData value for key: hb_req_imp_ext_param.imp2")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_param"].TargetingValueByImpId["imp3"], []byte(`{"testUser":"user3"}`), "incorrect requestTargetingData value for key: hb_req_imp_ext_param.imp3")

	assert.Len(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId, 3, "incorrect requestTargetingData length for key: hb_req_imp_ext_bidder_param")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId["imp1"], []byte(`111`), "incorrect requestTargetingData value for key: hb_req_imp_ext_bidder_param.imp1")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId["imp2"], []byte(`222`), "incorrect requestTargetingData value for key: hb_req_imp_ext_bidder_param.imp2")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_ext_bidder_param"].TargetingValueByImpId["imp3"], []byte(`333`), "incorrect requestTargetingData value for key: hb_req_imp_ext_bidder_param.imp3")

	assert.Len(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId, 3, "incorrect requestTargetingData length for key: hb_req_imp_param")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId["imp1"], []byte(`10`), "incorrect requestTargetingData value for key: hb_req_imp_param.imp1")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId["imp2"], []byte(`20`), "incorrect requestTargetingData value for key: hb_req_imp_param.imp2")
	assert.Equal(t, res.RequestTargetingData["hb_req_imp_param"].TargetingValueByImpId["imp3"], []byte(`30`), "incorrect requestTargetingData value for key: hb_req_imp_param.imp3")

	assert.Equal(t, res.RequestTargetingData["hb_req_ext_param"].SingleVal, json.RawMessage(`{"primaryadserver":1,"publisher":"","withcategory":true}`), "incorrect requestTargetingData value for key: hb_req_ext_param")
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
	resp := &openrtb2.BidResponse{
		ID:  "testResponse",
		Cur: "USD",
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
			{Key: "{{BIDDER}}_custom6", HasMacro: true, Path: "ext.prebid.seatExt"},
		},
	}
	bidResponse, warnings := resolve(adServerTargeting, resp, nil, nil)

	assert.Empty(t, warnings, "unexpected warnings")
	assert.NotNil(t, bidResponse, "incorrect resolved targeting data")
	assert.Len(t, bidResponse.SeatBid, 2, "incorrect seat bids number")

	assert.Len(t, bidResponse.SeatBid[0].Bid, 3, "incorrect bids number for bidder : %", "appnexus")
	assert.Len(t, bidResponse.SeatBid[1].Bid, 3, "incorrect bids number for bidder : %", "rubicon")

	assert.JSONEq(t, SeatBid0Bid0Ext, string(bidResponse.SeatBid[0].Bid[0].Ext), "incorrect ext for SeatBid[0].Bid[0]")
	assert.JSONEq(t, SeatBid0Bid1Ext, string(bidResponse.SeatBid[0].Bid[1].Ext), "incorrect ext for SeatBid[0].Bid[1]")
	assert.JSONEq(t, SeatBid0Bid2Ext, string(bidResponse.SeatBid[0].Bid[2].Ext), "incorrect ext for SeatBid[0].Bid[2]")
	assert.JSONEq(t, SeatBid1Bid0Ext, string(bidResponse.SeatBid[1].Bid[0].Ext), "incorrect ext for SeatBid[1].Bid[0]")
	assert.JSONEq(t, SeatBid1Bid1Ext, string(bidResponse.SeatBid[1].Bid[1].Ext), "incorrect ext for SeatBid[1].Bid[1]")
	assert.JSONEq(t, SeatBid1Bid2Ext, string(bidResponse.SeatBid[1].Bid[2].Ext), "incorrect ext for SeatBid[1].Bid[2]")

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

	assert.JSONEq(t, Bid0Ext, string(bidResponse.SeatBid[0].Bid[0].Ext), "incorrect ext for SeatBid[0].Bid[0]")
	assert.JSONEq(t, Bid1Ext, string(bidResponse.SeatBid[0].Bid[1].Ext), "incorrect ext for SeatBid[0].Bid[1]")
	assert.JSONEq(t, Bid2Ext, string(bidResponse.SeatBid[0].Bid[2].Ext), "incorrect ext for SeatBid[0].Bid[2]")

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

	bidResponseExt := &openrtb_ext.ExtBidResponse{}

	reqBytes, err := json.Marshal(r)
	assert.NoError(t, err, "unexpected req marshal error")
	resResp := Apply(rw, reqBytes, resp, params, bidResponseExt, nil)
	assert.Len(t, resResp.SeatBid, 2, "Incorrect response: seat bid number")
	assert.Nil(t, bidResponseExt.Warnings, "Incorrect response: no warnings expected")

	apnBids := resResp.SeatBid[0].Bid
	rbcBids := resResp.SeatBid[1].Bid

	assert.Len(t, apnBids, 3, "Incorrect response: appnexus bids number")
	assert.Len(t, rbcBids, 3, "Incorrect response: rubicon bid number")

	assert.JSONEq(t, ApnBid0Ext, string(apnBids[0].Ext), "incorrect ext for appnexus bid[0]")
	assert.JSONEq(t, ApnBid1Ext, string(apnBids[1].Ext), "incorrect ext for appnexus bid[1]")
	assert.JSONEq(t, ApnBid2Ext, string(apnBids[2].Ext), "incorrect ext for appnexus bid[2]")

	assert.JSONEq(t, RbcBid0Ext, string(rbcBids[0].Ext), "incorrect ext for rubicon bid[0]")
	assert.JSONEq(t, RbcBid1Ext, string(rbcBids[1].Ext), "incorrect ext for rubicon bid[1]")
	assert.JSONEq(t, RbcBid2Ext, string(rbcBids[2].Ext), "incorrect ext for rubicon bid[2]")

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

	reqBytes, err := json.Marshal(r)
	assert.NoError(t, err, "unexpected req marshal error")
	resResp := Apply(rw, reqBytes, resp, params, bidResponseExt, nil)
	assert.Len(t, resResp.SeatBid, 2, "Incorrect response: seat bid number")

	apnBids := resResp.SeatBid[0].Bid
	rbcBids := resResp.SeatBid[1].Bid

	assert.Len(t, apnBids, 3, "Incorrect response: appnexus bids number")
	assert.Len(t, rbcBids, 3, "Incorrect response: rubicon bid number")

	warnings := bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral]
	assert.Len(t, warnings, 5, "Incorrect response: seat bid number")
	assert.Equal(t, "value not found for path: ext.prebid.amp.data.ampkey", warnings[0].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: imp.ext.prebid.test", warnings[1].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: imp.ext.bidder1.tagid", warnings[2].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: imp.bidfloor", warnings[3].Message, "Incorrect warning")
	assert.Equal(t, "value not found for path: user.yob", warnings[4].Message, "Incorrect warning")
}

const (
	reqExt = `{
  "prebid": {
    "adservertargeting": [
      {
        "key": "hb_amp_param",
        "source": "bidrequest",
        "value": "ext.prebid.amp.data.ampkey"
      },
      {
        "key": "hb_req_imp_ext_param",
        "source": "bidrequest",
        "value": "imp.ext.prebid.test"
      },
      {
        "key": "hb_req_imp_ext_bidder_param",
        "source": "bidrequest",
        "value": "imp.ext.bidder1.tagid"
      },
      {
        "key": "hb_req_imp_param",
        "source": "bidrequest",
        "value": "imp.bidfloor"
      },
      {
        "key": "hb_req_ext_param",
        "source": "bidrequest",
        "value": "ext.prebid.targeting.includebrandcategory"
      },
	  {
        "key": "hb_req_user_param",
        "source": "bidrequest",
        "value": "user.yob"
      },
      {
        "key": "hb_static_thing",
        "source": "static",
        "value": "test-static-value"
      },
      {
        "key": "{{BIDDER}}_custom1",
        "source": "bidresponse",
        "value": "seatbid.bid.ext.custom1"
      },
      {
        "key": "custom2",
        "source": "bidresponse",
        "value": "seatbid.bid.ext.custom2"
      },
      {
        "key": "{{BIDDER}}_imp",
        "source": "bidresponse",
        "value": "seatbid.bid.impid"
      },
      {
        "key": "seat_cur",
        "source": "bidresponse",
        "value": "cur"
      }
    ],
    "targeting": {
      "includewinners": false,
      "includebidderkeys": true,
      "includebrandcategory": {
        "primaryadserver": 1,
        "publisher": "",
        "withcategory": true
      }
    }
  }
}`

	SeatBid0Bid0Ext = `{
  "prebid": {
    "foo": "bar1",
    "targeting": {
      "appnexus_custom1": "[\"cat11\",\"cat12\"]",
      "appnexus_custom6": "true",
      "custom2": "bar1",
      "custom3": "10",
      "custom4": "barApn",
      "custom5": "USD",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "111"
    }
  }
}`

	SeatBid0Bid1Ext = `
{
  "prebid": {
    "foo": "bar2",
    "targeting": {
      "appnexus_custom1": "[\"cat21\",\"cat22\"]",
      "appnexus_custom6": "true",
      "custom2": "bar2",
      "custom3": "20",
      "custom4": "barApn",
      "custom5": "USD",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "222"
    }
  }
}`
	SeatBid0Bid2Ext = `
{
  "prebid": {
    "foo": "bar3",
    "targeting": {
      "appnexus_custom1": "[\"cat31\",\"cat32\"]",
      "appnexus_custom6": "true",
      "custom2": "bar3",
      "custom3": "30",
      "custom4": "barApn",
      "custom5": "USD",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "333"
    }
  }
}`

	SeatBid1Bid0Ext = `{
  "prebid": {
    "foo": "bar11",
    "targeting": {
      "custom2": "bar11",
      "custom3": "11",
      "custom4": "barRubicon",
      "custom5": "USD",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "111",
      "rubicon_custom1": "[\"cat111\",\"cat112\"]",
      "rubicon_custom6": "true",
      "testInput": 111
    }
  }
}`
	SeatBid1Bid1Ext = `
{
  "prebid": {
    "foo": "bar22",
    "targeting": {
      "custom2": "bar22",
      "custom3": "22",
      "custom4": "barRubicon",
      "custom5": "USD",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "222",
      "rubicon_custom1": "[\"cat221\",\"cat222\"]",
      "rubicon_custom6": "true",
      "testInput": 222
    }
  }
}`

	SeatBid1Bid2Ext = `{
  "prebid": {
    "foo": "bar33",
    "targeting": {
      "custom2": "bar33",
      "custom3": "33",
      "custom4": "barRubicon",
      "custom5": "USD",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "333",
      "rubicon_custom1": "[\"cat331\",\"cat332\"]",
      "rubicon_custom6": "true",
      "testInput": 333
    }
  }
}`

	Bid0Ext = `{"prebid": {"foo":"bar1", "targeting": {"custom_attr":"bar1"}}}`
	Bid1Ext = `{"prebid": {"foo":"bar2", "targeting": {"custom_attr":"bar2"}}}`
	Bid2Ext = `{"prebid": {"foo":"bar3", "targeting": {"custom_attr":"bar3"}}}`

	ApnBid0Ext = `{
  "custom1": 1111,
  "custom2": "a1111",
  "prebid": {
    "foo": "bar1",
    "targeting": {
      "appnexus_custom1": "1111",
      "appnexus_imp":"imp1",
      "custom2": "a1111",
      "hb_amp_param": "testAmpKey",
      "hb_req_ext_param": "{\"primaryadserver\":1,\"publisher\":\"\",\"withcategory\":true}",
      "hb_req_imp_ext_bidde": "111",
      "hb_req_imp_ext_param": "{\"testUser\":\"user1\"}",
      "hb_req_imp_param": "10",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "seat_cur":"USD"
    }
  }
}`

	ApnBid1Ext = `{
  "custom1": 2222,
  "custom2": "a2222",
  "prebid": {
    "foo": "bar2",
    "targeting": {
      "appnexus_custom1": "2222",
      "appnexus_imp":"imp2",
      "custom2": "a2222",
      "hb_amp_param": "testAmpKey",
      "hb_req_ext_param": "{\"primaryadserver\":1,\"publisher\":\"\",\"withcategory\":true}",
      "hb_req_imp_ext_bidde": "222",
      "hb_req_imp_ext_param": "{\"testUser\":\"user2\"}",
      "hb_req_imp_param": "20",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "seat_cur":"USD"
    }
  }
}`
	ApnBid2Ext = `{
  "custom1": 3333,
  "custom2": "a3333",
  "prebid": {
    "foo": "bar3",
    "targeting": {
      "appnexus_custom1": "3333",
      "appnexus_imp":"imp3",
      "custom2": "a3333",
      "hb_amp_param": "testAmpKey",
      "hb_req_ext_param": "{\"primaryadserver\":1,\"publisher\":\"\",\"withcategory\":true}",
      "hb_req_imp_ext_bidde": "333",
      "hb_req_imp_ext_param": "{\"testUser\":\"user3\"}",
      "hb_req_imp_param": "30",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "seat_cur":"USD"
    }
  }
}`

	RbcBid0Ext = `{
  "custom1": 4444,
  "custom2": "r4444",
  "prebid": {
    "foo": "bar11",
    "targeting": {
      "custom2": "r4444",
      "rubicon_imp":"imp1",
      "hb_amp_param": "testAmpKey",
      "hb_req_ext_param": "{\"primaryadserver\":1,\"publisher\":\"\",\"withcategory\":true}",
      "hb_req_imp_ext_bidde": "111",
      "hb_req_imp_ext_param": "{\"testUser\":\"user1\"}",
      "hb_req_imp_param": "10",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "rubicon_custom1": "4444",
      "testInput": 111,
      "seat_cur":"USD"
    }
  }
}`
	RbcBid1Ext = `{
  "custom1": 5555,
  "custom2": "r5555",
  "prebid": {
    "foo": "bar22",
    "targeting": {
      "custom2": "r5555",
      "rubicon_imp":"imp2",
      "hb_amp_param": "testAmpKey",
      "hb_req_ext_param": "{\"primaryadserver\":1,\"publisher\":\"\",\"withcategory\":true}",
      "hb_req_imp_ext_bidde": "222",
      "hb_req_imp_ext_param": "{\"testUser\":\"user2\"}",
      "hb_req_imp_param": "20",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "rubicon_custom1": "5555",
      "testInput": 222,
      "seat_cur":"USD"
    }
  }
}`
	RbcBid2Ext = `{
  "custom1": 6666,
  "custom2": "r6666",
  "prebid": {
    "foo": "bar33",
    "targeting": {
      "custom2": "r6666",
      "rubicon_imp":"imp3",
      "hb_amp_param": "testAmpKey",
      "hb_req_ext_param": "{\"primaryadserver\":1,\"publisher\":\"\",\"withcategory\":true}",
      "hb_req_imp_ext_bidde": "333",
      "hb_req_imp_ext_param": "{\"testUser\":\"user3\"}",
      "hb_req_imp_param": "30",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "rubicon_custom1": "6666",
      "testInput": 333,
      "seat_cur":"USD"
    }
  }
}`
)
