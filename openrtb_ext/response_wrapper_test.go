package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestRebuildResponseExt(t *testing.T) {
	testCases := loadBidResponseTestCases(prebidSample1, prebidSample2)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// create required filed in the test loop to keep test declaration easier to read
			if test.respExt.extMap == nil {
				test.respExt.extMap = make(map[string]json.RawMessage)
			}
			rw := ResponseWrapper{BidResponse: &test.response, responseExt: test.respExt}
			err := rw.rebuildResponseExt()
			assert.Equal(t, test.expectedResponse, *rw.BidResponse, test.name)
			assert.Equal(t, test.expectedErr, err, test.name)
		})
	}
}

func TestGetResponseExt(t *testing.T) {
	testCases := append(loadBidResponseTestCases(prebidSample1, prebidSample2), bidResponseTestcase{
		name:    "Empty - ResponseExt",
		respExt: &ResponseExt{},
	})
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			rw := ResponseWrapper{BidResponse: &test.response, responseExt: test.respExt}
			actualRespExt, actualErr := rw.GetResponseExt()
			assert.Equal(t, test.respExt, actualRespExt, test.name)
			assert.Equal(t, test.expectedErr, actualErr, test.name)
		})
	}
}

func TestRebuildResponse(t *testing.T) {
	testCases := loadBidResponseTestCases(prebidSample1, prebidSample2)
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			rw := ResponseWrapper{BidResponse: &test.response, responseExt: test.respExt}
			if test.respExt.extMap == nil {
				test.respExt.extMap = make(map[string]json.RawMessage)
			}
			actualErr := rw.RebuildResponse()
			assert.Equal(t, test.expectedResponse, *rw.BidResponse, test.name)
			assert.Equal(t, test.expectedErr, actualErr, test.name)
		})
	}
}

func TestResponseExtMarshal(t *testing.T) {
	testCases := loadResponseExtTestCases()
	for _, test := range testCases {
		if test.name != "Populated - Empty prebid - Cleared" {
			continue
		}
		t.Run(test.name, func(t *testing.T) {
			// create required filed in the test loop to keep test declaration easier to read
			if test.responseExt.extMap == nil && !test.responseExt.extMapDirty {
				test.responseExt.extMap = make(map[string]json.RawMessage)
			}

			actualResponse, actualErr := test.responseExt.marshal()
			assert.Equal(t, test.expectedResponse, actualResponse, test.name)
			assert.Equal(t, test.expectedErr, actualErr, test.name)
		})
	}
}

func TestResponseExtUnMarshal(t *testing.T) {
	testCases := loadResponseExtTestCases()
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			re := ResponseExt{}
			extJson := json.RawMessage{}
			actualErr := re.unmarshal(extJson)
			assert.Equal(t, test.expectedResponse, re, test.name)
			assert.Equal(t, test.expectedErr, actualErr, test.name)
		})
	}
}

// == testdata

type bidResponseTestcase struct {
	name             string
	response         openrtb2.BidResponse
	respExt          *ResponseExt
	expectedResponse openrtb2.BidResponse
	expectedErr      error
}

func loadBidResponseTestCases(prebidSample1, prebidSample2 ExtResponsePrebid) []bidResponseTestcase {
	prebidSample1Json, _ := json.Marshal(struct {
		Prebid ExtResponsePrebid `json:"prebid"`
	}{Prebid: prebidSample1})

	prebidSample2Json, _ := json.Marshal(struct {
		Prebid ExtResponsePrebid `json:"prebid"`
	}{Prebid: prebidSample2})

	customExt := []byte(`{"key1": "value1"}`)
	customExtMap := make(map[string]json.RawMessage)
	json.Unmarshal(customExt, &customExtMap)

	var testCases = []bidResponseTestcase{
		{
			name:             "Empty - Not Dirty",
			response:         openrtb2.BidResponse{},
			respExt:          &ResponseExt{},
			expectedResponse: openrtb2.BidResponse{},
		},
		{
			name:             "Empty - Dirty",
			response:         openrtb2.BidResponse{},
			respExt:          &ResponseExt{prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: prebidSample1Json},
		},
		{
			name:             "Empty - Dirty - No Change",
			response:         openrtb2.BidResponse{},
			respExt:          &ResponseExt{prebid: nil, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{},
		},
		{
			name:             "Populated - Not Dirty",
			response:         openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{},
			expectedResponse: openrtb2.BidResponse{Ext: prebidSample1Json},
		},
		{
			name:             "Populated - Dirty",
			response:         openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{prebid: &prebidSample2, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: prebidSample2Json},
		},
		{
			name:             "Populated - Dirty - No Change",
			response:         openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: prebidSample1Json},
		},
		{
			name:             "Populated - Dirty - Cleared",
			response:         openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{prebid: nil, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{},
		},
		{
			name:             "Appended - Dirty",
			response:         openrtb2.BidResponse{Ext: customExt},
			respExt:          &ResponseExt{prebid: &prebidSample1, prebidDirty: true, extMap: customExtMap, extMapDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: []byte(`{"key1":"value1","prebid":{"auctiontimestamp":118}}`)},
		},
	}
	return testCases
}

var prebidSample1 = ExtResponsePrebid{AuctionTimestamp: 118}
var prebidSample2 = ExtResponsePrebid{AuctionTimestamp: 218}

type responseExtTescase struct {
	name             string
	responseExt      ResponseExt
	expectedResponse json.RawMessage
	expectedErr      error
}

func loadResponseExtTestCases() []responseExtTescase {
	customExt := []byte(`{"key1":"value1"}`)
	customExtMap := make(map[string]json.RawMessage)
	json.Unmarshal(customExt, &customExtMap)

	prebidSample1Json, _ := json.Marshal(struct {
		Prebid ExtResponsePrebid `json:"prebid"`
	}{Prebid: prebidSample1})
	prebidSample1Map := make(map[string]json.RawMessage)
	json.Unmarshal(prebidSample1Json, &prebidSample1Map)

	testCases := []responseExtTescase{
		{
			name:             "Empty - Not Dirty",
			responseExt:      ResponseExt{},
			expectedResponse: nil,
		},
		{
			name:             "Populated - Dirty ext",
			responseExt:      ResponseExt{extMap: customExtMap, extMapDirty: true},
			expectedResponse: customExt,
		},
		{
			name:             "Populated - Dirty prebid",
			responseExt:      ResponseExt{prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: prebidSample1Json,
		},
		{
			name:             "Empty - Dirty - No Change",
			responseExt:      ResponseExt{prebid: nil, prebidDirty: true},
			expectedResponse: nil,
		},
		{
			name:             "Populated - Not Dirty",
			responseExt:      ResponseExt{extMap: customExtMap},
			expectedResponse: customExt,
		},
		// {
		// 	name: "Populated - Dirty",
		// },
		{
			name:             "Populated - Dirty - No Change",
			responseExt:      ResponseExt{extMap: prebidSample1Map, prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: prebidSample1Json,
		},
		{
			name:             "Populated - Dirty - Cleared",
			responseExt:      ResponseExt{extMap: nil, extMapDirty: true},
			expectedResponse: nil,
		},
		{
			name:             "Appended - Dirty",
			responseExt:      ResponseExt{extMap: customExtMap, prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: json.RawMessage(`{"key1":"value1","prebid":{"auctiontimestamp":118}}`),
		},
		{
			name:             "Populated - Empty prebid - Cleared",
			responseExt:      ResponseExt{extMap: prebidSample1Map, prebid: &ExtResponsePrebid{}, prebidDirty: true},
			expectedResponse: nil,
		},
	}
	return testCases
}
