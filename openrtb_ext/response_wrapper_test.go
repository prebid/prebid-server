package openrtb_ext

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRebuildResponseExt(t *testing.T) {
	testCases := append(loadBidResponseTestCases(prebidSample1, prebidSample2), bidResponseTestcase{
		name:        "error case",
		respExt:     &mockRespExt{},
		expectedErr: errors.New("some_error"),
		mock: func(respExt *mock.Mock) {
			respExt.On("dirty").Return(true)
			respExt.On("marshal").Return(nil, errors.New("some_error"))
		},
	})
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// create required filed in the test loop to keep test declaration easier to read
			var respExt *ResponseExt
			_, mock := test.respExt.(*mockRespExt)
			if mock {
				respExt = &test.respExt.(*mockRespExt).ResponseExt
			} else {
				respExt = test.respExt.(*ResponseExt)
			}
			if mock && test.mock != nil {
				test.mock(&test.respExt.(*mockRespExt).Mock)
			}
			if respExt.extMap == nil {
				respExt.extMap = make(map[string]json.RawMessage)
			}
			rw := ResponseWrapper{BidResponse: test.response, responseExt: test.respExt}
			err := rw.rebuildResponseExt()
			var actualResponse openrtb2.BidResponse
			if rw.BidResponse != nil {
				actualResponse = *rw.BidResponse
			}
			assert.Equal(t, test.expectedResponse, actualResponse, test.name)
			assert.Equal(t, test.expectedErr, err, test.name)
		})
	}
}

func TestGetResponseExt(t *testing.T) {
	prebidSample1Json, _ := json.Marshal(struct {
		Prebid ExtResponsePrebid `json:"prebid"`
	}{Prebid: prebidSample1})
	prebidSample1Map := make(map[string]json.RawMessage)
	json.Unmarshal(prebidSample1Json, &prebidSample1Map)

	type args struct {
		resp ResponseWrapper
	}
	testCases := []struct {
		name            string
		args            args
		expectedRespExt *ResponseExt
		expectedErr     error
	}{
		{
			name:            "ResponseExt - not nil",
			args:            args{ResponseWrapper{responseExt: &ResponseExt{prebid: &prebidSample1}}},
			expectedRespExt: &ResponseExt{prebid: &prebidSample1},
		},
		{
			name:            "ResponseExt - nil, bidResponse - nil",
			args:            args{ResponseWrapper{responseExt: nil, BidResponse: nil}},
			expectedRespExt: &ResponseExt{},
		},
		{
			name:            "ResponseExt - nil, ext - nil",
			args:            args{ResponseWrapper{responseExt: nil, BidResponse: &openrtb2.BidResponse{Ext: nil}}},
			expectedRespExt: &ResponseExt{},
		},
		{
			name:            "ResponseExt - nil, bidResponse.Ext - non-nil",
			args:            args{ResponseWrapper{responseExt: nil, BidResponse: &openrtb2.BidResponse{Ext: prebidSample1Json}}},
			expectedRespExt: &ResponseExt{extMap: prebidSample1Map, prebid: &prebidSample1},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			actualRespExt, actualErr := test.args.resp.GetResponseExt()
			assert.Equal(t, test.expectedRespExt, actualRespExt, test.name)
			assert.Equal(t, test.expectedErr, actualErr, test.name)
		})
	}
}

func TestRebuildResponse(t *testing.T) {
	testCases := append(loadBidResponseTestCases(prebidSample1, prebidSample2), bidResponseTestcase{
		name:     "nil bid repsonse",
		response: nil,
		respExt:  &ResponseExt{},
	})
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			rw := ResponseWrapper{BidResponse: test.response, responseExt: test.respExt}
			respExt := test.respExt.(*ResponseExt)
			if respExt.extMap == nil {
				respExt.extMap = make(map[string]json.RawMessage)
			}
			actualErr := rw.RebuildResponse()
			var actualResponse openrtb2.BidResponse
			if rw.BidResponse != nil {
				actualResponse = *rw.BidResponse
			}
			assert.Equal(t, test.expectedResponse, actualResponse, test.name)
			assert.Equal(t, test.expectedErr, actualErr, test.name)
		})
	}
}

func TestResponseExtMarshal(t *testing.T) {
	testCases := append(loadResponseExtTestCases(), responseExtTescase{
		name: "error case",
		responseExt: &ResponseExt{
			prebid:      &ExtResponsePrebid{Passthrough: []byte(`{`)},
			prebidDirty: true,
		},
		expectedErr: true,
	})

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var respExt *ResponseExt
			_, mock := test.responseExt.(*mockRespExt)
			if mock {
				respExt = &test.responseExt.(*mockRespExt).ResponseExt
			} else {
				respExt = test.responseExt.(*ResponseExt)
			}
			if mock && test.mock != nil {
				test.mock(&test.responseExt.(*mockRespExt).Mock)
			}
			// create required filed in the test loop to keep test declaration easier to read

			if respExt.extMap == nil {
				respExt.extMap = make(map[string]json.RawMessage)
			}

			actualResponse, actualErr := test.responseExt.marshal()
			assert.Equal(t, test.expectedResponse, actualResponse, test.name)
			if test.expectedErr {
				assert.True(t, actualErr != nil, "expected error")
			}
		})
	}
}

func TestResponseExtUnMarshal(t *testing.T) {
	testCases := append(loadResponseExtTestCases(), responseExtTescase{
		name:            "extmap - invalid",
		responseExtJson: []byte(`{`),
		expectedErr:     true,
	}, responseExtTescase{
		name:            "prebid - invalid",
		responseExtJson: []byte(`{"prebid" : "invalid"}`),
		expectedErr:     true,
	}, responseExtTescase{
		name:        "extMap - not empty",
		responseExt: &ResponseExt{extMap: map[string]json.RawMessage{"key1": []byte(`value1`)}},
	})
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			re := ResponseExt{}
			actualErr := re.unmarshal(test.responseExtJson)
			assert.Equal(t, test.expectedResponseExt, re, test.name)
			if test.expectedErr {
				assert.True(t, actualErr != nil, test.name)
			}
		})
	}
}

func TestResponseExtSetPrebid(t *testing.T) {
	type args struct {
		prebid *ExtResponsePrebid
	}
	tests := []struct {
		name                string
		args                args
		expectedResponseExt ResponseExt
	}{
		{
			name:                "prebid - nil",
			args:                args{prebid: nil},
			expectedResponseExt: ResponseExt{prebidDirty: true, prebid: nil},
		},
		{
			name:                "prebid - object",
			args:                args{prebid: &prebidSample1},
			expectedResponseExt: ResponseExt{prebidDirty: true, prebid: &prebidSample1},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			re := &ResponseExt{}
			re.SetPrebid(test.args.prebid)
			assert.Equal(t, test.expectedResponseExt, *re, test.name)
		})
	}
}

func TestResponseExtSetExt(t *testing.T) {
	type args struct {
		ext map[string]json.RawMessage
	}
	tests := []struct {
		name                string
		args                args
		expectedResponseExt ResponseExt
	}{
		{
			name:                "ext - nil",
			args:                args{ext: nil},
			expectedResponseExt: ResponseExt{extMapDirty: true, extMap: nil},
		},
		{
			name:                "ext - object",
			args:                args{ext: map[string]json.RawMessage{}},
			expectedResponseExt: ResponseExt{extMapDirty: true, extMap: map[string]json.RawMessage{}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			re := &ResponseExt{}
			re.SetExt(test.args.ext)
			assert.Equal(t, test.expectedResponseExt, *re, test.name)
		})
	}
}

func TestResponseExtGetExt(t *testing.T) {
	type args struct{ responseExt ResponseExt }
	tests := []struct {
		name            string
		args            args
		expectedRespExt map[string]json.RawMessage
	}{
		{
			name:            "ext - empty",
			args:            args{responseExt: ResponseExt{extMap: map[string]json.RawMessage{}}},
			expectedRespExt: map[string]json.RawMessage{},
		},
		{
			name:            "ext - nil",
			args:            args{responseExt: ResponseExt{extMap: nil}},
			expectedRespExt: map[string]json.RawMessage{},
		},
		{
			name:            "ext - object",
			args:            args{responseExt: ResponseExt{extMap: map[string]json.RawMessage{"key1": []byte(`value`)}}},
			expectedRespExt: map[string]json.RawMessage{"key1": []byte(`value`)},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualResponseExt := test.args.responseExt.GetExt()
			assert.Equal(t, test.expectedRespExt, actualResponseExt, test.name)
			// ensure copy of ext is returned
			assert.NotEqual(t, &test.args.responseExt.extMap == &actualResponseExt, test.name)
		})
	}
}

func TestResponseExtGetPrebid(t *testing.T) {
	type args struct{ responseExt ResponseExt }
	tests := []struct {
		name            string
		args            args
		expectedRespExt *ExtResponsePrebid
	}{
		{
			name:            "ext - empty",
			args:            args{ResponseExt{prebid: &ExtResponsePrebid{}}},
			expectedRespExt: &ExtResponsePrebid{},
		},
		{
			name:            "ext - nil",
			args:            args{ResponseExt{prebid: nil}},
			expectedRespExt: nil,
		},
		{
			name:            "ext - object",
			args:            args{ResponseExt{prebid: &ExtResponsePrebid{AuctionTimestamp: 18}}},
			expectedRespExt: &ExtResponsePrebid{AuctionTimestamp: 18},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualResponseExt := test.args.responseExt.GetPrebid()
			assert.Equal(t, test.expectedRespExt, actualResponseExt, test.name)
			// ensure copy of ext is returned
			assert.NotEqual(t, &test.args.responseExt.prebid == &actualResponseExt, test.name)
		})
	}
}

// == testdata

type bidResponseTestcase struct {
	name                string
	response            *openrtb2.BidResponse
	respExt             iResponseExt
	expectedResponse    openrtb2.BidResponse
	expectedResponseExt iResponseExt
	expectedErr         error
	mock                func(*mock.Mock)
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
			name:                "Empty - Not Dirty",
			response:            &openrtb2.BidResponse{},
			respExt:             &ResponseExt{},
			expectedResponse:    openrtb2.BidResponse{},
			expectedResponseExt: &ResponseExt{},
		},
		{
			name:                "Empty - Dirty",
			response:            &openrtb2.BidResponse{},
			respExt:             &ResponseExt{prebid: &prebidSample1, prebidDirty: true},
			expectedResponse:    openrtb2.BidResponse{Ext: prebidSample1Json},
			expectedResponseExt: &ResponseExt{prebid: &prebidSample1, prebidDirty: true},
		},
		{
			name:                "Empty - Dirty - No Change",
			response:            &openrtb2.BidResponse{},
			respExt:             &ResponseExt{prebid: nil, prebidDirty: true},
			expectedResponse:    openrtb2.BidResponse{},
			expectedResponseExt: &ResponseExt{prebid: nil, prebidDirty: true},
		},
		{
			name:                "Populated - Not Dirty",
			response:            &openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:             &ResponseExt{},
			expectedResponse:    openrtb2.BidResponse{Ext: prebidSample1Json},
			expectedResponseExt: &ResponseExt{},
		},
		{
			name:             "Populated - Dirty",
			response:         &openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{prebid: &prebidSample2, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: prebidSample2Json},
		},
		{
			name:             "Populated - Dirty - No Change",
			response:         &openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: prebidSample1Json},
		},
		{
			name:             "Populated - Dirty - Cleared",
			response:         &openrtb2.BidResponse{Ext: prebidSample1Json},
			respExt:          &ResponseExt{prebid: nil, prebidDirty: true},
			expectedResponse: openrtb2.BidResponse{},
		},
		{
			name:             "Appended - Dirty",
			response:         &openrtb2.BidResponse{Ext: customExt},
			respExt:          &ResponseExt{prebid: &prebidSample1, prebidDirty: true, extMap: customExtMap, extMapDirty: true},
			expectedResponse: openrtb2.BidResponse{Ext: []byte(`{"key1":"value1","prebid":{"auctiontimestamp":118}}`)},
		},
	}
	return testCases
}

var prebidSample1 = ExtResponsePrebid{AuctionTimestamp: 118}
var prebidSample2 = ExtResponsePrebid{AuctionTimestamp: 218}

type responseExtTescase struct {
	name                string
	responseExt         iResponseExt
	responseExtJson     json.RawMessage
	expectedResponse    json.RawMessage
	expectedResponseExt ResponseExt
	expectedErr         bool
	mock                func(*mock.Mock)
}

func loadResponseExtTestCases() []responseExtTescase {
	var key1 = "key1"
	var value1 = "value1"
	var prebidKey = "prebid"
	customExt := []byte(fmt.Sprintf(`{"%s":"%s"}`, key1, value1))
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
			responseExt:      &ResponseExt{},
			expectedResponse: nil,

			responseExtJson:     nil,
			expectedResponseExt: ResponseExt{},
		},
		{
			name:             "Populated - Dirty ext",
			responseExt:      &ResponseExt{extMap: customExtMap, extMapDirty: true},
			expectedResponse: customExt,

			responseExtJson:     customExt,
			expectedResponseExt: ResponseExt{extMap: customExtMap, extMapDirty: false},
		},
		{
			name:             "Populated - Dirty prebid",
			responseExt:      &ResponseExt{prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: prebidSample1Json,

			responseExtJson:     prebidSample1Json,
			expectedResponseExt: ResponseExt{prebid: &prebidSample1, prebidDirty: false, extMap: prebidSample1Map, extMapDirty: false},
		},
		{
			name:             "Empty - Dirty - No Change",
			responseExt:      &ResponseExt{prebid: nil, prebidDirty: true},
			expectedResponse: nil,

			responseExtJson:     nil,
			expectedResponseExt: ResponseExt{prebid: nil, prebidDirty: false},
		},
		{
			name:             "Populated - Not Dirty",
			responseExt:      &ResponseExt{extMap: customExtMap},
			expectedResponse: customExt,

			responseExtJson:     customExt,
			expectedResponseExt: ResponseExt{extMap: customExtMap},
		},
		// {
		// 	name: "Populated - Dirty",
		// },
		{
			name:             "Populated - Dirty - No Change",
			responseExt:      &ResponseExt{extMap: prebidSample1Map, prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: prebidSample1Json,

			responseExtJson:     prebidSample1Json,
			expectedResponseExt: ResponseExt{extMap: prebidSample1Map, extMapDirty: false, prebid: &prebidSample1, prebidDirty: false},
		},
		{
			name:             "Populated - Dirty - Cleared",
			responseExt:      &ResponseExt{extMap: nil, extMapDirty: true},
			expectedResponse: nil,

			responseExtJson:     nil,
			expectedResponseExt: ResponseExt{extMap: nil},
		},
		{
			name:             "Appended - Dirty",
			responseExt:      &ResponseExt{extMap: customExtMap, prebid: &prebidSample1, prebidDirty: true},
			expectedResponse: json.RawMessage(`{"key1":"value1","prebid":{"auctiontimestamp":118}}`),

			responseExtJson: json.RawMessage(`{"key1":"value1","prebid":{"auctiontimestamp":118}}`),
			expectedResponseExt: ResponseExt{extMap: map[string]json.RawMessage{
				key1:      customExtMap[key1],
				prebidKey: prebidSample1Map[prebidKey],
			}, prebid: &prebidSample1, prebidDirty: false},
		},
		{
			name:             "Populated - Empty prebid - Cleared",
			responseExt:      &ResponseExt{extMap: prebidSample1Map, prebid: &ExtResponsePrebid{}, prebidDirty: true},
			expectedResponse: nil,

			responseExtJson:     nil,
			expectedResponseExt: ResponseExt{extMap: nil, extMapDirty: false, prebid: nil, prebidDirty: false},
		},
	}
	return testCases
}

type mockRespExt struct {
	ResponseExt
	mock.Mock
	prebid string
}

func (m *mockRespExt) marshal() (json.RawMessage, error) {
	args := m.Called()
	var arg0 json.RawMessage = nil
	if args.Get(0) != nil {
		arg0 = json.RawMessage(args.String(0))
	}
	return arg0, args.Error(1)
}

func (m *mockRespExt) unmarshal(json.RawMessage) error {
	return nil
}

func (m *mockRespExt) dirty() bool {
	args := m.Called()
	return args.Bool(0)
}
