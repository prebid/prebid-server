package anzuprebid

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const testExtraInfo = `{"zone2Endpoint":"https://zone2.example.com/"}`

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAnzuPrebid, config.Adapter{
		Endpoint:         "https://zone1.example.com/",
		ExtraAdapterInfo: testExtraInfo},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "anzuprebidtest", bidder)
}

func TestBuilderRejectsInvalidEndpoints(t *testing.T) {
	tests := []struct {
		name      string
		endpoint  string
		extraInfo string
		wantErr   string
	}{
		{"endpoint not https", "http://zone1.example.com/", testExtraInfo, `endpoint must be an https URL, got "http://zone1.example.com/"`},
		{"endpoint without host", "https:///bid", testExtraInfo, `endpoint must be an https URL, got "https:///bid"`},
		{"endpoint unparsable", "https://bad host/", testExtraInfo, `endpoint must be an https URL, got "https://bad host/"`},
		{"endpoint with query", "https://zone1.example.com/?pid=1", testExtraInfo, `endpoint must not include a query string or fragment, got "https://zone1.example.com/?pid=1"`},
		{"endpoint with fragment", "https://zone1.example.com/#x", testExtraInfo, `endpoint must not include a query string or fragment, got "https://zone1.example.com/#x"`},
		{"extra_info missing", "https://zone1.example.com/", "", "invalid extra_info: "},
		{"extra_info not JSON", "https://zone1.example.com/", "https://zone2.example.com/", "invalid extra_info: "},
		{"zone2Endpoint missing", "https://zone1.example.com/", `{}`, `extra_info zone2Endpoint must be an https URL, got ""`},
		{"zone2Endpoint not https", "https://zone1.example.com/", `{"zone2Endpoint":"http://zone2.example.com/"}`, `extra_info zone2Endpoint must be an https URL, got "http://zone2.example.com/"`},
		{"zone2Endpoint with query", "https://zone1.example.com/", `{"zone2Endpoint":"https://zone2.example.com/?c=rtb"}`, `extra_info zone2Endpoint must not include a query string or fragment, got "https://zone2.example.com/?c=rtb"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Builder(openrtb_ext.BidderAnzuPrebid, config.Adapter{Endpoint: test.endpoint, ExtraAdapterInfo: test.extraInfo}, config.Server{})
			if assert.Error(t, err) {
				assert.True(t, strings.HasPrefix(err.Error(), test.wantErr), "got error %q, want prefix %q", err.Error(), test.wantErr)
			}
		})
	}
}

func TestEndpointsComeFromConfig(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAnzuPrebid, config.Adapter{
		Endpoint:         "https://host-zone1.example.org/bid",
		ExtraAdapterInfo: `{"zone2Endpoint":"https://host-zone2.example.org/openrtb"}`}, config.Server{})
	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	request := &openrtb2.BidRequest{
		ID: "req-1",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"zone":"zone1","queryString":"pid=abc"}}`)},
			{ID: "imp-2", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"zone":"zone2","queryString":"c=rtb&m=req&key=xyz"}}`)},
		},
	}
	requests, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})
	assert.Empty(t, errs)
	if assert.Len(t, requests, 2) {
		assert.Equal(t, "https://host-zone1.example.org/bid?pid=abc", requests[0].Uri)
		assert.Equal(t, "https://host-zone2.example.org/openrtb?c=rtb&m=req&key=xyz", requests[1].Uri)
	}
}

func TestNormalizeQueryString(t *testing.T) {
	valid := map[string]string{
		"a=1":                   "a=1",
		"pid=123":               "pid=123",
		"c=rtb&m=req&key=a-b-c": "c=rtb&m=req&key=a-b-c",
		"?pid=123":              "pid=123",
		"k=v%20w":               "k=v%20w",
		"list=1,2,3":            "list=1,2,3",
		"path=/x/y&t=a:b@c":     "path=/x/y&t=a:b@c",
		"flag":                  "flag",
	}
	for raw, want := range valid {
		got, err := normalizeQueryString(raw)
		assert.NoError(t, err, raw)
		assert.Equal(t, want, got, raw)
	}

	invalid := []string{"", "?", "a=1#frag", "a=1 b", "a=1;b=2", "a=%zz", `a="x"`, "a=<b>", "a=1\nb=2", strings.Repeat("a", 2049)}
	for _, raw := range invalid {
		_, err := normalizeQueryString(raw)
		assert.Error(t, err, "expected %q to be rejected", raw)
	}
}

func TestDetectMediaType(t *testing.T) {
	tests := map[string]openrtb_ext.BidType{
		`<VAST version="4.0"></VAST>`:                             openrtb_ext.BidTypeVideo,
		`  <?xml version="1.0"?><VAST version="3.0"></VAST>`:      openrtb_ext.BidTypeVideo,
		`<vast version="2.0"></vast>`:                             openrtb_ext.BidTypeVideo,
		`<div><Vast></Vast></div>`:                                openrtb_ext.BidTypeVideo,
		`<div>a < b <va</div>`:                                    openrtb_ext.BidTypeBanner,
		`{"native":{"assets":[]}}`:                                openrtb_ext.BidTypeNative,
		` {"ver":"1.2","assets":[{"id":1,"title":{"text":"x"}}]}`: openrtb_ext.BidTypeNative,
		`{"foo":1}`:     openrtb_ext.BidTypeBanner,
		`{not json`:     openrtb_ext.BidTypeBanner,
		`<div>ad</div>`: openrtb_ext.BidTypeBanner,
		``:              openrtb_ext.BidTypeBanner,
	}
	for adm, want := range tests {
		assert.Equal(t, want, detectMediaType(adm), "markup %q", adm)
	}
}
