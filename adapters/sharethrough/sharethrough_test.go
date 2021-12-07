package sharethrough

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

type MockStrAdServer struct {
	mockRequestFromOpenRTB func() (*adapters.RequestData, error)
	mockResponseToOpenRTB  func() (*adapters.BidderResponse, []error)

	StrOpenRTBInterface
}

func (m MockStrAdServer) requestFromOpenRTB(imp openrtb2.Imp, request *openrtb2.BidRequest, domain string) (*adapters.RequestData, error) {
	return m.mockRequestFromOpenRTB()
}

func (m MockStrAdServer) responseToOpenRTB(strRawResp []byte, btlrReq *adapters.RequestData) (*adapters.BidderResponse, []error) {
	return m.mockResponseToOpenRTB()
}

type MockStrUriHelper struct {
	mockBuildUri func() string
	mockParseUri func() (*StrAdSeverParams, error)

	StrAdServerUriInterface
}

func (m MockStrUriHelper) buildUri(params StrAdSeverParams) string {
	return m.mockBuildUri()
}

func (m MockStrUriHelper) parseUri(uri string) (*StrAdSeverParams, error) {
	return m.mockParseUri()
}

func TestNewSharethroughBidder(t *testing.T) {
	tests := map[string]struct {
		input  config.Adapter
		output SharethroughAdapter
	}{
		"Creates Sharethrough adapter": {
			input: config.Adapter{Endpoint: "test endpoint"},
			output: SharethroughAdapter{
				AdServer: StrOpenRTBTranslator{
					UriHelper: StrUriHelper{BaseURI: "test endpoint", Clock: Clock{}},
					Util:      Util{Clock: Clock{}},
					UserAgentParsers: UserAgentParsers{
						ChromeVersion:    regexp.MustCompile(`Chrome\/(?P<ChromeVersion>\d+)`),
						ChromeiOSVersion: regexp.MustCompile(`CriOS\/(?P<chromeiOSVersion>\d+)`),
						SafariVersion:    regexp.MustCompile(`Version\/(?P<safariVersion>\d+)`),
					},
				},
			},
		},
	}

	assert := assert.New(t)
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		bidder, buildErr := Builder(openrtb_ext.BidderSharethrough, test.input)

		assert.NoError(buildErr)
		assert.Equal(bidder, &test.output)
	}
}

func TestSuccessMakeRequests(t *testing.T) {
	stubReq := &adapters.RequestData{
		Method: "POST",
		Uri:    "http://test.com",
		Body:   nil,
		Headers: http.Header{
			"Content-Type": []string{"text/plain;charset=utf-8"},
			"Accept":       []string{"application/json"},
		},
	}

	tests := map[string]struct {
		input    *openrtb2.BidRequest
		expected []*adapters.RequestData
	}{
		"Generates expected Request": {
			input: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Page: "test.com",
				},
				Device: &openrtb2.Device{
					UA: "Android Chome/60",
				},
				Imp: []openrtb2.Imp{{
					ID:  "abc",
					Ext: []byte(`{"pkey": "pkey", "iframe": true, "iframeSize": [10, 20]}`),
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{{H: 30, W: 40}},
					},
				}},
			},
			expected: []*adapters.RequestData{stubReq},
		},
	}

	mockAdServer := MockStrAdServer{
		mockRequestFromOpenRTB: func() (*adapters.RequestData, error) {
			return stubReq, nil
		},
	}

	adapter := SharethroughAdapter{AdServer: mockAdServer}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		output, actualErrors := adapter.MakeRequests(test.input, &adapters.ExtraRequestInfo{})

		if len(output) != 1 {
			t.Errorf("Expected one request in result, got %d\n", len(output))
			return
		}

		assertRequestDataEquals(t, testName, test.expected[0], output[0])
		if len(actualErrors) != 0 {
			t.Errorf("Expected no errors, got %d\n", len(actualErrors))
		}
	}
}

func TestFailureMakeRequests(t *testing.T) {
	tests := map[string]struct {
		input    *openrtb2.BidRequest
		expected string
	}{
		"Returns nil if failed to generate request": {
			input: &openrtb2.BidRequest{
				Site: &openrtb2.Site{
					Page: "test.com",
				},
				Device: &openrtb2.Device{
					UA: "Android Chome/60",
				},
				Imp: []openrtb2.Imp{{
					ID:  "abc",
					Ext: []byte(`{"pkey": "pkey", "iframe": true, "iframeSize": [10, 20]}`),
					Banner: &openrtb2.Banner{
						Format: []openrtb2.Format{{H: 30, W: 40}},
					},
				}},
			},
			expected: "error generating request",
		},
	}

	mockAdServer := MockStrAdServer{
		mockRequestFromOpenRTB: func() (*adapters.RequestData, error) {
			return nil, fmt.Errorf("error generating request")
		},
	}

	adapter := SharethroughAdapter{AdServer: mockAdServer}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		output, actualErrors := adapter.MakeRequests(test.input, &adapters.ExtraRequestInfo{})

		if output != nil {
			t.Errorf("Expected result to be nil, got %d elements\n", len(output))
		}
		if len(actualErrors) != 1 {
			t.Errorf("Expected one error, got %d\n", len(actualErrors))
		}
		if actualErrors[0].Error() != test.expected {
			t.Errorf("Error mismatch: expected '%s' got '%s'\n", test.expected, actualErrors[0].Error())
		}
	}
}

func TestSuccessMakeBids(t *testing.T) {
	stubBidderResponse := adapters.BidderResponse{}

	tests := map[string]struct {
		inputResponse *adapters.ResponseData
		expected      *adapters.BidderResponse
	}{
		"Returns nil,nil if ad server responded with no content": {
			inputResponse: &adapters.ResponseData{
				StatusCode: http.StatusNoContent,
			},
			expected: nil,
		},
		"Generates response if ad server responded with 200": {
			inputResponse: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{}`),
			},
			expected: &stubBidderResponse,
		},
	}

	mockAdServer := MockStrAdServer{
		mockResponseToOpenRTB: func() (*adapters.BidderResponse, []error) {
			return &stubBidderResponse, []error{}
		},
	}

	adapter := SharethroughAdapter{AdServer: mockAdServer}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		response, errors := adapter.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, test.inputResponse)
		if len(errors) > 0 {
			t.Errorf("Expected no errors, got %d\n", len(errors))
		}
		if response != test.expected {
			t.Errorf("Response mismatch: expected '%+v' got '%+v'\n", test.expected, response)
		}
	}
}

func TestFailureMakeBids(t *testing.T) {
	tests := map[string]struct {
		inputResponse *adapters.ResponseData
		expected      []error
	}{
		"Returns BadInput error if ad server responds with BadRequest": {
			inputResponse: &adapters.ResponseData{
				StatusCode: http.StatusBadRequest,
			},
			expected: []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", http.StatusBadRequest),
			}},
		},
		"Returns default error if ad server does not respond with Status OK": {
			inputResponse: &adapters.ResponseData{
				StatusCode: http.StatusInternalServerError,
			},
			expected: []error{fmt.Errorf("unexpected status code: %d. Run with request.debug = 1 for more info", http.StatusInternalServerError)},
		},
		"Passes by errors from responseToOpenRTB": {
			inputResponse: &adapters.ResponseData{
				StatusCode: http.StatusOK,
				Body:       []byte(`{}`),
			},
			expected: []error{fmt.Errorf("failed in responseToOpenRTB")},
		},
	}

	mockAdServer := MockStrAdServer{
		mockResponseToOpenRTB: func() (*adapters.BidderResponse, []error) {
			return nil, []error{fmt.Errorf("failed in responseToOpenRTB")}
		},
	}

	adapter := SharethroughAdapter{AdServer: mockAdServer}
	for testName, test := range tests {
		t.Logf("Test case: %s\n", testName)

		response, errors := adapter.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, test.inputResponse)
		if response != nil {
			t.Errorf("Expected response to be nil, got %+v\n", response)
		}
		if len(errors) != 1 {
			t.Errorf("Expected no errors, got %d\n", len(errors))
		}
		if errors[0].Error() != test.expected[0].Error() {
			t.Errorf("Error mismatch: expected '%s' got '%s'\n", test.expected[0].Error(), errors[0].Error())
		}
	}
}
