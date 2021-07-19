package openrtb2

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var request = &openrtb2.BidRequest{
	ID: "some-id",
	Imp: []openrtb2.Imp{
		{
			ID: "my-imp-id",
			Banner: &openrtb2.Banner{
				Format: []openrtb2.Format{
					{
						W: 300,
						H: 600,
					},
				},
			},
			Instl: 1,
			Ext:   json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
		},
	},
	Device: &openrtb2.Device{
		H:   640,
		W:   320,
		Ext: json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 60, "minheightperc": 60}}}`),
	},
}

func TestInterstitial(t *testing.T) {
	myRequest := request
	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}
	targetFormat := []openrtb2.Format{
		{
			W: 300,
			H: 600,
		},
		{
			W: 250,
			H: 600,
		},
		{
			W: 300,
			H: 480,
		},
		{
			W: 180,
			H: 500,
		},
		{
			W: 300,
			H: 500,
		},
		{
			W: 300,
			H: 431,
		},
		{
			W: 300,
			H: 430,
		},
		{
			W: 200,
			H: 600,
		},
		{
			W: 202,
			H: 600,
		},
		{
			W: 300,
			H: 360,
		},
	}
	assert.Equal(t, targetFormat, myRequest.Imp[0].Banner.Format)

}
