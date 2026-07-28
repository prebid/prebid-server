package adocean

import (
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
