package info

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestPrepareBiddersDetailResponse(t *testing.T) {
	bidderAInfo := config.BidderInfo{Endpoint: "https://secureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "bidderA"}}
	bidderAResponse := []byte(`{"status":"ACTIVE","usesHttps":true,"maintainer":{"email":"bidderA"}}`)

	bidderBInfo := config.BidderInfo{Endpoint: "http://unsecureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "bidderB"}}
	bidderBResponse := []byte(`{"status":"ACTIVE","usesHttps":false,"maintainer":{"email":"bidderB"}}`)

	allResponseBidderA := bytes.Buffer{}
	allResponseBidderA.WriteString(`{"a":`)
	allResponseBidderA.Write(bidderAResponse)
	allResponseBidderA.WriteString(`}`)

	allResponseBidderAB := bytes.Buffer{}
	allResponseBidderAB.WriteString(`{"a":`)
	allResponseBidderAB.Write(bidderAResponse)
	allResponseBidderAB.WriteString(`,"b":`)
	allResponseBidderAB.Write(bidderBResponse)
	allResponseBidderAB.WriteString(`}`)

	var testCases = []struct {
		name              string
		givenBidders      config.BidderInfos
		expectedResponses map[string][]byte
	}{
		{
			name:              "none",
			givenBidders:      config.BidderInfos{},
			expectedResponses: map[string][]byte{"all": []byte(`{}`)},
		},
		{
			name:              "one",
			givenBidders:      config.BidderInfos{"a": bidderAInfo},
			expectedResponses: map[string][]byte{"a": bidderAResponse, "all": allResponseBidderA.Bytes()},
		},
		{
			name:              "many",
			givenBidders:      config.BidderInfos{"a": bidderAInfo, "b": bidderBInfo},
			expectedResponses: map[string][]byte{"a": bidderAResponse, "b": bidderBResponse, "all": allResponseBidderAB.Bytes()},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			responses, err := prepareBiddersDetailResponse(test.givenBidders)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedResponses, responses)
		})
	}
}

func TestMapDetails(t *testing.T) {
	var (
		bidderAInfo   = config.BidderInfo{Endpoint: "https://secureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "bidderA"}}
		bidderADetail = bidderDetail{Status: "ACTIVE", UsesHTTPS: ptrutil.ToPtr(true), Maintainer: &maintainer{Email: "bidderA"}}

		bidderBInfo   = config.BidderInfo{Endpoint: "http://unsecureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "bidderB"}}
		bidderBDetail = bidderDetail{Status: "ACTIVE", UsesHTTPS: ptrutil.ToPtr(false), Maintainer: &maintainer{Email: "bidderB"}}
	)

	var testCases = []struct {
		name            string
		givenBidders    config.BidderInfos
		expectedDetails map[string]bidderDetail
	}{
		{
			name:            "none",
			givenBidders:    config.BidderInfos{},
			expectedDetails: map[string]bidderDetail{},
		},
		{
			name:            "one",
			givenBidders:    config.BidderInfos{"a": bidderAInfo},
			expectedDetails: map[string]bidderDetail{"a": bidderADetail},
		},
		{
			name:            "many",
			givenBidders:    config.BidderInfos{"a": bidderAInfo, "b": bidderBInfo},
			expectedDetails: map[string]bidderDetail{"a": bidderADetail, "b": bidderBDetail},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			details := mapDetails(test.givenBidders)
			assert.Equal(t, test.expectedDetails, details)
		})
	}
}

func TestMarshalDetailsResponse(t *testing.T) {
	// Verifies omitempty is working correctly for bidderDetail, maintainer, capabilities, and aliasOf.
	bidderDetailA := bidderDetail{Status: "ACTIVE", Maintainer: &maintainer{Email: "bidderA"}}
	bidderDetailAResponse := []byte(`{"status":"ACTIVE","maintainer":{"email":"bidderA"}}`)

	// Verifies omitempty is working correctly for capabilities.app / capabilities.site.
	bidderDetailB := bidderDetail{Status: "ACTIVE", Maintainer: &maintainer{Email: "bidderB"}, Capabilities: &capabilities{App: &platform{MediaTypes: []string{"banner"}}}}
	bidderDetailBResponse := []byte(`{"status":"ACTIVE","maintainer":{"email":"bidderB"},"capabilities":{"app":{"mediaTypes":["banner"]}}}`)

	var testCases = []struct {
		name             string
		givenDetails     map[string]bidderDetail
		expectedResponse map[string][]byte
	}{
		{
			name:             "none",
			givenDetails:     map[string]bidderDetail{},
			expectedResponse: map[string][]byte{},
		},
		{
			name:             "one",
			givenDetails:     map[string]bidderDetail{"a": bidderDetailA},
			expectedResponse: map[string][]byte{"a": bidderDetailAResponse},
		},
		{
			name:             "many",
			givenDetails:     map[string]bidderDetail{"a": bidderDetailA, "b": bidderDetailB},
			expectedResponse: map[string][]byte{"a": bidderDetailAResponse, "b": bidderDetailBResponse},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			response, err := marshalDetailsResponse(test.givenDetails)
			assert.NoError(t, err)
			assert.Equal(t, test.expectedResponse, response)
		})
	}
}

func TestMarshalAllResponse(t *testing.T) {
	responses := map[string][]byte{
		"a": []byte(`{"Status":"ACTIVE"}`),
		"b": []byte(`{"Status":"DISABLED"}`),
	}

	result, err := marshalAllResponse(responses)

	assert.NoError(t, err)
	assert.Equal(t, []byte(`{"a":{"Status":"ACTIVE"},"b":{"Status":"DISABLED"}}`), result)
}

func TestMapDetailFromConfig(t *testing.T) {
	trueValue := true
	falseValue := false

	var testCases = []struct {
		name            string
		givenBidderInfo config.BidderInfo
		expected        bidderDetail
	}{
		{
			name: "enabled-all-values",
			givenBidderInfo: config.BidderInfo{
				Endpoint: "http://anyEndpoint",
				Disabled: false,
				Maintainer: &config.MaintainerInfo{
					Email: "foo@bar.com",
				},
				Capabilities: &config.CapabilitiesInfo{
					App:  &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}},
					Site: &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}},
					DOOH: &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeNative}},
				},
			},
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
				Maintainer: &maintainer{
					Email: "foo@bar.com",
				},
				Capabilities: &capabilities{
					App:  &platform{MediaTypes: []string{"banner"}},
					Site: &platform{MediaTypes: []string{"video"}},
					DOOH: &platform{MediaTypes: []string{"native"}},
				},
				AliasOf: "",
			},
		},
		{
			name: "disabled-all-values",
			givenBidderInfo: config.BidderInfo{
				Endpoint: "http://anyEndpoint",
				Disabled: true,
				Maintainer: &config.MaintainerInfo{
					Email: "foo@bar.com",
				},
				Capabilities: &config.CapabilitiesInfo{
					App:  &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}},
					Site: &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}},
				},
			},
			expected: bidderDetail{
				Status:    "DISABLED",
				UsesHTTPS: nil,
				Maintainer: &maintainer{
					Email: "foo@bar.com",
				},
				Capabilities: nil,
				AliasOf:      "",
			},
		},
		{
			name: "enabled-no-values",
			givenBidderInfo: config.BidderInfo{
				Endpoint: "http://amyEndpoint",
				Disabled: false,
			},
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
			},
		},
		{
			name: "enabled-protocol-http",
			givenBidderInfo: config.BidderInfo{
				Endpoint: "http://amyEndpoint",
				Disabled: false,
			},
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
			},
		},
		{
			name: "enabled-protocol-https",
			givenBidderInfo: config.BidderInfo{
				Endpoint: "https://amyEndpoint",
				Disabled: false,
			},
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &trueValue,
			},
		},
		{
			name: "enabled-protocol-https-case-insensitive",
			givenBidderInfo: config.BidderInfo{
				Disabled: false,
				Endpoint: "https://amyEndpoint",
			},
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &trueValue,
			},
		},
		{
			name: "enabled-protocol-unknown",
			givenBidderInfo: config.BidderInfo{
				Endpoint: "endpointWithoutProtocol",
				Disabled: false,
			},
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := mapDetailFromConfig(test.givenBidderInfo)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestMapMediaTypes(t *testing.T) {
	var testCases = []struct {
		name       string
		mediaTypes []openrtb_ext.BidType
		expected   []string
	}{
		{
			name:       "nil",
			mediaTypes: nil,
			expected:   nil,
		},
		{
			name:       "none",
			mediaTypes: []openrtb_ext.BidType{},
			expected:   []string{},
		},
		{
			name:       "one",
			mediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			expected:   []string{"banner"},
		},
		{
			name:       "many",
			mediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo},
			expected:   []string{"banner", "video"},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result := mapMediaTypes(test.mediaTypes)
			assert.ElementsMatch(t, test.expected, result)
		})
	}
}

func TestBiddersDetailHandler(t *testing.T) {
	bidderAInfo := config.BidderInfo{Endpoint: "https://secureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "bidderA"}}
	bidderAResponse := []byte(`{"status":"ACTIVE","usesHttps":true,"maintainer":{"email":"bidderA"}}`)

	bidderBInfo := config.BidderInfo{Endpoint: "http://unsecureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "bidderB"}}
	bidderBResponse := []byte(`{"status":"ACTIVE","usesHttps":false,"maintainer":{"email":"bidderB"}}`)

	aliasInfo := config.BidderInfo{AliasOf: "appnexus", Endpoint: "https://secureEndpoint.com", Disabled: false, Maintainer: &config.MaintainerInfo{Email: "alias"}}
	aliasResponse := []byte(`{"status":"ACTIVE","usesHttps":true,"maintainer":{"email":"alias"},"aliasOf":"appnexus"}`)

	allResponse := bytes.Buffer{}
	allResponse.WriteString(`{"aAlias":`)
	allResponse.Write(aliasResponse)
	allResponse.WriteString(`,"appnexus":`)
	allResponse.Write(bidderAResponse)
	allResponse.WriteString(`,"rubicon":`)
	allResponse.Write(bidderBResponse)
	allResponse.WriteString(`}`)

	bidders := config.BidderInfos{"aAlias": aliasInfo, "appnexus": bidderAInfo, "rubicon": bidderBInfo}

	handler := NewBiddersDetailEndpoint(bidders)

	openrtb_ext.SetAliasBidderName("aAlias", "appnexus")

	var testCases = []struct {
		name             string
		givenBidder      string
		expectedStatus   int
		expectedHeaders  http.Header
		expectedResponse []byte
	}{
		{
			name:             "bidder-a",
			givenBidder:      "appnexus",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: bidderAResponse,
		},
		{
			name:             "bidder-b",
			givenBidder:      "rubicon",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: bidderBResponse,
		},
		{
			name:             "bidder-b-case-insensitive",
			givenBidder:      "RUBICON",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: bidderBResponse,
		},
		{
			name:             "bidder-a-alias",
			givenBidder:      "aAlias",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: aliasResponse,
		},
		{
			name:             "bidder-a-alias-case-insensitive",
			givenBidder:      "aAlias",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: aliasResponse,
		},
		{
			name:             "all-bidders",
			givenBidder:      "all",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: allResponse.Bytes(),
		},
		{
			name:             "all-bidders-case-insensitive",
			givenBidder:      "All",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: allResponse.Bytes(),
		},
		{
			name:             "invalid",
			givenBidder:      "doesntExist",
			expectedStatus:   http.StatusNotFound,
			expectedHeaders:  http.Header{},
			expectedResponse: []byte{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			responseRecorder := httptest.NewRecorder()
			handler(responseRecorder, nil, httprouter.Params{{
				Key:   "bidderName",
				Value: test.givenBidder,
			}})

			result := responseRecorder.Result()
			assert.Equal(t, test.expectedStatus, result.StatusCode, "statuscode")

			resultBody, _ := io.ReadAll(result.Body)
			fmt.Println(string(test.expectedResponse))
			assert.Equal(t, test.expectedResponse, resultBody, "body")

			resultHeaders := result.Header
			assert.Equal(t, test.expectedHeaders, resultHeaders, "headers")
		})
	}
}
