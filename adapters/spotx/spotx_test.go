package spotx

import (
	"encoding/json"
	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestSpotxMakeBid(t *testing.T) {

	var secure int8 = 1

	parmsJSON := []byte(`{
        "bidder": {
          "channel_id": "85394",
          "ad_unit": "instream",
          "secure": true,
          "ad_volume": 0.800000,
          "price_floor": 9,
          "hide_skin": false
        }
      }`)

	request := &openrtb.BidRequest{
		ID: "1559039248176",
		Imp: []openrtb.Imp{
			{
				ID: "28635736ddc2bb",
				Video: &openrtb.Video{
					MIMEs: []string{"video/3gpp"},
				},
				Secure: &secure,
				Exp:    2,
				Ext:    parmsJSON,
			},
		},
	}

	extReq := adapters.ExtraRequestInfo{}
	reqData, err := NewSpotxBidder("https://search.spotxchange.com/openrtb/2.3/dados").MakeRequests(request, &extReq)
	if err != nil {
		t.Error("Some err occurred while forming request")
		t.FailNow()
	}

	assert.Equal(t, reqData[0].Method, "POST")
	assert.Equal(t, reqData[0].Uri, "https://search.spotxchange.com/openrtb/2.3/dados/85394")
	assert.Equal(t, reqData[0].Headers.Get("Content-Type"), "application/json;charset=utf-8")

	var bodyMap map[string]interface{}
	_ = json.Unmarshal(reqData[0].Body, &bodyMap)
	assert.Equal(t, bodyMap["id"].(string), "85394")

	impMap := bodyMap["imp"].(map[string]interface{})
	assert.Equal(t, impMap["bidfloor"].(float64), float64(9))
	assert.Equal(t, impMap["secure"].(float64), float64(1))

	extMap := impMap["video"].(map[string]interface{})["ext"].(map[string]interface{})
	assert.Equal(t, extMap["ad_unit"], "instream")
	assert.Equal(t, extMap["ad_volume"], 0.8)
	assert.Equal(t, extMap["hide_skin"].(float64), float64(0))

}
