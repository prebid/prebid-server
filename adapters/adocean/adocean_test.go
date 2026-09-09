package adocean

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"text/template"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

const testBidID = "30470a14-2949-4110-abce-b62d57304ad5"

type testUUIDGenerator struct {
	id  string
	err error
}

func (g testUUIDGenerator) Generate() (string, error) {
	return g.id, g.err
}

type jsonTestBidder struct {
	adapters.Bidder
	requestsByURI map[string]*adapters.RequestData
}

func (b *jsonTestBidder) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requests, errs := b.Bidder.MakeRequests(request, requestInfo)
	for _, requestData := range requests {
		b.requestsByURI[requestData.Uri] = requestData
	}
	return requests, errs
}

func (b *jsonTestBidder) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if requestData, found := b.requestsByURI[externalRequest.Uri]; found {
		externalRequest = requestData
	}
	return b.Bidder.MakeBids(internalRequest, externalRequest, response)
}

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
	bidder.(*adapter).uuidGenerator = testUUIDGenerator{id: testBidID}

	adapterstest.RunJSONBidderTest(t, "adoceantest", &jsonTestBidder{
		Bidder:        bidder,
		requestsByURI: make(map[string]*adapters.RequestData),
	})
}

func TestMakeRequestURLLength(t *testing.T) {
	endpointTemplate := template.Must(template.New("endpoint").Parse("https://{{.Host}}.adocean.pl"))
	bidder := adapter{endpointTemplate: endpointTemplate}
	width, height := int64(300), int64(250)
	request := &openrtb2.BidRequest{Test: 1}
	imp := &openrtb2.Imp{
		ID: "test-imp",
		Banner: &openrtb2.Banner{
			W: &width,
			H: &height,
		},
	}
	params := &openrtb_ext.ExtImpAdOcean{
		EmitterPrefix: "myao",
		MasterID:      "a",
		SlaveID:       "adoceanmyaozpniqismex",
	}

	baseRequest, err := bidder.makeRequest(request, imp, params)
	if err != nil {
		t.Fatalf("makeRequest returned unexpected error: %v", err)
	}
	params.MasterID = strings.Repeat("a", maxUriLength-len(baseRequest.Uri)+1)

	requestAtLimit, err := bidder.makeRequest(request, imp, params)
	if err != nil {
		t.Fatalf("makeRequest should accept a URL at maxUriLength: %v", err)
	}
	if len(requestAtLimit.Uri) != maxUriLength {
		t.Fatalf("makeRequest returned URL length %d, expected %d", len(requestAtLimit.Uri), maxUriLength)
	}

	params.MasterID += "a"
	_, err = bidder.makeRequest(request, imp, params)
	if err == nil {
		t.Fatal("makeRequest should reject a URL that exceeds maxUriLength")
	}
}

func TestBuildQueryEmitterParamsCannotOverrideAdapterParams(t *testing.T) {
	request := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{GDPR: openrtb2.Int8Ptr(1)},
		User: &openrtb2.User{Consent: "consent", BuyerUID: "buyer-id"},
	}
	imp := &openrtb2.Imp{Banner: &openrtb2.Banner{
		Format: []openrtb2.Format{{W: 300, H: 250}},
	}}
	params := &openrtb_ext.ExtImpAdOcean{
		MasterID: "master-id",
		SlaveID:  "adoceanmyaozpniqismex",
		EmitterRequestParams: map[string]any{
			"pbsrv_v":      "override",
			"id":           "override",
			"slaves":       "override",
			"gdpr":         "override",
			"gdpr_consent": "override",
			"aouserid":     "override",
			"aosize":       "override",
			"custom":       "value",
		},
	}

	query, err := buildQuery(request, imp, params)
	if err != nil {
		t.Fatalf("buildQuery returned unexpected error: %v", err)
	}

	expected := map[string]string{
		"pbsrv_v":      adapterVersion,
		"id":           "master-id",
		"slaves":       "zpniqismex",
		"gdpr":         "1",
		"gdpr_consent": "consent",
		"aouserid":     "buyer-id",
		"aosize":       "300x250",
		"custom":       "value",
	}
	for key, value := range expected {
		if got := query.Get(key); got != value {
			t.Errorf("buildQuery returned %s=%q, expected %q", key, got, value)
		}
		if len(query[key]) != 1 {
			t.Errorf("buildQuery returned duplicate values for %s: %v", key, query[key])
		}
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

func TestValidateImpVideoPlacement(t *testing.T) {
	tests := []struct {
		name      string
		video     *openrtb2.Video
		wantError bool
	}{
		{name: "OpenRTB 2.6 instream", video: &openrtb2.Video{Plcmt: adcom1.VideoPlcmtInstream}},
		{name: "legacy instream", video: &openrtb2.Video{Placement: adcom1.VideoPlacementInStream}},
		{name: "missing placement", video: &openrtb2.Video{}, wantError: true},
		{name: "OpenRTB 2.6 accompanying content", video: &openrtb2.Video{Plcmt: adcom1.VideoPlcmtAccompanyingContent}, wantError: true},
		{name: "legacy in-banner", video: &openrtb2.Video{Placement: adcom1.VideoPlacementInBanner}, wantError: true},
		{
			name:  "OpenRTB 2.6 instream takes precedence over legacy outstream",
			video: &openrtb2.Video{Plcmt: adcom1.VideoPlcmtInstream, Placement: adcom1.VideoPlacementInBanner},
		},
		{
			name:      "OpenRTB 2.6 outstream takes precedence over legacy instream",
			video:     &openrtb2.Video{Plcmt: adcom1.VideoPlcmtInterstitial, Placement: adcom1.VideoPlacementInStream},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateImp(&openrtb2.Imp{ID: "video-imp", Video: test.video})
			if (err != nil) != test.wantError {
				t.Fatalf("validateImp returned error %v, wantError=%t", err, test.wantError)
			}
		})
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
	bidder := adapter{uuidGenerator: testUUIDGenerator{id: testBidID}}
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
	bid, currency, err := bidder.makeBid(valid, "imp-1")
	if err != nil {
		t.Fatalf("makeBid returned unexpected error: %v", err)
	}
	if currency != "EUR" || bid.Bid.AdM != "creative markup" || bid.BidType != openrtb_ext.BidTypeVideo || bid.Bid.ImpID != "imp-1" {
		t.Fatalf("makeBid returned unexpected bid: %+v, currency=%q", bid, currency)
	}
	if bid.Bid.ADomain == nil || bid.BidMeta.AdvertiserDomains == nil {
		t.Fatal("makeBid should provide empty advertiser-domain slices")
	}
	if bid.Bid.ID != testBidID {
		t.Fatalf("makeBid returned bid ID %q, expected %q", bid.Bid.ID, testBidID)
	}

	failingBidder := adapter{uuidGenerator: testUUIDGenerator{err: errors.New("uuid failure")}}
	_, _, err = failingBidder.makeBid(valid, "imp-1")
	if err == nil || !strings.Contains(err.Error(), "failed to generate bid ID") {
		t.Fatalf("makeBid returned unexpected UUID generation error: %v", err)
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
			_, _, err := bidder.makeBid(test.adUnit, "imp-1")
			if err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("makeBid returned unexpected error: %v", err)
			}
		})
	}
}

func TestMakeBidsUsesRequestImpIDAndRejectsInconsistentCurrency(t *testing.T) {
	bidder := adapter{uuidGenerator: testUUIDGenerator{id: testBidID}}
	responseBody, err := json.Marshal([]responseAdUnit{
		{ID: "placement-one", Currency: "EUR", Price: "1", Width: "300", Height: "250", Code: "first"},
		{ID: "placement-two", Currency: "USD", Price: "2", Width: "300", Height: "250", Code: "second"},
	})
	if err != nil {
		t.Fatalf("failed to marshal test response: %v", err)
	}

	bidderResponse, errs := bidder.MakeBids(
		&openrtb2.BidRequest{},
		&adapters.RequestData{ImpIDs: []string{"imp-from-request"}},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody},
	)

	if len(errs) != 1 || !strings.Contains(errs[0].Error(), "inconsistent currencies") {
		t.Fatalf("MakeBids returned unexpected errors: %v", errs)
	}
	if bidderResponse == nil || len(bidderResponse.Bids) != 1 {
		t.Fatalf("MakeBids returned unexpected response: %+v", bidderResponse)
	}
	if bidderResponse.Currency != "EUR" || bidderResponse.Bids[0].Bid.ImpID != "imp-from-request" {
		t.Fatalf("MakeBids returned unexpected currency or impression ID: %+v", bidderResponse)
	}
}
