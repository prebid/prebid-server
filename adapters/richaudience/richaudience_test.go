package richaudience

import (
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint: "http://ortb.richaudience.com/ortb/?bidder=pbs",
	})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "richaudiencetest", bidder)
}

func TestGetBuilder(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint: "http://ortb.richaudience.com/ortb/?bidder=pbs"})

	if buildErr != nil {
		t.Errorf("error %s", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "richaudience", bidder)
}

func TestGetSite(t *testing.T) {
	raBidRequest := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain: "www.test.com",
		},
	}

	richaudienceRequestTest := &richaudienceRequest{
		Site: richaudienceSite{
			Domain: "www.test.com",
		},
	}

	setSite(raBidRequest, richaudienceRequestTest)

	if raBidRequest.Site.Domain != richaudienceRequestTest.Site.Domain {
		t.Errorf("error %s", richaudienceRequestTest.Site.Domain)
	}

	raBidRequest = &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Page:   "http://www.test.com/test",
			Domain: "",
		},
	}

	richaudienceRequestTest = &richaudienceRequest{
		Site: richaudienceSite{
			Domain: "",
		},
	}

	setSite(raBidRequest, richaudienceRequestTest)

	if "" == richaudienceRequestTest.Site.Domain {
		t.Errorf("error domain is diferent %s", richaudienceRequestTest.Site.Domain)
	}
}

func TestGetDevice(t *testing.T) {

	raBidRequest := &openrtb2.BidRequest{
		Device: &openrtb2.Device{
			IP: "11.222.33.44",
			UA: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
		},
	}

	richaudienceRequestTest := &richaudienceRequest{
		Device: richaudienceDevice{
			IP:  "11.222.33.44",
			Lmt: 0,
			DNT: 0,
			UA:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.114 Safari/537.36",
		},
	}

	setDevice(raBidRequest, richaudienceRequestTest)

	if raBidRequest.Device.IP != richaudienceRequestTest.Device.IP {
		t.Errorf("error %s", richaudienceRequestTest.Device.IP)
	}

	if richaudienceRequestTest.Device.Lmt == 1 {
		t.Errorf("error %v", richaudienceRequestTest.Device.Lmt)
	}

	if richaudienceRequestTest.Device.DNT == 1 {
		t.Errorf("error %v", richaudienceRequestTest.Device.DNT)
	}

	if raBidRequest.Device.UA != richaudienceRequestTest.Device.UA {
		t.Errorf("error %s", richaudienceRequestTest.Device.UA)
	}
}

func TestResponseEmpty(t *testing.T) {
	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}
	bidder := new(adapter)
	bidResponse, errs := bidder.MakeBids(nil, nil, httpResp)

	assert.Nil(t, bidResponse, "Expected Nil")
	assert.Empty(t, errs, "Errors: %d", len(errs))
}

func TestEmptyConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint:         ``,
		ExtraAdapterInfo: ``,
	})

	assert.NoError(t, buildErr)
	assert.Empty(t, bidder)
}
