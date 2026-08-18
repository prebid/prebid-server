package adocean

import (
	"encoding/json"
	"strings"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestEndpointTemplateMalformed(t *testing.T) {
	_, err := Builder(openrtb_ext.BidderAdOcean, config.Adapter{Endpoint: "{{Malformed}}"}, config.Server{})
	if err == nil {
		t.Fatal("Builder should reject a malformed endpoint template")
	}
}

func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderAdOcean, config.Adapter{
		Endpoint: "https://{{.Host}}.adocean.pl",
	}, config.Server{})
	if err != nil {
		t.Fatalf("Builder returned unexpected error: %v", err)
	}

	adapterstest.RunJSONBidderTest(t, "adoceantest", bidder)
}

func TestMakeRequestRejectsLongURL(t *testing.T) {
	endpointTemplate := template.Must(template.New("endpoint").Parse("https://{{.Host}}.adocean.pl"))
	bidder := adapter{endpointTemplate: endpointTemplate}
	width, height := int64(300), int64(250)
	_, err := bidder.makeRequest(&openrtb2.BidRequest{}, &openrtb2.Imp{
		ID: "test-imp",
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
		},
	}, &openrtb_ext.ExtImpAdOcean{
		EmitterPrefix: "myao",
		MasterID:      strings.Repeat("a", maxUriLength),
		SlaveID:       "adoceanmyaozpniqismex",
	})
	if err == nil {
		t.Fatal("makeRequest should reject a URL that exceeds maxUriLength")
	}
}

func TestResolveEndpointTemplate(t *testing.T) {
	endpointTemplate := template.Must(template.New("endpoint").Parse("https://{{.Host}}.adocean.pl"))
	bidder := adapter{endpointTemplate: endpointTemplate}
	url, err := bidder.resolveEndpointTemplate("myao")
	if err != nil {
		t.Fatalf("resolveEndpointTemplate returned unexpected error: %v", err)
	}
	expectedURL := "https://myao.adocean.pl"
	if url != expectedURL {
		t.Fatalf("resolveEndpointTemplate returned %v, expected %v", url, expectedURL)
	}
}

func TestMakeRequestsRejectsEmptyRequest(t *testing.T) {
	bidder := adapter{}
	requests, errs := bidder.MakeRequests(&openrtb2.BidRequest{}, nil)
	if requests != nil {
		t.Fatal("MakeRequests should not return requests for an empty bid request")
	}
	if len(errs) != 1 || errs[0].Error() != "No impression in the bid request" {
		t.Fatalf("MakeRequests returned unexpected errors: %v", errs)
	}
}

func TestParseImpExt(t *testing.T) {
	params, err := parseImpExt(&openrtb2.Imp{ID: "valid", Ext: json.RawMessage(`{"bidder":{"emitterPrefix":"myao","masterId":"master","slaveId":"placement"}}`)})
	if err != nil {
		t.Fatalf("parseImpExt returned unexpected error: %v", err)
	}
	if params.EmitterPrefix != "myao" || params.MasterID != "master" || params.SlaveID != "placement" {
		t.Fatalf("parseImpExt returned unexpected params: %+v", params)
	}

	_, err = parseImpExt(&openrtb2.Imp{ID: "invalid", Ext: json.RawMessage(`{`)})
	if err == nil {
		t.Fatal("parseImpExt should reject an invalid extension")
	}
}

func TestHelpers(t *testing.T) {
	if got := shortSlaveID("short"); got != "short" {
		t.Fatalf("shortSlaveID returned %q, expected short ID", got)
	}
	if got := shortSlaveID("adoceanmyaozpniqismex"); got != "zpniqismex" {
		t.Fatalf("shortSlaveID returned %q, expected last %d characters", got, slaveIDLength)
	}

	width, height := int64(300), int64(250)
	if sizes := getBannerSizes(&openrtb2.Banner{W: &width, H: &height}); len(sizes) != 1 || sizes[0] != "300x250" {
		t.Fatalf("getBannerSizes returned unexpected dimensions: %v", sizes)
	}
	if sizes := getBannerSizes(&openrtb2.Banner{}); sizes != nil {
		t.Fatalf("getBannerSizes returned %v for a banner without dimensions", sizes)
	}
}

func TestMakeBid(t *testing.T) {
	valid := responseAdUnit{
		ID:       "placement",
		Currency: "EUR",
		Price:    "1.25",
		TTL:      "",
		Width:    "300",
		Height:   "250",
		IsVideo:  true,
		Code:     "creative%20markup",
	}
	bid, currency, err := makeBid(valid, "imp-1")
	if err != nil {
		t.Fatalf("makeBid returned unexpected error: %v", err)
	}
	if currency != "EUR" || bid.Bid.AdM != "creative markup" || bid.BidType != openrtb_ext.BidTypeVideo || bid.Bid.ImpID != "imp-1" {
		t.Fatalf("makeBid returned unexpected bid: %+v, currency=%q", bid, currency)
	}
	if bid.Bid.ADomain == nil || bid.BidMeta.AdvertiserDomains == nil {
		t.Fatal("makeBid should provide empty advertiser-domain slices")
	}

	for _, test := range []struct {
		name   string
		adUnit responseAdUnit
		field  string
	}{
		{name: "incomplete", adUnit: responseAdUnit{ID: "placement"}, field: "incomplete bid"},
		{name: "invalid price", adUnit: responseAdUnit{ID: "placement", Price: "invalid", Width: "300", Height: "250", Code: "markup"}, field: "invalid price"},
		{name: "invalid width", adUnit: responseAdUnit{ID: "placement", Price: "1", Width: "invalid", Height: "250", Code: "markup"}, field: "invalid width"},
		{name: "invalid height", adUnit: responseAdUnit{ID: "placement", Price: "1", Width: "300", Height: "invalid", Code: "markup"}, field: "invalid height"},
		{name: "invalid ttl", adUnit: responseAdUnit{ID: "placement", Price: "1", TTL: "invalid", Width: "300", Height: "250", Code: "markup"}, field: "invalid ttl"},
		{name: "invalid code", adUnit: responseAdUnit{ID: "placement", Price: "1", Width: "300", Height: "250", Code: "%"}, field: "invalid code"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := makeBid(test.adUnit, "imp-1")
			if err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("makeBid returned unexpected error: %v", err)
			}
		})
	}
}
