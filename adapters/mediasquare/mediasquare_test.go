package mediasquare

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestMakeBids(t *testing.T) {
	a, _ := Builder("mediasquare", config.Adapter{}, config.Server{})
	tests := []struct {
		// tests inputs
		request     *openrtb2.BidRequest
		requestData *adapters.RequestData
		// tests expected-results
		response       *adapters.ResponseData
		bidderResponse *adapters.BidderResponse
		errs           []error
	}{
		{
			request:     &openrtb2.BidRequest{},
			requestData: &adapters.RequestData{},

			response:       &adapters.ResponseData{StatusCode: http.StatusBadRequest},
			bidderResponse: nil,
			errs: []error{&errortypes.BadInput{
				Message: fmt.Sprintf("<MakeBids> Unexpected status code: %d.", http.StatusBadRequest),
			}},
		},
		{
			request:     &openrtb2.BidRequest{},
			requestData: &adapters.RequestData{},

			response:       &adapters.ResponseData{StatusCode: 42},
			bidderResponse: nil,
			errs: []error{&errortypes.BadServerResponse{
				Message: fmt.Sprintf("<MakeBids> Unexpected status code: %d. Run with request.debug = 1 for more info.", 42),
			}},
		},
		{
			request:     &openrtb2.BidRequest{},
			requestData: &adapters.RequestData{},

			response:       &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte("")},
			bidderResponse: nil,
			errs: []error{&errortypes.BadServerResponse{
				Message: fmt.Sprint("<MakeBids> Bad server response: unexpected end of JSON input."),
			}},
		},
		{
			request:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "1"}, {ID: "2"}, {ID: "3"}}},
			requestData: &adapters.RequestData{},

			response: &adapters.ResponseData{StatusCode: http.StatusOK, Body: []byte(`{"id":"id-ok"}`)},
			bidderResponse: &adapters.BidderResponse{
				Currency:             "USD",
				Bids:                 []*adapters.TypedBid{},
				FledgeAuctionConfigs: nil,
			},
			errs: nil,
		},
	}

	for index, test := range tests {
		resp, errs := a.MakeBids(test.request, test.requestData, test.response)

		errsVal, _ := json.Marshal(errs)
		errsExp, _ := json.Marshal(test.errs)
		assert.Equal(t, test.bidderResponse, resp, fmt.Sprintf("resp >> index: %d.", index))
		assert.Equal(t, errsExp, errsVal, fmt.Sprintf("errs >> index: %d.", index))
	}
}

func TestMakeRequest(t *testing.T) {
	a, _ := Builder("mediasquare", config.Adapter{Endpoint: "edp-mediasquare"}, config.Server{})
	tests := []struct {
		// tests inputs
		request *openrtb2.BidRequest
		reqInfo *adapters.ExtraRequestInfo
		// tests expected-results
		result []*adapters.RequestData
		errs   []error
	}{
		{
			request: &openrtb2.BidRequest{ID: "id-ok",
				Imp: []openrtb2.Imp{
					{ID: "0"},
					{ID: "1", Ext: []byte(`{"id-1":"content-1"}`)},
					{ID: "-42", Ext: []byte(`{"prebid":-42}`)},
					{ID: "-1", Ext: []byte(`{"bidder":{}}`)},
					{ID: "-0", Ext: []byte(`{"bidder":{"owner":"owner-ok","code":0}}`), Native: &openrtb2.Native{}},
					{ID: "42", Ext: []byte(`{"bidder":{"owner":"owner-ok","code":"code-ok"}}`), Native: &openrtb2.Native{}},
				},
			},
			reqInfo: &adapters.ExtraRequestInfo{GlobalPrivacyControlHeader: "global-ok"},

			result: []*adapters.RequestData{
				{Method: "POST", Uri: "edp-mediasquare", Headers: headerList, ImpIDs: []string{"0", "1", "-42", "-1", "-0", "42"},
					Body: []byte(`{"codes":[{"adunit":"","auctionid":"id-ok","bidid":"42","code":"code-ok","owner":"owner-ok","mediatypes":{"banner":null,"video":null,"native":{"title":null,"icon":null,"image":null,"clickUrl":null,"displayUrl":null,"privacyLink":null,"privacyIcon":null,"cta":null,"rating":null,"downloads":null,"likes":null,"price":null,"saleprice":null,"address":null,"phone":null,"body":null,"body2":null,"sponsoredBy":null,"sizes":null,"type":"native"}},"floor":{"*":{}}}],"gdpr":{"consent_required":false,"consent_string":""},"type":"pbs","dsa":"","tech":{"device":null,"app":null},"test":false}`)},
			},
			errs: []error{
				errors.New("<MakeRequests> imp[ext]: is empty."),
				errors.New("<MakeRequests> imp-bidder[ext]: is empty."),
				errors.New("<MakeRequests> imp[ext]: json: cannot unmarshal number into Go struct field ExtImpBidder.prebid of type openrtb_ext.ExtImpPrebid"),
				errors.New("<MakeRequests> imp-bidder[ext]: json: cannot unmarshal number into Go struct field ImpExtMediasquare.code of type string"),
			},
		},
	}
	for index, test := range tests {
		result, errs := a.MakeRequests(test.request, test.reqInfo)

		resultBytes, _ := json.Marshal(result)
		expectedBytes, _ := json.Marshal(test.result)
		assert.Equal(t, string(expectedBytes), string(resultBytes), fmt.Sprintf("result >> index: %d.", index))
		assert.Equal(t, test.errs, errs, fmt.Sprintf("errs >> index: %d.", index))
	}

	// test reference : []error<MakeRequests> on empty request.
	_, errs := a.MakeRequests(nil, nil)
	assert.Equal(t, []error{errorWritter("<MakeRequests> request", nil, true)}, errs, "[]error<MakeRequests>")

	var msqAdapter adapter
	_, errNil := msqAdapter.makeRequest(nil, nil)
	assert.Equal(t, errorWritter("<makeRequest> msqParams", nil, true), errNil, "error<makeRequest> errNil")
	_, errChan := msqAdapter.makeRequest(nil, &MsqParameters{DSA: make(chan int)})
	assert.Equal(t, errorWritter("<makeRequest> json.Marshal", errors.New("json: unsupported type: chan int"), false), errChan, "error<makeRequest> errChan")
}
