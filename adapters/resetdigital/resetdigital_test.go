package resetdigital

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/stretchr/testify/assert"
)

func TestMakeRequests(t *testing.T) {
	bidder := new(adapter)

	request := &openrtb2.BidRequest{
		ID: "12345",

		Imp: []openrtb2.Imp{{
			ID: "001",

			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{W: 300, H: 250},
				},
			},
			Ext: json.RawMessage(``),
		}},
		Site: &openrtb2.Site{
			Domain: "https://test.com",
			Page:   "https://test.com/2016/06/12",
		},
		Cur: []string{"USD"},
		Device: &openrtb2.Device{
			UA:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/72.0.3626.121 Safari/537.36",
			IP:       "127.0.0.1",
			Language: "EN",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
}

func TestMakeBids(t *testing.T) {
	request := &openrtb2.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb2.Imp{{
			ID: "test-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{{
					W: 320,
					H: 250,
				}},
			},
			Ext: json.RawMessage(`{"bidder": {
				"accountId": 2763,
				"siteId": 68780,
				"zoneId": 327642
			}}`),
		}},
		Ext: json.RawMessage(``),
	}

	requestJson, _ := json.Marshal(request)
	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    requestJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"bids":[{"bid_id":"01","imp_id":"001","cpm":10.00,"cid":"1002088","crid":"1000763-1002088","adid":"1002088","w":"300","h":"250","seat":"resetdigital","html":"<scriptsrc=\"https://data.resetdigital.co/evts?S0B=1&R0E=1&R0M=3_3&testad=US-HEADER-15&R0A=1000048_1001096_1001117_1627360746&R0P=resetio_1234_muscleandfitness.com_Site_1_Banner&R0L=*_*_*_*_*&R0D=*_*_*_*_*_*&R0B=*_*_*\"type=\"text/javascript\"></script><imagesrc='https://adsreq.resetdigital.co?brid=0000000000000001'/><imagesrc='https://sync2.resetdigital.co/hbsync?ck=0000000000000001'/>"}]}`),
	}

	bidder := new(adapter)
	bidResponse, errs := bidder.MakeBids(request, reqData, httpResp)

	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))

	assert.Equal(t, float64(10), bidResponse.Bids[0].Bid.Price,
		"Expected Price 10. Got: %s", bidResponse.Bids[0].Bid.Price)
}
