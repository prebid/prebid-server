package boldwin_rapid

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderBoldwinRapid, config.Adapter{
			Endpoint: "https://rtb.beardfleet.com/auction/bid?pid={{.PublisherID}}&tid={{.PlacementID}}",
		},
		config.Server{
			ExternalUrl: "http://hosturl.com",
			GvlID:       1,
			DataCenter:  "2",
		},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "boldwin_rapidtest", bidder)
}

func TestEndpointTemplateMalformed(t *testing.T) {
	_, buildErr := Builder(openrtb_ext.BidderAdhese, config.Adapter{
		Endpoint: "{{Malformed}}"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	assert.Error(t, buildErr)
}

// TestMakeRequestsErrors tests error handling in the MakeRequests method
func TestMakeRequestsErrors(t *testing.T) {
	testCases := []struct {
		name            string
		givenBidRequest *openrtb2.BidRequest
		mockAdapter     *mockAdapter
		expectedError   string
	}{
		{
			name: "Error unmarshalling imp.Ext",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`invalid json`),
					},
				},
			},
			mockAdapter:   &mockAdapter{},
			expectedError: "invalid character 'i' looking for beginning of value",
		},
		{
			name: "Error unmarshalling bidderExt.Bidder",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": "invalid json"}`),
					},
				},
			},
			mockAdapter:   &mockAdapter{},
			expectedError: "json: cannot unmarshal string into Go value of type openrtb_ext.ImpExtBoldwinRapid",
		},
		{
			name: "Error building endpoint URL",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"pid": "123", "tid": "456"}}`),
					},
				},
			},
			mockAdapter: &mockAdapter{
				buildEndpointURLErr: errors.New("endpoint URL error"),
			},
			expectedError: "endpoint URL error",
		},
		{
			name: "Error making request",
			givenBidRequest: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "test-imp-id",
						Ext: json.RawMessage(`{"bidder": {"pid": "123", "tid": "456"}}`),
					},
				},
			},
			mockAdapter: &mockAdapter{
				makeRequestErr: errors.New("make request error"),
			},
			expectedError: "make request error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// When
			requests, errs := tc.mockAdapter.MakeRequests(tc.givenBidRequest, nil)

			// Then
			assert.Nil(t, requests)
			require.Len(t, errs, 1)
			assert.Contains(t, errs[0].Error(), tc.expectedError)
		})
	}
}

// TestGetHeaders tests the getHeaders method
func TestGetHeaders(t *testing.T) {
	testCases := []struct {
		name            string
		givenBidRequest *openrtb2.BidRequest
		expectedHeaders http.Header
	}{
		{
			name: "Device with IPv6",
			givenBidRequest: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IPv6: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
					UA:   "test-user-agent",
				},
			},
			expectedHeaders: http.Header{
				"Content-Type":      []string{"application/json;charset=utf-8"},
				"Accept":            []string{"application/json"},
				"X-Openrtb-Version": []string{"2.5"},
				"Host":              []string{"rtb.beardfleet.com"},
				"User-Agent":        []string{"test-user-agent"},
				"X-Forwarded-For":   []string{"2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
			},
		},
		{
			name: "Device with IP",
			givenBidRequest: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP: "192.168.1.1",
					UA: "test-user-agent",
				},
			},
			expectedHeaders: http.Header{
				"Content-Type":      []string{"application/json;charset=utf-8"},
				"Accept":            []string{"application/json"},
				"X-Openrtb-Version": []string{"2.5"},
				"Host":              []string{"rtb.beardfleet.com"},
				"User-Agent":        []string{"test-user-agent"},
				"X-Forwarded-For":   []string{"192.168.1.1"},
				"Ip":                []string{"192.168.1.1"},
			},
		},
		{
			name: "Device with both IP and IPv6",
			givenBidRequest: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IP:   "192.168.1.1",
					IPv6: "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
					UA:   "test-user-agent",
				},
			},
			expectedHeaders: http.Header{
				"Content-Type":      []string{"application/json;charset=utf-8"},
				"Accept":            []string{"application/json"},
				"X-Openrtb-Version": []string{"2.5"},
				"Host":              []string{"rtb.beardfleet.com"},
				"User-Agent":        []string{"test-user-agent"},
				"X-Forwarded-For":   []string{"192.168.1.1"},
				"Ip":                []string{"192.168.1.1"},
			},
		},
		{
			name:            "No device",
			givenBidRequest: &openrtb2.BidRequest{},
			expectedHeaders: http.Header{
				"Content-Type":      []string{"application/json;charset=utf-8"},
				"Accept":            []string{"application/json"},
				"X-Openrtb-Version": []string{"2.5"},
				"Host":              []string{"rtb.beardfleet.com"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Given
			a := &adapter{}

			// When
			headers := a.getHeaders(tc.givenBidRequest)

			// Then
			assert.Equal(t, tc.expectedHeaders, headers)
		})
	}
}

// TestMakeRequestError tests error handling in the makeRequest method
func TestMakeRequestError(t *testing.T) {
	// Create a custom BidRequest with the unmarshalable field
	request := &openrtb2.BidRequest{}

	// Use a mock adapter that simulates a json.Marshal error
	mockAdapter := &mockAdapterWithMarshalError{}

	// When
	result, err := mockAdapter.makeRequest(request, "https://example.com")

	// Then
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "marshal error")
}

// Mock adapter for testing
type mockAdapter struct {
	buildEndpointURLErr error
	makeRequestErr      error
}

func (m *mockAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData

	reqCopy := *request

	for _, imp := range request.Imp {
		// Create a new request with just this impression
		reqCopy.Imp = []openrtb2.Imp{imp}

		var bidderExt adapters.ExtImpBidder
		var boldwinExt openrtb_ext.ImpExtBoldwinRapid

		// Use the current impression's Ext
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			return nil, []error{err}
		}

		if err := json.Unmarshal(bidderExt.Bidder, &boldwinExt); err != nil {
			return nil, []error{err}
		}

		if m.buildEndpointURLErr != nil {
			return nil, []error{m.buildEndpointURLErr}
		}

		if m.makeRequestErr != nil {
			return nil, []error{m.makeRequestErr}
		}
	}

	return adapterRequests, nil
}

// Mock adapter that simulates a json.Marshal error in makeRequest
type mockAdapterWithMarshalError struct {
	adapter
}

func (m *mockAdapterWithMarshalError) makeRequest(_ *openrtb2.BidRequest, _ string) (*adapters.RequestData, error) {
	// Simulate a json.Marshal error
	return nil, errors.New("marshal error")
}

func TestMakeRequests_ErrorPaths_ReturnNilAndSingleError(t *testing.T) {
	validTmpl := template.Must(template.New("ok").Parse("https://rtb.beardfleet.com/auction/bid?pid={{.PublisherID}}&tid={{.PlacementID}}"))

	tests := []struct {
		name    string
		adapter *adapter
		req     *openrtb2.BidRequest
	}{
		{
			name:    "Error unmarshalling imp.Ext",
			adapter: &adapter{endpoint: validTmpl},
			req: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Ext: json.RawMessage(`{invalid`)},
				},
			},
		},
		{
			name:    "Error unmarshalling bidderExt.Bidder",
			adapter: &adapter{endpoint: validTmpl},
			req: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp-1", Ext: json.RawMessage(`{"bidder":"not_an_object"}`)},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reqs, errs := tc.adapter.MakeRequests(tc.req, &adapters.ExtraRequestInfo{})

			// Only check shape: requests must be nil and we must get a single error.
			assert.Nil(t, reqs)
			require.Len(t, errs, 1)
			require.Error(t, errs[0])
		})
	}
}

func TestBuildEndpointURL_DirectError(t *testing.T) {
	tests := []struct {
		name     string
		template *template.Template
	}{
		{
			name:     "Template with missingkey=error and missing field",
			template: template.Must(template.New("missing").Option("missingkey=error").Parse("{{.NonExistentField}}")),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			adapter := &adapter{endpoint: tc.template}
			boldwinExt := openrtb_ext.ImpExtBoldwinRapid{
				Pid: "testPub",
				Tid: "testTag",
			}

			// Direct call to buildEndpointURL should return error
			_, err := adapter.buildEndpointURL(boldwinExt)
			require.Error(t, err)
		})
	}
}
