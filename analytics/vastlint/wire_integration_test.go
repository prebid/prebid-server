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
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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
			response, err := module.client.Validate(ctx, &api.ValidateRequest{Document: c.xml, SummaryOnly: true})
			require.NoError(t, err)
			require.True(t, response.SummaryOnly)
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

func TestRustSummaryRepliesFitReceiveCap(t *testing.T) {
	endpoint := os.Getenv("VASTLINT_TEST_ENDPOINT")
	require.NotEmpty(t, endpoint, "set VASTLINT_TEST_ENDPOINT for the Rust wire suite")
	cfg := enabledConfig()
	cfg.Endpoint = endpoint
	cfg.TimeoutMS = 2000
	cfg.MaxAgeMS = 5000
	cfg.MaxDocumentBytes = 128 * 1024
	module, err := NewModule(cfg)
	require.NoError(t, err)
	defer module.Shutdown()
	const prefix = `<VAST version="2.0"><Ad id="1"><InLine><AdSystem>Publisher</AdSystem><AdTitle>Ad</AdTitle><Impression>https://example.com/imp</Impression><Creatives><Creative><Linear><Duration>00:00:30</Duration><MediaFiles>`
	const suffix = `</MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`
	const media = `<MediaFile/>`
	small := prefix + strings.Repeat(media, 200) + suffix
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	full, err := module.client.Validate(ctx, &api.ValidateRequest{Document: small}, grpc.MaxCallRecvMsgSize(4*1024*1024))
	require.NoError(t, err)
	require.False(t, full.SummaryOnly, "legacy callers retain full replies")
	require.Greater(t, proto.Size(full), 64*1024, "fixture must expose the original receive-cap failure")
	t.Logf("full_reply_bytes=%d document_bytes=%d", proto.Size(full), len(small))
	count := (cfg.MaxDocumentBytes - len(prefix) - len(suffix)) / len(media)
	maximum := prefix + strings.Repeat(media, count)
	maximum += strings.Repeat(" ", cfg.MaxDocumentBytes-len(maximum)-len(suffix)) + suffix
	for i, xml := range []string{small, maximum} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		response, err := module.client.Validate(ctx, &api.ValidateRequest{Document: xml, SummaryOnly: true})
		cancel()
		require.NoError(t, err)
		require.True(t, response.SummaryOnly)
		require.Less(t, proto.Size(response), 1024)
		require.NotNil(t, response.Verdict)
		require.Empty(t, response.Verdict.ProtoReflect().GetUnknown(), "detailed findings must be omitted by the server")
		require.Greater(t, response.Verdict.Summary.Errors, uint32(0))
		if i == 0 {
			require.True(t, proto.Equal(full.Verdict.Summary, response.Verdict.Summary))
			require.True(t, proto.Equal(full.Verdict.Provenance, response.Verdict.Provenance))
		}
		module.LogAuctionObject(eventWith(video(xml)))
		require.Eventually(t, func() bool { return module.Stats()["checked"] == uint64(i+1) }, 3*time.Second, time.Millisecond)
		require.Zero(t, module.Stats()["failed"])
		t.Logf("document_bytes=%d summary_reply_bytes=%d errors=%d", len(xml), proto.Size(response), response.Verdict.Summary.Errors)
	}
	require.EqualValues(t, 2, module.Stats()["with_findings"])
}
