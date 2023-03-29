package adservertargeting

const (
	reqValid = `{
  "id": "req_id",
  "imp": [
    {
      "id": "test_imp1",
      "ext": {"appnexus": {"placementId": 250419771}},
      "banner": {"format": [{"h": 250, "w": 300}]}
    },
    {
      "id": "test_imp2",
      "ext": {"appnexus": {"placementId": 250419771}},
      "banner": {"format": [{"h": 250, "w": 300}]}
    }
  ],
  "site": {"page": "test.com"}
}`

	reqInvalid = `{
  "id": "req_id",
  "imp": {
	 "incorrect":true
   },
  "site": {"page": "test.com"}
}`

	reqNoImps = `{
  "id": "req_id",
  "site": {"page": "test.com"}
}`

	testUrl = "https://www.test-url.com?amp-key=testAmpKey&data-override-height=400"

	reqFullValid = `{
  "id": "req_id",
  "imp": [
    {
      "id": "test_imp1",
      "ext": {"appnexus": {"placementId": 123}},
      "banner": {"format": [{"h": 250, "w": 300}], "w": 260, "h": 350}
    },
    {
      "id": "test_imp2",
      "ext": {"appnexus": {"placementId": 456}},
      "banner": {"format": [{"h": 400, "w": 600}], "w": 270, "h": 360}
    }
  ],
  "site": {"page": "test.com"}
}`

	reqFullInvalid = `{
  "imp": [
    {
      "ext": {"appnexus": {"placementId": 123}}
    },
    {
      "ext": {"appnexus": {"placementId": 456}}
    }
  ]
}`

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

	seatBid0Bid0Ext = `{
  "prebid": {
    "foo": "bar1",
    "targeting": {
      "appnexus_custom6": "true",
      "custom2": "bar1",
      "custom3": "10",
      "custom4": "barApn",
      "custom5": "USD",
      "custom6":"testResponse", 
      "custom7":"testBidId", 
      "custom8":"testCustomData", 
      "custom9":"2",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "111"
    }
  }
}`

	seatBid0Bid1Ext = `
{
  "prebid": {
    "foo": "bar2",
    "targeting": {
      "appnexus_custom6": "true",
      "custom2": "bar2",
      "custom3": "20",
      "custom4": "barApn",
      "custom5": "USD",
      "custom6":"testResponse", 
      "custom7":"testBidId", 
      "custom8":"testCustomData", 
      "custom9":"2",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "222"
    }
  }
}`
	seatBid0Bid2Ext = `
{
  "prebid": {
    "foo": "bar3",
    "targeting": {
      "appnexus_custom6": "true",
      "custom2": "bar3",
      "custom3": "30",
      "custom4": "barApn",
      "custom5": "USD",
      "custom6":"testResponse", 
      "custom7":"testBidId", 
      "custom8":"testCustomData", 
      "custom9":"2",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "333"
    }
  }
}`

	seatBid1Bid0Ext = `{
  "prebid": {
    "foo": "bar11",
    "targeting": {
      "custom2": "bar11",
      "custom3": "11",
      "custom4": "barRubicon",
      "custom5": "USD",
      "custom6":"testResponse", 
      "custom7":"testBidId", 
      "custom8":"testCustomData", 
      "custom9":"2",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "111",
      "rubicon_custom6": "true",
      "testInput": 111
    }
  }
}`
	seatBid1Bid1Ext = `
{
  "prebid": {
    "foo": "bar22",
    "targeting": {
      "custom2": "bar22",
      "custom3": "22",
      "custom4": "barRubicon",
      "custom5": "USD",
      "custom6":"testResponse", 
      "custom7":"testBidId", 
      "custom8":"testCustomData", 
      "custom9":"2",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "222",
      "rubicon_custom6": "true",
      "testInput": 222
    }
  }
}`

	seatBid1Bid2Ext = `{
  "prebid": {
    "foo": "bar33",
    "targeting": {
      "custom2": "bar33",
      "custom3": "33",
      "custom4": "barRubicon",
      "custom5": "USD",
      "custom6":"testResponse", 
      "custom7":"testBidId", 
      "custom8":"testCustomData", 
      "custom9":"2",
      "hb_amp_param": "testAmpKey",
      "hb_imp_param": "333",
      "rubicon_custom6": "true",
      "testInput": 333
    }
  }
}`

	bid0Ext = `{"prebid": {"foo":"bar1", "targeting": {"custom_attr":"bar1"}}}`
	bid1Ext = `{"prebid": {"foo":"bar2", "targeting": {"custom_attr":"bar2"}}}`
	bid2Ext = `{"prebid": {"foo":"bar3", "targeting": {"custom_attr":"bar3"}}}`

	apnBid0Ext = `{
  "custom1": 1111,
  "custom2": "a1111",
  "prebid": {
    "foo": "bar1",
    "targeting": {
      "appnexus_custom1": "1111",
      "appnexus_imp":"imp1",
      "custom2": "a1111",
      "hb_amp_param": "testAmpKey",
      "hb_req_imp_ext_bidde": "111",
      "hb_req_imp_param": "10",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "seat_cur":"USD"
    }
  }
}`

	apnBid1Ext = `{
  "custom1": 2222,
  "custom2": "a2222",
  "prebid": {
    "foo": "bar2",
    "targeting": {
      "appnexus_custom1": "2222",
      "appnexus_imp":"imp2",
      "custom2": "a2222",
      "hb_amp_param": "testAmpKey",
      "hb_req_imp_ext_bidde": "222",
      "hb_req_imp_param": "20",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "seat_cur":"USD"
    }
  }
}`
	apnBid2Ext = `{
  "custom1": 3333,
  "custom2": "a3333",
  "prebid": {
    "foo": "bar3",
    "targeting": {
      "appnexus_custom1": "3333",
      "appnexus_imp":"imp3",
      "custom2": "a3333",
      "hb_amp_param": "testAmpKey",
      "hb_req_imp_ext_bidde": "333",
      "hb_req_imp_param": "30",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "seat_cur":"USD"
    }
  }
}`

	rbcBid0Ext = `{
  "custom1": 4444,
  "custom2": "r4444",
  "prebid": {
    "foo": "bar11",
    "targeting": {
      "custom2": "r4444",
      "rubicon_imp":"imp1",
      "hb_amp_param": "testAmpKey",
      "hb_req_imp_ext_bidde": "111",
      "hb_req_imp_param": "10",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "rubicon_custom1": "4444",
      "testInput": 111,
      "seat_cur":"USD"
    }
  }
}`
	rbcBid1Ext = `{
  "custom1": 5555,
  "custom2": "r5555",
  "prebid": {
    "foo": "bar22",
    "targeting": {
      "custom2": "r5555",
      "rubicon_imp":"imp2",
      "hb_amp_param": "testAmpKey",
      "hb_req_imp_ext_bidde": "222",
      "hb_req_imp_param": "20",
      "hb_req_user_param": "2000",
      "hb_static_thing": "test-static-value",
      "rubicon_custom1": "5555",
      "testInput": 222,
      "seat_cur":"USD"
    }
  }
}`
	rbcBid2Ext = `{
  "custom1": 6666,
  "custom2": "r6666",
  "prebid": {
    "foo": "bar33",
    "targeting": {
      "custom2": "r6666",
      "rubicon_imp":"imp3",
      "hb_amp_param": "testAmpKey",
      "hb_req_imp_ext_bidde": "333",
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
