package mediasquare

import (
	"testing"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBidderMediasquare(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderMediasquare, config.Adapter{
		Endpoint: "https://pbs-front.mediasquare.fr/msq_prebid"},
		config.Server{ExternalUrl: "https://pbs-front.mediasquare.fr/msq_prebid", GvlID: 1, DataCenter: "2"})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	// MakeRequests : case request is empty.
	resp, errs := bidder.MakeRequests(nil, nil)
	expectingErrors := []error{errorWritter("<MakeRequests> request", nil, true)}
	assert.Equal(t, []*adapters.RequestData(nil), resp, "resp, was supposed to be empty result.")
	assert.Equal(t, expectingErrors, errs, "errs, was supposed to be :", expectingErrors)

	// starting json-tests.
	adapterstest.RunJSONBidderTest(t, "mediasquaretest", bidder)
}
