package info

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestPrepareBiddersDetailResponse(t *testing.T) {
	bidderAInfo := config.BidderInfo{Enabled: true, Maintainer: &config.MaintainerInfo{Email: "bidderA"}}
	bidderAConfig := config.Adapter{Endpoint: "https://secureEndpoint.com"}
	bidderAResponse := []byte(`{"status":"ACTIVE","usesHttps":true,"maintainer":{"email":"bidderA"}}`)

	bidderBInfo := config.BidderInfo{Enabled: true, Maintainer: &config.MaintainerInfo{Email: "bidderB"}}
	bidderBConfig := config.Adapter{Endpoint: "http://unsecureEndpoint.com"}
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
		description        string
		givenBidders       config.BidderInfos
		givenBiddersConfig map[string]config.Adapter
		givenAliases       map[string]string
		expectedResponses  map[string][]byte
		expectedError      string
	}{
		{
			description:        "None",
			givenBidders:       config.BidderInfos{},
			givenBiddersConfig: map[string]config.Adapter{},
			givenAliases:       map[string]string{},
			expectedResponses:  map[string][]byte{"all": []byte(`{}`)},
		},
		{
			description:        "One",
			givenBidders:       config.BidderInfos{"a": bidderAInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig},
			givenAliases:       map[string]string{},
			expectedResponses:  map[string][]byte{"a": bidderAResponse, "all": allResponseBidderA.Bytes()},
		},
		{
			description:        "Many",
			givenBidders:       config.BidderInfos{"a": bidderAInfo, "b": bidderBInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig, "b": bidderBConfig},
			givenAliases:       map[string]string{},
			expectedResponses:  map[string][]byte{"a": bidderAResponse, "b": bidderBResponse, "all": allResponseBidderAB.Bytes()},
		},
		{
			description:        "Error - Map Details", // Returns error due to invalid alias.
			givenBidders:       config.BidderInfos{"a": bidderAInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig},
			givenAliases:       map[string]string{"zAlias": "z"},
			expectedError:      "base adapter z for alias zAlias not found",
		},
	}

	for _, test := range testCases {
		responses, err := prepareBiddersDetailResponse(test.givenBidders, test.givenBiddersConfig, test.givenAliases)

		if test.expectedError == "" {
			assert.Equal(t, test.expectedResponses, responses, test.description+":responses")
			assert.NoError(t, err, test.expectedError, test.description+":err")
		} else {
			assert.Empty(t, responses, test.description+":responses")
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}

func TestMapDetails(t *testing.T) {
	trueValue := true
	falseValue := false

	bidderAInfo := config.BidderInfo{Enabled: true, Maintainer: &config.MaintainerInfo{Email: "bidderA"}}
	bidderAConfig := config.Adapter{Endpoint: "https://secureEndpoint.com"}
	bidderADetail := bidderDetail{Status: "ACTIVE", UsesHTTPS: &trueValue, Maintainer: &maintainer{Email: "bidderA"}}
	aliasADetail := bidderDetail{Status: "ACTIVE", UsesHTTPS: &trueValue, Maintainer: &maintainer{Email: "bidderA"}, AliasOf: "a"}

	bidderBInfo := config.BidderInfo{Enabled: true, Maintainer: &config.MaintainerInfo{Email: "bidderB"}}
	bidderBConfig := config.Adapter{Endpoint: "http://unsecureEndpoint.com"}
	bidderBDetail := bidderDetail{Status: "ACTIVE", UsesHTTPS: &falseValue, Maintainer: &maintainer{Email: "bidderB"}}
	aliasBDetail := bidderDetail{Status: "ACTIVE", UsesHTTPS: &falseValue, Maintainer: &maintainer{Email: "bidderB"}, AliasOf: "b"}

	var testCases = []struct {
		description        string
		givenBidders       config.BidderInfos
		givenBiddersConfig map[string]config.Adapter
		givenAliases       map[string]string
		expectedDetails    map[string]bidderDetail
		expectedError      string
	}{
		{
			description:        "None",
			givenBidders:       config.BidderInfos{},
			givenBiddersConfig: map[string]config.Adapter{},
			givenAliases:       map[string]string{},
			expectedDetails:    map[string]bidderDetail{},
		},
		{
			description:        "One Core Bidder",
			givenBidders:       config.BidderInfos{"a": bidderAInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig},
			givenAliases:       map[string]string{},
			expectedDetails:    map[string]bidderDetail{"a": bidderADetail},
		},
		{
			description:        "Many Core Bidders",
			givenBidders:       config.BidderInfos{"a": bidderAInfo, "b": bidderBInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig, "b": bidderBConfig},
			givenAliases:       map[string]string{},
			expectedDetails:    map[string]bidderDetail{"a": bidderADetail, "b": bidderBDetail},
		},
		{
			description:        "One Alias",
			givenBidders:       config.BidderInfos{"a": bidderAInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig},
			givenAliases:       map[string]string{"aAlias": "a"},
			expectedDetails:    map[string]bidderDetail{"a": bidderADetail, "aAlias": aliasADetail},
		},
		{
			description:        "Many Aliases - Same Core Bidder",
			givenBidders:       config.BidderInfos{"a": bidderAInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig},
			givenAliases:       map[string]string{"aAlias1": "a", "aAlias2": "a"},
			expectedDetails:    map[string]bidderDetail{"a": bidderADetail, "aAlias1": aliasADetail, "aAlias2": aliasADetail},
		},
		{
			description:        "Many Aliases - Different Core Bidders",
			givenBidders:       config.BidderInfos{"a": bidderAInfo, "b": bidderBInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig, "b": bidderBConfig},
			givenAliases:       map[string]string{"aAlias": "a", "bAlias": "b"},
			expectedDetails:    map[string]bidderDetail{"a": bidderADetail, "b": bidderBDetail, "aAlias": aliasADetail, "bAlias": aliasBDetail},
		},
		{
			description:        "Error - Alias Without Core Bidder",
			givenBidders:       config.BidderInfos{"a": bidderAInfo},
			givenBiddersConfig: map[string]config.Adapter{"a": bidderAConfig},
			givenAliases:       map[string]string{"zAlias": "z"},
			expectedError:      "base adapter z for alias zAlias not found",
		},
	}

	for _, test := range testCases {
		details, err := mapDetails(test.givenBidders, test.givenBiddersConfig, test.givenAliases)

		if test.expectedError == "" {
			assert.Equal(t, test.expectedDetails, details, test.description+":details")
			assert.NoError(t, err, test.expectedError, test.description+":err")
		} else {
			assert.Empty(t, details, test.description+":details")
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}
	}
}

func TestResolveEndpoint(t *testing.T) {
	var testCases = []struct {
		description        string
		givenBidder        string
		givenBiddersConfig map[string]config.Adapter
		expectedEndpoint   string
	}{
		{
			description:        "Bidder Found - Uses Config Value",
			givenBidder:        "a",
			givenBiddersConfig: map[string]config.Adapter{"a": {Endpoint: "anyEndpoint"}},
			expectedEndpoint:   "anyEndpoint",
		},
		{
			description:        "Bidder Not Found - Returns Empty",
			givenBidder:        "hasNoConfig",
			givenBiddersConfig: map[string]config.Adapter{"a": {Endpoint: "anyEndpoint"}},
			expectedEndpoint:   "",
		},
	}

	for _, test := range testCases {
		result := resolveEndpoint(test.givenBidder, test.givenBiddersConfig)
		assert.Equal(t, test.expectedEndpoint, result, test.description)
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
		description      string
		givenDetails     map[string]bidderDetail
		expectedResponse map[string][]byte
	}{
		{
			description:      "None",
			givenDetails:     map[string]bidderDetail{},
			expectedResponse: map[string][]byte{},
		},
		{
			description:      "One",
			givenDetails:     map[string]bidderDetail{"a": bidderDetailA},
			expectedResponse: map[string][]byte{"a": bidderDetailAResponse},
		},
		{
			description:      "Many",
			givenDetails:     map[string]bidderDetail{"a": bidderDetailA, "b": bidderDetailB},
			expectedResponse: map[string][]byte{"a": bidderDetailAResponse, "b": bidderDetailBResponse},
		},
	}

	for _, test := range testCases {
		response, err := marshalDetailsResponse(test.givenDetails)

		assert.NoError(t, err, test.description+":err")
		assert.Equal(t, test.expectedResponse, response, test.description+":response")
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
		description     string
		givenBidderInfo config.BidderInfo
		givenEndpoint   string
		expected        bidderDetail
	}{
		{
			description: "Enabled - All Values Present",
			givenBidderInfo: config.BidderInfo{
				Enabled: true,
				Maintainer: &config.MaintainerInfo{
					Email: "foo@bar.com",
				},
				Capabilities: &config.CapabilitiesInfo{
					App:  &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}},
					Site: &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}},
				},
			},
			givenEndpoint: "http://amyEndpoint",
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
				Maintainer: &maintainer{
					Email: "foo@bar.com",
				},
				Capabilities: &capabilities{
					App:  &platform{MediaTypes: []string{"banner"}},
					Site: &platform{MediaTypes: []string{"video"}},
				},
				AliasOf: "",
			},
		},
		{
			description: "Disabled - All Values Present",
			givenBidderInfo: config.BidderInfo{
				Enabled: false,
				Maintainer: &config.MaintainerInfo{
					Email: "foo@bar.com",
				},
				Capabilities: &config.CapabilitiesInfo{
					App:  &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner}},
					Site: &config.PlatformInfo{MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeVideo}},
				},
			},
			givenEndpoint: "http://amyEndpoint",
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
			description: "Enabled - No Values Present",
			givenBidderInfo: config.BidderInfo{
				Enabled: true,
			},
			givenEndpoint: "http://amyEndpoint",
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
			},
		},
		{
			description: "Enabled - Protocol - HTTP",
			givenBidderInfo: config.BidderInfo{
				Enabled: true,
			},
			givenEndpoint: "http://amyEndpoint",
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
			},
		},
		{
			description: "Enabled - Protocol - HTTPS",
			givenBidderInfo: config.BidderInfo{
				Enabled: true,
			},
			givenEndpoint: "https://amyEndpoint",
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &trueValue,
			},
		},
		{
			description: "Enabled - Protocol - HTTPS - Case Insensitive",
			givenBidderInfo: config.BidderInfo{
				Enabled: true,
			},
			givenEndpoint: "https://amyEndpoint",
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &trueValue,
			},
		},
		{
			description: "Enabled - Protocol - Unknown",
			givenBidderInfo: config.BidderInfo{
				Enabled: true,
			},
			givenEndpoint: "endpointWithoutProtocol",
			expected: bidderDetail{
				Status:    "ACTIVE",
				UsesHTTPS: &falseValue,
			},
		},
	}

	for _, test := range testCases {
		result := mapDetailFromConfig(test.givenBidderInfo, test.givenEndpoint)
		assert.Equal(t, test.expected, result, test.description)
	}
}

func TestMapMediaTypes(t *testing.T) {
	var testCases = []struct {
		description string
		mediaTypes  []openrtb_ext.BidType
		expected    []string
	}{
		{
			description: "Nil",
			mediaTypes:  nil,
			expected:    nil,
		},
		{
			description: "None",
			mediaTypes:  []openrtb_ext.BidType{},
			expected:    []string{},
		},
		{
			description: "One",
			mediaTypes:  []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			expected:    []string{"banner"},
		},
		{
			description: "Many",
			mediaTypes:  []openrtb_ext.BidType{openrtb_ext.BidTypeBanner, openrtb_ext.BidTypeVideo},
			expected:    []string{"banner", "video"},
		},
	}

	for _, test := range testCases {
		result := mapMediaTypes(test.mediaTypes)
		assert.ElementsMatch(t, test.expected, result, test.description)
	}
}

func TestBiddersDetailHandler(t *testing.T) {
	bidderAInfo := config.BidderInfo{Enabled: true, Maintainer: &config.MaintainerInfo{Email: "bidderA"}}
	bidderAConfig := config.Adapter{Endpoint: "https://secureEndpoint.com"}
	bidderAResponse := []byte(`{"status":"ACTIVE","usesHttps":true,"maintainer":{"email":"bidderA"}}`)
	aliasAResponse := []byte(`{"status":"ACTIVE","usesHttps":true,"maintainer":{"email":"bidderA"},"aliasOf":"a"}`)

	bidderBInfo := config.BidderInfo{Enabled: true, Maintainer: &config.MaintainerInfo{Email: "bidderB"}}
	bidderBConfig := config.Adapter{Endpoint: "http://unsecureEndpoint.com"}
	bidderBResponse := []byte(`{"status":"ACTIVE","usesHttps":false,"maintainer":{"email":"bidderB"}}`)

	allResponse := bytes.Buffer{}
	allResponse.WriteString(`{"a":`)
	allResponse.Write(bidderAResponse)
	allResponse.WriteString(`,"aAlias":`)
	allResponse.Write(aliasAResponse)
	allResponse.WriteString(`,"b":`)
	allResponse.Write(bidderBResponse)
	allResponse.WriteString(`}`)

	bidders := config.BidderInfos{"a": bidderAInfo, "b": bidderBInfo}
	biddersConfig := map[string]config.Adapter{"a": bidderAConfig, "b": bidderBConfig}
	aliases := map[string]string{"aAlias": "a"}

	handler := NewBiddersDetailEndpoint(bidders, biddersConfig, aliases)

	var testCases = []struct {
		description      string
		givenBidder      string
		expectedStatus   int
		expectedHeaders  http.Header
		expectedResponse []byte
	}{
		{
			description:      "Bidder A",
			givenBidder:      "a",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: bidderAResponse,
		},
		{
			description:      "Bidder B",
			givenBidder:      "b",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: bidderBResponse,
		},
		{
			description:      "Bidder A Alias",
			givenBidder:      "aAlias",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: aliasAResponse,
		},
		{
			description:      "All Bidders",
			givenBidder:      "all",
			expectedStatus:   http.StatusOK,
			expectedHeaders:  http.Header{"Content-Type": []string{"application/json"}},
			expectedResponse: allResponse.Bytes(),
		},
		{
			description:      "All Bidders - Wrong Case",
			givenBidder:      "ALL",
			expectedStatus:   http.StatusNotFound,
			expectedHeaders:  http.Header{},
			expectedResponse: []byte{},
		},
		{
			description:      "Invalid Bidder",
			givenBidder:      "doesntExist",
			expectedStatus:   http.StatusNotFound,
			expectedHeaders:  http.Header{},
			expectedResponse: []byte{},
		},
	}

	for _, test := range testCases {
		responseRecorder := httptest.NewRecorder()
		handler(responseRecorder, nil, httprouter.Params{{"bidderName", test.givenBidder}})

		result := responseRecorder.Result()
		assert.Equal(t, result.StatusCode, test.expectedStatus, test.description+":statuscode")

		resultBody, _ := ioutil.ReadAll(result.Body)
		assert.Equal(t, test.expectedResponse, resultBody, test.description+":body")

		resultHeaders := result.Header
		assert.Equal(t, test.expectedHeaders, resultHeaders, test.description+":headers")
	}
}
