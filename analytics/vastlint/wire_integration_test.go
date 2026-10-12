//go:build vastlint_integration

package vastlint

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
	"github.com/stretchr/testify/require"
)

// This opt-in suite must run against a separately built Rust service.
func TestRustValidatorWireContract(t *testing.T) {
	endpoint := os.Getenv("VASTLINT_TEST_ENDPOINT")
	require.NotEmpty(t, endpoint, "set VASTLINT_TEST_ENDPOINT for the Rust wire suite")
	cfg := enabledConfig()
	cfg.Endpoint = endpoint
	cfg.TimeoutMS = 1000
	cfg.MaxAgeMS = 5000
	module, err := NewModule(cfg)
	require.NoError(t, err)
	defer module.Shutdown()
	control := `<VAST version="2.0"><Ad id="1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Impression><![CDATA[https://example.com/imp]]></Impression><Creatives><Creative><Linear><Duration>00:00:30</Duration><MediaFiles><MediaFile delivery="progressive" type="video/mp4" width="640" height="360">https://cdn.example.com/ad.mp4</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`
	cases := []struct {
		name, xml string
		valid     bool
	}{
		{"valid", control, true},
		{"unclosed_root", strings.TrimSuffix(control, "</VAST>"), false},
		{"duplicate_attribute", strings.Replace(control, `version="2.0"`, `version="2.0" version="3.0"`, 1), false},
		{"undefined_entity", strings.Replace(control, "<AdTitle>Test</AdTitle>", "<AdTitle>&undefined;</AdTitle>", 1), false},
		{"malformed_markup", "this is not XML", false},
		{"no_ad", `<VAST version="2.0"></VAST>`, true},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			response, err := module.client.Validate(ctx, &api.ValidateRequest{Document: c.xml})
			require.NoError(t, err)
			require.NotNil(t, response.Verdict)
			require.Equal(t, c.valid, response.Verdict.Valid)
			require.NotEmpty(t, response.Verdict.Provenance.EngineVersion)
			require.True(t, validDigest(response.Verdict.Provenance.CatalogDigest))
			event := eventWith(video(c.xml))
			module.LogAuctionObject(event)
			require.Equal(t, c.xml, event.Response.SeatBid[0].Bid[0].AdM)
			require.Eventually(t, func() bool { return module.Stats()["checked"] == uint64(i+1) }, 2*time.Second, time.Millisecond)
			require.Zero(t, module.Stats()["failed"])
			t.Logf("engine=%s catalog=%s", response.Verdict.Provenance.EngineVersion, response.Verdict.Provenance.CatalogDigest)
		})
	}
	// The valid control has warnings. A valid verdict does not mean no findings.
	require.EqualValues(t, 5, module.Stats()["with_findings"])
}
