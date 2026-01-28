package sparteo

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adapters/adapterstest"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuilder verifies that the Builder function correctly creates a bidder instance
// and that the endpoint template renders fields using the new macros.
func TestBuilder(t *testing.T) {
	cfg := config.Adapter{
		Endpoint: "https://bid-test.sparteo.com/s2s-auction",
	}
	bidder, err := Builder(openrtb_ext.BidderSparteo, cfg, config.Server{})
	if assert.NoError(t, err) {
		assert.NotNil(t, bidder)
	}
}

// TestJsonSamples runs JSON sample tests using the shared adapterstest framework.
func TestJsonSamples(t *testing.T) {
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{
		Endpoint: "https://bid-test.sparteo.com/s2s-auction",
	}, config.Server{GvlID: 1028})
	require.NoError(t, err, "Builder returned an error")

	adapterstest.RunJSONBidderTest(t, "sparteotest", bidder)
}

// TestGetMediaType_InvalidJSON verifies that getMediaType returns an error and an empty result
// when the extension JSON is invalid.
func TestGetMediaType_InvalidJSON(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`invalid-json`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for invalid JSON")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for invalid JSON")
}

// TestGetMediaType_EmptyType verifies that getMediaType returns an error and an empty result
// when the extension JSON is valid but the "type" field is empty.
func TestGetMediaType_EmptyType(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: json.RawMessage(`{"prebid":{"type":""}}`),
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for empty type")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for empty type")
}

// TestGetMediaType_NilExt verifies that getMediaType returns an error and an empty result
// when the bid's extension is nil.
func TestGetMediaType_NilExt(t *testing.T) {
	adapter := &adapter{}
	bid := &openrtb2.Bid{
		Ext: nil,
	}
	result, err := adapter.getMediaType(bid)
	assert.Error(t, err, "Expected error for nil extension")
	assert.Equal(t, openrtb_ext.BidType(""), result, "Expected empty result for nil extension")
}

// TestMakeRequests_ResolvesQueryParams verifies that the adapter correctly resolves macros in the endpoint URL for Site traffic.
func TestMakeRequests_ResolvesQueryParams(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{
		Endpoint: endpoint,
	}, config.Server{GvlID: 1028})
	require.NoError(t, err, "Builder returned an error")

	in := &openrtb2.BidRequest{
		ID: "req-qp-1",
		Site: &openrtb2.Site{
			Domain: "dev.sparteo.com",
		},
		Imp: []openrtb2.Imp{
			{
				ID:  "imp-1",
				Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`),
			},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1, "Expected exactly one outgoing request")
	assert.Empty(t, errs, "Unexpected adapter errors")

	expectedURI := "https://bid-test.sparteo.com/s2s-auction?network_id=networkID&site_domain=dev.sparteo.com"
	assert.Equal(t, expectedURI, reqs[0].Uri)
}

// TestMakeRequests_AppBundleMacro verifies that the adapter correctly resolves app_domain and bundle for App traffic.
func TestMakeRequests_AppBundleMacro(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-app-bundle-1",
		App: &openrtb2.App{
			Domain: "dev.sparteo.com",
			Bundle: "com.sparteo.app",
			Publisher: &openrtb2.Publisher{
				ID: "sparteo",
			},
		},
		Imp: []openrtb2.Imp{{
			ID:  "imp-1",
			Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	assert.Empty(t, errs)

	expected := "https://bid-test.sparteo.com/s2s-auction?network_id=networkID&app_domain=dev.sparteo.com&bundle=com.sparteo.app"
	assert.Equal(t, expected, reqs[0].Uri, "endpoint should include app_domain and bundle for App traffic")
}

// TestMakeRequests_AppBundleMissing_AppendsUnknown verifies that when app.bundle is empty or missing we send bundle=unknown (and warn).
func TestMakeRequests_AppBundleMissing_AppendsUnknown(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-app-nobundle-1",
		App: &openrtb2.App{
			Domain:    "dev.sparteo.com",
			Publisher: &openrtb2.Publisher{ID: "sparteo"},
		},
		Imp: []openrtb2.Imp{{
			ID:  "imp-1",
			Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Len(t, errs, 1)

	expected := "https://bid-test.sparteo.com/s2s-auction?network_id=networkID&app_domain=dev.sparteo.com&bundle=unknown"
	assert.Equal(t, expected, reqs[0].Uri, "endpoint should contain bundle=unknown when app.bundle is empty")
}

// TestMakeRequests_SiteDomain verifies that the adapter uses site.Domain and does not include app fields for Site traffic.
func TestMakeRequests_SiteDomain(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{
		Endpoint: endpoint,
	}, config.Server{GvlID: 1028})
	require.NoError(t, err, "Builder returned an error")

	in := &openrtb2.BidRequest{
		ID: "req-fallback-1",
		Site: &openrtb2.Site{
			Domain: "dev.sparteo.com",
		},
		Imp: []openrtb2.Imp{
			{
				ID:  "imp-1",
				Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`),
			},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1, "Expected exactly one outgoing request")
	assert.Empty(t, errs, "Unexpected adapter errors")

	expectedURI := "https://bid-test.sparteo.com/s2s-auction?network_id=networkID&site_domain=dev.sparteo.com"
	assert.Equal(t, expectedURI, reqs[0].Uri)
}

// TestMakeRequests_DomainPrecedence_SiteDomainWins verifies that Site wins over App when both exist.
func TestMakeRequests_DomainPrecedence_SiteDomainWins(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-domain-1",
		Site: &openrtb2.Site{
			Domain: "site.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "site-pub.sparteo.com",
			},
		},
		App: &openrtb2.App{
			Domain: "app.sparteo.com",
			Publisher: &openrtb2.Publisher{
				Domain: "app-pub.should-not-be-used",
			},
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	assert.Equal(t,
		"https://bid-test.sparteo.com/s2s-auction?network_id=networkID&site_domain=site.sparteo.com",
		reqs[0].Uri,
	)
}

// TestMakeRequests_SitePresentWithEmptyDomain_UsesUnknownNotApp verifies that when Site exists but has no domain/page, we send site_domain=unknown and do NOT use App.
func TestMakeRequests_SitePresentWithEmptyDomain_UsesUnknownNotApp(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-domain-2",
		Site: &openrtb2.Site{
			Domain: "",
			Publisher: &openrtb2.Publisher{
				Domain: "dev.sparteo.com",
			},
		},
		App: &openrtb2.App{
			Domain: "app.sparteo.com",
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Len(t, errs, 1)

	assert.Equal(t,
		"https://bid-test.sparteo.com/s2s-auction?network_id=networkID&site_domain=unknown",
		reqs[0].Uri,
	)
}

// TestMakeRequests_DomainPrecedence_AppDomainWhenNoSite verifies that if there is no Site, we use app.domain (and include bundle if present).
func TestMakeRequests_DomainPrecedence_AppDomainWhenNoSite(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID:   "req-domain-3",
		Site: nil,
		App: &openrtb2.App{
			Domain: "app.sparteo.com",
			Bundle: "com.sparteo.app",
			Publisher: &openrtb2.Publisher{
				Domain: "app-pub.should-not-be-used",
			},
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	assert.Equal(t,
		"https://bid-test.sparteo.com/s2s-auction?network_id=networkID&app_domain=app.sparteo.com&bundle=com.sparteo.app",
		reqs[0].Uri,
	)
}

// TestMakeRequests_SitePageWhenNoSiteDomainNoApp verifies that if there is Site without domain but page is present, we use site.page.
func TestMakeRequests_SitePageWhenNoSiteDomainNoApp(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-domain-page",
		Site: &openrtb2.Site{
			Domain: "",
			Page:   "https://www.dev.sparteo.com:3000/some/path?x=1",
		},
		App: nil,
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	assert.Equal(t,
		"https://bid-test.sparteo.com/s2s-auction?network_id=networkID&site_domain=dev.sparteo.com",
		reqs[0].Uri,
	)
}

// TestNormalizeHostname_PortAndPathAndURL verifies normalizeHostname behavior.
func TestNormalizeHostname_PortAndPathAndURL(t *testing.T) {
	base := "dev.sparteo.com"

	tests := []struct {
		name string
		in   string
		out  string
	}{
		// Bare host + port
		{"host_with_port", "www." + base + ":8080", base},
		{"host_upper_with_tls_port", "DEV.SPARTEO.COM:443", base},
		{"host_trailing_dot_with_port", base + ".:8443", base},

		// Bare host + path
		{"host_with_path", base + "/some/path?x=1", base},
		{"www_host_with_path", "www." + base + "/p", base},

		// Bare host + port + path
		{"host_with_port_and_path", base + ":8080/p?q=1", base},
		{"www_host_with_port_and_path", "www." + base + ":3000/some/path", base},

		// Absolute URLs
		{"https_url", "https://www." + base + "/x", base},
		{"http_url_with_port", "http://WWW." + base + ":8080/abc", base},

		// Scheme-relative URL
		{"scheme_relative", "//www." + base + "/x", base},

		// Odd spacing around URLs/hosts
		{"spaced_https_url", "   https://www." + base + ":3000/x  ", base},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeHostname(tc.in)
			assert.Equal(t, tc.out, got)
		})
	}
}

// --- Helper for reading publisher.ext.params.networkId in output JSON ---
func readNetworkIDFromPublisherExt(t *testing.T, ext json.RawMessage) (string, bool) {
	if len(ext) == 0 {
		return "", false
	}
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(ext, &m))
	params, _ := m["params"].(map[string]interface{})
	if params == nil {
		return "", false
	}
	val, _ := params["networkId"].(string)
	if val == "" {
		return "", false
	}
	return val, true
}

// When site exists but site.publisher is nil, the adapter must create publisher and upsert networkId into site.publisher.ext
func TestMakeRequests_UpdatePublisherExtension_CreatesSitePublisherIfMissing(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-upsert-site",
		Site: &openrtb2.Site{
			Domain:    "site.sparteo.com",
			Publisher: nil,
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(reqs[0].Body, &out))

	require.NotNil(t, out.Site)
	require.NotNil(t, out.Site.Publisher, "publisher should be created when missing")
	val, ok := readNetworkIDFromPublisherExt(t, out.Site.Publisher.Ext)
	require.True(t, ok, "site.publisher.ext.params.networkId should exist")
	assert.Equal(t, "networkID", val)
}

// When no site is present, create app.publisher if missing and upsert networkId into app.publisher.ext
func TestMakeRequests_UpdatePublisherExtension_CreatesAppPublisherIfMissingWhenNoSite(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID:   "req-upsert-app",
		Site: nil,
		App: &openrtb2.App{
			Domain:    "app.sparteo.com",
			Bundle:    "com.sparteo.app",
			Publisher: nil,
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(reqs[0].Body, &out))

	require.Nil(t, out.Site)
	require.NotNil(t, out.App)
	require.NotNil(t, out.App.Publisher, "app.publisher should be created when missing")
	val, ok := readNetworkIDFromPublisherExt(t, out.App.Publisher.Ext)
	require.True(t, ok, "app.publisher.ext.params.networkId should exist")
	assert.Equal(t, "networkID", val)
}

// TestMakeRequests_UpdatePublisherExtension_PrefersSiteOverApp verifies that we prefer Site when both Site and App exist; only Site should receive the networkId
func TestMakeRequests_UpdatePublisherExtension_PrefersSiteOverApp(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-upsert-prefer-site",
		Site: &openrtb2.Site{
			Domain:    "site.sparteo.com",
			Publisher: &openrtb2.Publisher{},
		},
		App: &openrtb2.App{
			Domain:    "app.sparteo.com",
			Publisher: &openrtb2.Publisher{},
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(reqs[0].Body, &out))

	require.NotNil(t, out.Site)
	require.NotNil(t, out.Site.Publisher)
	siteVal, siteOk := readNetworkIDFromPublisherExt(t, out.Site.Publisher.Ext)
	require.True(t, siteOk)
	assert.Equal(t, "networkID", siteVal)

	if out.App != nil && out.App.Publisher != nil {
		_, appOk := readNetworkIDFromPublisherExt(t, out.App.Publisher.Ext)
		assert.False(t, appOk, "app.publisher.ext should not contain networkId when site exists")
	}
}

// TestMakeRequests_NoSiteNoApp_NoDomainParam_NoWarning verifies that with neither Site nor App we don't send domain params and produce no warnings.
func TestMakeRequests_NoSiteNoApp_NoDomainParam_NoWarning(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-domain-missing",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})

	require.Len(t, reqs, 1)
	require.Len(t, errs, 0)

	assert.Equal(t,
		"https://bid-test.sparteo.com/s2s-auction?network_id=networkID",
		reqs[0].Uri,
	)
}

// TestMakeRequests_SitePageLiteralNull_TreatedAsMissingDomain verifies that a literal "null" in site.page is treated as an unknown site domain.
func TestMakeRequests_SitePageLiteralNull_TreatedAsMissingDomain(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s-auction"

	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		ID: "req-page-null",
		Site: &openrtb2.Site{
			Domain: "",
			Page:   "null",
		},
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)},
		},
	}

	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})

	require.Len(t, reqs, 1)
	require.Len(t, errs, 1)
	assert.Equal(t,
		"https://bid-test.sparteo.com/s2s-auction?network_id=networkID&site_domain=unknown",
		reqs[0].Uri,
	)

	var badInput *errortypes.BadInput
	require.True(t, errors.As(errs[0], &badInput))
	assert.Contains(t, badInput.Error(), "Domain not found")
}

// TestMakeRequests_AppBundle_NormalizationVariants ensures blank/whitespace/"null" bundles normalize to "unknown" with a single warning.
func TestMakeRequests_AppBundle_NormalizationVariants(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	cases := []struct {
		name   string
		bundle string
	}{
		{"empty", ""},
		{"spaces", "   "},
		{"lower_null", "null"},
		{"weird_case", "NuLl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &openrtb2.BidRequest{
				App: &openrtb2.App{Domain: "dev.sparteo.com", Bundle: tc.bundle, Publisher: &openrtb2.Publisher{}},
				Imp: []openrtb2.Imp{{ID: "i1", Ext: json.RawMessage(`{"bidder":{"networkId":"n"}}`)}},
			}
			reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
			require.Len(t, reqs, 1)
			require.Len(t, errs, 1)
			assert.Contains(t, reqs[0].Uri, "app_domain=dev.sparteo.com")
			assert.Contains(t, reqs[0].Uri, "bundle=unknown")
			var badInput *errortypes.BadInput
			require.True(t, errors.As(errs[0], &badInput))
			assert.Contains(t, badInput.Error(), "Bundle not found")
		})
	}
}

// TestMakeRequests_SitePresent_IgnoresAppFields ensures when Site is present, only site_domain is sent and app fields are ignored.
func TestMakeRequests_SitePresent_IgnoresAppFields(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "site.sparteo.com"},
		App:  &openrtb2.App{Domain: "app.sparteo.com", Bundle: "com.app"},
		Imp:  []openrtb2.Imp{{ID: "i1", Ext: json.RawMessage(`{"bidder":{"networkId":"N"}}`)}},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)
	assert.Equal(t, "https://bid-test.sparteo.com/s2s?network_id=N&site_domain=site.sparteo.com", reqs[0].Uri)
}

// TestMakeRequests_SiteWithPageFallback_IgnoresApp ensures Site.Page is used when Site.Domain is empty and App is ignored when Site exists.
func TestMakeRequests_SiteWithPageFallback_IgnoresApp(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "", Page: "https://www.example.com:8080/p"},
		App:  &openrtb2.App{Domain: "app.sparteo.com", Bundle: "com.app"},
		Imp:  []openrtb2.Imp{{ID: "i1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)}},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)
	assert.Equal(t, "https://bid-test.sparteo.com/s2s?network_id=networkID&site_domain=example.com", reqs[0].Uri)
}

// TestMakeRequests_AppMissingBundle_WarnsAndUnknown ensures App with missing bundle yields bundle=unknown and one bundle warning.
func TestMakeRequests_AppMissingBundle_WarnsAndUnknown(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		App: &openrtb2.App{Domain: "dev.sparteo.com", Bundle: "", Publisher: &openrtb2.Publisher{}},
		Imp: []openrtb2.Imp{{ID: "i1", Ext: json.RawMessage(`{"bidder":{"networkId":"networkID"}}`)}},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Len(t, errs, 1)
	assert.Equal(t, "https://bid-test.sparteo.com/s2s?network_id=networkID&app_domain=dev.sparteo.com&bundle=unknown", reqs[0].Uri)
	var badInput *errortypes.BadInput
	require.True(t, errors.As(errs[0], &badInput))
	assert.Contains(t, badInput.Error(), "Bundle not found")
}

// TestMakeRequests_ExtRewrite_MovesBidderIntoSparteoParams verifies bidder object is moved under sparteo.params and bidder is removed.
func TestMakeRequests_ExtRewrite_MovesBidderIntoSparteoParams(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "dev.sparteo.com"},
		Imp:  []openrtb2.Imp{{ID: "i1", Ext: json.RawMessage(`{"bidder":{"a":1}}`)}},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(reqs[0].Body, &out))
	var ext map[string]any
	require.NoError(t, json.Unmarshal(out.Imp[0].Ext, &ext))
	_, hasBidder := ext["bidder"]
	assert.False(t, hasBidder)
	sparteoNode, ok := ext["sparteo"].(map[string]any)
	require.True(t, ok)
	params, ok := sparteoNode["params"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(1), params["a"])
}

// TestMakeRequests_ExtRewrite_PreservesExistingSparteoParams verifies existing sparteo.params keys are preserved when merging bidder fields.
func TestMakeRequests_ExtRewrite_PreservesExistingSparteoParams(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "dev.sparteo.com"},
		Imp: []openrtb2.Imp{{
			ID:  "i1",
			Ext: json.RawMessage(`{"sparteo":{"params":{"keep":true}},"bidder":{"add":42}}`),
		}},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(reqs[0].Body, &out))
	var ext map[string]any
	require.NoError(t, json.Unmarshal(out.Imp[0].Ext, &ext))
	sparteoNode, ok := ext["sparteo"].(map[string]any)
	require.True(t, ok)
	params, ok := sparteoNode["params"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, true, params["keep"])
	assert.Equal(t, float64(42), params["add"])
}

// TestMakeRequests_NetworkId_FirstNonEmptyWins verifies the first non-empty networkId across imps is used.
func TestMakeRequests_NetworkId_FirstNonEmptyWins(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "dev.sparteo.com"},
		Imp: []openrtb2.Imp{
			{ID: "i1", Ext: json.RawMessage(`{"bidder":{"networkId":""}}`)},
			{ID: "i2", Ext: json.RawMessage(`{"bidder":{"networkId":"N2"}}`)},
			{ID: "i3", Ext: json.RawMessage(`{"bidder":{"networkId":"N3"}}`)},
		},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)
	assert.Equal(t, "https://bid-test.sparteo.com/s2s?network_id=N2&site_domain=dev.sparteo.com", reqs[0].Uri)
}

// TestMakeRequests_AllImpsBadExt_AggregatesErrors verifies a request is still built when all imps have bad ext and errors are aggregated.
func TestMakeRequests_AllImpsBadExt_AggregatesErrors(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "dev.sparteo.com"},
		Imp: []openrtb2.Imp{
			{ID: "i1", Ext: json.RawMessage(`not-json`)},
			{ID: "i2", Ext: json.RawMessage(`{"bidder":}`)},
		},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Len(t, errs, 2)
	assert.Equal(t, "https://bid-test.sparteo.com/s2s?network_id=&site_domain=dev.sparteo.com", reqs[0].Uri)
}

// TestMakeRequests_UpdatePublisherExtension_PreservesOtherKeys verifies publisher.ext keeps existing keys while adding params.networkId.
func TestMakeRequests_UpdatePublisherExtension_PreservesOtherKeys(t *testing.T) {
	endpoint := "https://bid-test.sparteo.com/s2s"
	bidder, err := Builder(openrtb_ext.BidderSparteo, config.Adapter{Endpoint: endpoint}, config.Server{GvlID: 1028})
	require.NoError(t, err)

	startExt := json.RawMessage(`{"params":{"foo":"bar"},"other":123}`)
	in := &openrtb2.BidRequest{
		Site: &openrtb2.Site{
			Domain:    "dev.sparteo.com",
			Publisher: &openrtb2.Publisher{Ext: startExt},
		},
		Imp: []openrtb2.Imp{{ID: "i1", Ext: json.RawMessage(`{"bidder":{"networkId":"N"}}`)}},
	}
	reqs, errs := bidder.MakeRequests(in, &adapters.ExtraRequestInfo{})
	require.Len(t, reqs, 1)
	require.Empty(t, errs)

	var out openrtb2.BidRequest
	require.NoError(t, json.Unmarshal(reqs[0].Body, &out))
	var m map[string]any
	require.NoError(t, json.Unmarshal(out.Site.Publisher.Ext, &m))
	params, _ := m["params"].(map[string]any)
	require.NotNil(t, params)
	assert.Equal(t, "bar", params["foo"])
	assert.Equal(t, float64(123), m["other"])
	assert.Equal(t, "N", params["networkId"])
}

// TestMakeBids_SkipsAudioBids verifies bids with prebid.type=audio are skipped with no returned errors.
func TestMakeBids_SkipsAudioBids(t *testing.T) {
	a := &adapter{}
	body := []byte(`{"cur":"USD","seatbid":[{"bid":[{"impid":"1","price":1,"crid":"c","ext":{"prebid":{"type":"audio"}}}]}]}`)
	br, errs := a.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: 200, Body: body})
	require.NoError(t, nil)
	require.Nil(t, errs)
	require.NotNil(t, br)
	assert.Equal(t, 0, len(br.Bids))
}

// TestMakeBids_SetsMTypeAndCurrency verifies mtype is set correctly for banner/video/native and currency is propagated.
func TestMakeBids_SetsMTypeAndCurrency(t *testing.T) {
	a := &adapter{}
	body := []byte(`{
		"cur":"EUR",
		"seatbid":[{"bid":[
			{"impid":"b1","price":1,"crid":"c1","ext":{"prebid":{"type":"banner"}}},
			{"impid":"v1","price":2,"crid":"c2","ext":{"prebid":{"type":"video"}}},
			{"impid":"n1","price":3,"crid":"c3","ext":{"prebid":{"type":"native"}}}
		]}]
	}`)
	br, errs := a.MakeBids(&openrtb2.BidRequest{}, &adapters.RequestData{}, &adapters.ResponseData{StatusCode: 200, Body: body})
	require.Nil(t, errs)
	require.NotNil(t, br)
	require.Equal(t, "EUR", br.Currency)
	require.Equal(t, 3, len(br.Bids))
	assert.Equal(t, openrtb2.MarkupBanner, br.Bids[0].Bid.MType)
	assert.Equal(t, openrtb2.MarkupVideo, br.Bids[1].Bid.MType)
	assert.Equal(t, openrtb2.MarkupNative, br.Bids[2].Bid.MType)
}
