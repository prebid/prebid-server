package boldwin_rapid

import (
	"encoding/json"
	"errors"
	"testing"

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
