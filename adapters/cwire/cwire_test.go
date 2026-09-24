package cwire

import (
	"math"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderCWire,
		config.Adapter{
			Endpoint: "https://cwi.re/prebid/adapter-endpoint",
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

	adapterstest.RunJSONBidderTest(t, "cwiretest", bidder)
}

func TestMakeRequestsMarshalError(t *testing.T) {
	bidder := &adapter{}
	request := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{BidFloor: math.NaN()}},
	}

	requests, errs := bidder.MakeRequests(request, nil)

	assert.Nil(t, requests)
	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], "Error while encoding OpenRTB BidRequest: json: unsupported value: NaN")
}

func TestMakeBidsUnknownMarkupType(t *testing.T) {
	bidder := &adapter{}
	httpResponse := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"seatbid":[{"bid":[{"impid":"imp-1","mtype":0}]}]}`),
	}

	response, errs := bidder.MakeBids(nil, nil, httpResponse)

	assert.Nil(t, response)
	require.Len(t, errs, 1)
	assert.EqualError(t, errs[0], "could not define media type for C Wire impression: imp-1")
}
