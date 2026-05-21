package openrtb2

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
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

var requestWithoutPrebidDeviceExt = &openrtb2.BidRequest{
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
		Ext: json.RawMessage(`{"field": 1}`),
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

func TestInterstitialWithoutPrebidDeviceExt(t *testing.T) {
	myRequest := requestWithoutPrebidDeviceExt
	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}
	targetFormat := []openrtb2.Format{
		{
			W: 300,
			H: 600,
		},
	}
	assert.Equal(t, targetFormat, myRequest.Imp[0].Banner.Format)
}

func TestInterstitialUsesDeviceSizeInDipsWhenFormatIsAbsent(t *testing.T) {
	myRequest := &openrtb2.BidRequest{
		ID: "some-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "my-imp-id",
				Banner: &openrtb2.Banner{},
				Instl:  1,
				Ext:    json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
			},
		},
		Device: &openrtb2.Device{
			H:       1920,
			W:       1080,
			PxRatio: 3,
			Ext:     json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 60, "minheightperc": 60}}}`),
		},
		Ext: json.RawMessage(`{"prebid":{"sdk":{"usepxratio":true}}}`),
	}

	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}

	assert.Contains(t, myRequest.Imp[0].Banner.Format, openrtb2.Format{W: 320, H: 480})
	assert.NotContains(t, myRequest.Imp[0].Banner.Format, openrtb2.Format{W: 768, H: 1024})
}

func TestInterstitialUsesDeviceSizeInDipsWhenFormatIsOneByOne(t *testing.T) {
	myRequest := &openrtb2.BidRequest{
		ID: "some-id",
		Imp: []openrtb2.Imp{
			{
				ID: "my-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 1, H: 1}},
				},
				Instl: 1,
				Ext:   json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
			},
		},
		Device: &openrtb2.Device{
			H:       1920,
			W:       1080,
			PxRatio: 3,
			Ext:     json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 60, "minheightperc": 60}}}`),
		},
		Ext: json.RawMessage(`{"prebid":{"sdk":{"usepxratio":true}}}`),
	}

	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}

	assert.Contains(t, myRequest.Imp[0].Banner.Format, openrtb2.Format{W: 320, H: 480})
	assert.NotContains(t, myRequest.Imp[0].Banner.Format, openrtb2.Format{W: 768, H: 1024})
}

func TestInterstitialDoesNotConvertExplicitFormatSizeUsingDevicePxRatio(t *testing.T) {
	myRequest := &openrtb2.BidRequest{
		ID: "some-id",
		Imp: []openrtb2.Imp{
			{
				ID: "my-imp-id",
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 400, H: 600}},
				},
				Instl: 1,
				Ext:   json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
			},
		},
		Device: &openrtb2.Device{
			H:       1920,
			W:       1080,
			PxRatio: 3,
			Ext:     json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 80, "minheightperc": 80}}}`),
		},
		Ext: json.RawMessage(`{"prebid":{"sdk":{"usepxratio":true}}}`),
	}

	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}

	assert.Equal(t, []openrtb2.Format{
		{W: 320, H: 480},
		{W: 336, H: 544},
		{W: 320, H: 568},
		{W: 320, H: 500},
		{W: 320, H: 481},
	}, myRequest.Imp[0].Banner.Format)
}

func TestInterstitialKeepsCurrentDeviceSizeBehaviorWhenUsePxRatioIsTrueAndDevicePxRatioIsAbsent(t *testing.T) {
	myRequest := &openrtb2.BidRequest{
		ID: "some-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "my-imp-id",
				Banner: &openrtb2.Banner{},
				Instl:  1,
				Ext:    json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
			},
		},
		Device: &openrtb2.Device{
			H:   1920,
			W:   1080,
			Ext: json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 1, "minheightperc": 1}}}`),
		},
		Ext: json.RawMessage(`{"prebid":{"sdk":{"usepxratio":true}}}`),
	}

	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}

	assert.Contains(t, myRequest.Imp[0].Banner.Format, openrtb2.Format{W: 768, H: 1024})
}

func TestInterstitialKeepsCurrentDeviceSizeBehaviorWhenUsePxRatioIsAbsent(t *testing.T) {
	myRequest := &openrtb2.BidRequest{
		ID: "some-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "my-imp-id",
				Banner: &openrtb2.Banner{},
				Instl:  1,
				Ext:    json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
			},
		},
		Device: &openrtb2.Device{
			H:       1920,
			W:       1080,
			PxRatio: 3,
			Ext:     json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 1, "minheightperc": 1}}}`),
		},
	}

	if err := processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}); err != nil {
		t.Fatalf("Error processing interstitials: %v", err)
	}

	assert.Contains(t, myRequest.Imp[0].Banner.Format, openrtb2.Format{W: 768, H: 1024})
}

func TestInterstitialDoesNotParseUsePxRatioWhenRequestHasNoInterstitialImps(t *testing.T) {
	myRequest := &openrtb2.BidRequest{
		ID: "some-id",
		Imp: []openrtb2.Imp{
			{
				ID:     "my-imp-id",
				Banner: &openrtb2.Banner{},
				Instl:  0,
				Ext:    json.RawMessage(`{"appnexus": {"placementId": 12883451}}`),
			},
		},
		Device: &openrtb2.Device{
			H:   1920,
			W:   1080,
			Ext: json.RawMessage(`{"prebid": {"interstitial": {"minwidthperc": 1, "minheightperc": 1}}}`),
		},
		Ext: json.RawMessage(`{"prebid":{"sdk":`),
	}

	assert.NoError(t, processInterstitials(&openrtb_ext.RequestWrapper{BidRequest: myRequest}))
}

func TestDeviceSizeToDips(t *testing.T) {
	assert.Equal(t, int64(360), deviceSizeToDips(1080, 3))
	assert.Equal(t, int64(1080), deviceSizeToDips(1080, 0))
}
