package richaudience

import (
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

type richaudienceRequest struct {
	ID     string             `json:"id,omitempty"`
	Imp    []openrtb2.Imp     `json:"imp,omitempty"`
	User   richaudienceUser   `json:"user,omitempty"`
	Device richaudienceDevice `json:"device,omitempty"`
	Site   richaudienceSite   `json:"site,omitempty"`
	Test   int8               `json:"test,omitempty"`
}

type richaudienceUser struct {
	BuyerUID string              `json:"buyeruid,omitempty"`
	Ext      richaudienceUserExt `json:"ext,omitempty"`
}

type richaudienceUserExt struct {
	Eids    []openrtb2.EID `json:"eids,omitempty"`
	Consent string         `json:"consent,omitempty"`
}

type richaudienceDevice struct {
	IP   string `json:"ip,omitempty"`
	IPv6 string `json:"ipv6,omitempty"`
	Lmt  int8   `json:"lmt,omitempty"`
	DNT  int8   `json:"dnt,omitempty"`
	UA   string `json:"ua,omitempty"`
}

type richaudienceSite struct {
	Domain string `json:"domain,omitempty"`
	Page   string `json:"page,omitempty"`
}

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint: "https://ortb.richaudience.com/ortb/?bidder=pbs",
	}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "richaudiencetest", bidder)
}

func TestGetBuilder(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderRichaudience, config.Adapter{
		Endpoint: "https://ortb.richaudience.com/ortb/?bidder=pbs"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Errorf("error %s", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "richaudiencetest", bidder)
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

	getIsUrlSecure(raBidRequest)

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

	getIsUrlSecure(raBidRequest)
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
	}, config.Server{})

	assert.NoError(t, buildErr)
	assert.Empty(t, bidder)
}
