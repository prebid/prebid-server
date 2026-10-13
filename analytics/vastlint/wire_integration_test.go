//go:build vastlint_integration

package vastlint

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
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
	const prefix = `<VAST version="2.0"><Ad id="1"><InLine><AdSystem>Publisher</AdSystem><AdTitle>Ad</AdTitle><Impression id="imp1"><![CDATA[https://example.com/imp]]></Impression><Creatives><Creative><Linear><Duration>00:00:30</Duration><MediaFiles>`
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

func TestRustPolicyAndCoverage(t *testing.T) {
	cfg := enabledConfig()
	cfg.Endpoint = os.Getenv("VASTLINT_TEST_ENDPOINT")
	require.NotEmpty(t, cfg.Endpoint)
	m, err := NewModule(cfg)
	require.NoError(t, err)
	defer m.Shutdown()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	health, err := grpc_health_v1.NewHealthClient(m.conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	require.NoError(t, err)
	require.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, health.Status)
	for xml, want := range map[string]string{`<VAST version="1.0"><Ad><MediaFile/></Ad></VAST>`: "unsupported", `<VAST version="9.9"/>`: "unsupported", `<Other/>`: "unsupported", `<VAST version="2.0">`: "malformed", `<VAST version="2.0"/>`: "supported"} {
		reply, err := m.client.Validate(ctx, &api.ValidateRequest{Document: xml, SummaryOnly: true})
		require.NoError(t, err)
		require.Equal(t, want, reply.InputStatus)
		require.True(t, validDigest(reply.RulePolicyDigest))
		require.True(t, validProfile(reply.RuleProfile))
	}
	canceled, stop := context.WithCancel(context.Background())
	stop()
	_, err = m.client.Validate(canceled, &api.ValidateRequest{Document: `<VAST version="2.0"/>`, SummaryOnly: true})
	require.Equal(t, codes.Canceled, status.Code(err))
	_, err = m.client.Validate(ctx, &api.ValidateRequest{Document: `<VAST version="2.0"/>`, SummaryOnly: true}, grpc.MaxCallRecvMsgSize(8))
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	reply, err := m.client.Validate(ctx, &api.ValidateRequest{Document: `<VAST version="2.0"/>`, SummaryOnly: true})
	require.NoError(t, err)
	require.True(t, reply.SummaryOnly)
}

func TestRustHostRuleLevels(t *testing.T) {
	level := os.Getenv("VASTLINT_TEST_RULE_LEVEL")
	if level == "" {
		t.Skip("the compatibility runner supplies each host rule profile")
	}
	cfg := enabledConfig()
	cfg.Endpoint = os.Getenv("VASTLINT_TEST_ENDPOINT")
	m, err := NewModule(cfg)
	require.NoError(t, err)
	defer m.Shutdown()
	const control = `<VAST version="2.0"><Ad id="1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Impression id="imp1"><![CDATA[https://example.com/imp]]></Impression><Creatives><Creative><Linear><Duration>00:00:30</Duration><MediaFiles><MediaFile delivery="progressive" type="video/mp4" width="640" height="360">https://example.com/ad.mp4</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`
	defect := strings.Replace(control, `<Impression id="imp1"><![CDATA[https://example.com/imp]]></Impression>`, "", 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ctx = metadata.AppendToOutgoingContext(ctx, "x-vastlint-caller", "prebid-policy-probe")
	before, err := m.client.Validate(ctx, &api.ValidateRequest{Document: control, SummaryOnly: true})
	require.NoError(t, err)
	after, err := m.client.Validate(ctx, &api.ValidateRequest{Document: defect, SummaryOnly: true})
	require.NoError(t, err)
	require.Equal(t, before.RulePolicyDigest, after.RulePolicyDigest)
	expected := proto.Clone(before.Verdict.Summary).(*api.Summary)
	switch level {
	case "error", "default":
		expected.Errors++
	case "warning":
		expected.Warnings++
	case "info":
		expected.Infos++
	case "off":
	default:
		t.Fatalf("unknown test level %q", level)
	}
	require.Equal(t, expected.Errors, after.Verdict.Summary.Errors)
	require.Equal(t, expected.Warnings, after.Verdict.Summary.Warnings)
	require.Equal(t, expected.Infos, after.Verdict.Summary.Infos)
	fixed, err := m.client.Validate(ctx, &api.ValidateRequest{Document: control, SummaryOnly: true})
	require.NoError(t, err)
	require.True(t, proto.Equal(before.Verdict.Summary, fixed.Verdict.Summary), "repairing the creative restores its original findings")
	url := os.Getenv("VASTLINT_TEST_METRICS_URL")
	require.NotEmpty(t, url)
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(url)
	require.NoError(t, err)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	severity := level
	if severity == "default" {
		severity = "error"
	}
	matched := false
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "vastlint_grpc_rule_documents_total{") && strings.Contains(line, `caller="prebid-policy-probe"`) && strings.Contains(line, `rule_id="VAST-2.0-inline-impression"`) {
			matched = true
			require.Contains(t, line, `severity="`+severity+`"`)
			require.True(t, strings.HasSuffix(line, " 1"))
		}
	}
	require.Equal(t, level != "off", matched)
	require.Contains(t, string(body), "vastlint_grpc_policy_info{")
	t.Logf("rule=%s profile=%s policy=%s", level, after.RuleProfile, after.RulePolicyDigest)
}

func TestRustInFlightCancellation(t *testing.T) {
	cfg := enabledConfig()
	cfg.Endpoint = os.Getenv("VASTLINT_TEST_ENDPOINT")
	metricsURL := os.Getenv("VASTLINT_TEST_METRICS_URL")
	require.NotEmpty(t, cfg.Endpoint)
	require.NotEmpty(t, metricsURL)
	cfg.MaxDocumentBytes = 128 * 1024
	m, err := NewModule(cfg)
	require.NoError(t, err)
	defer m.Shutdown()
	client := &http.Client{Timeout: time.Second}
	workers := func(want string) bool {
		response, err := client.Get(metricsURL)
		if err != nil {
			return false
		}
		defer response.Body.Close()
		body, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
		return err == nil && strings.Contains(string(body), "\nvastlint_grpc_active_workers "+want+"\n")
	}
	require.Eventually(t, func() bool { return workers("0") }, 3*time.Second, time.Millisecond)
	// Enough findings keep the CPU worker observable without raising input limits.
	xml := `<VAST version="2.0"><Ad><InLine><Creatives><Creative><Linear><MediaFiles>` + strings.Repeat(`<MediaFile/>`, 8000) + `</MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := m.client.Validate(ctx, &api.ValidateRequest{Document: xml, SummaryOnly: true})
		result <- err
	}()
	require.Eventually(t, func() bool { return workers("1") }, 3*time.Second, time.Millisecond, "observe actual Rust CPU work before canceling")
	cancel()
	select {
	case err := <-result:
		require.Equal(t, codes.Canceled, status.Code(err))
	case <-time.After(time.Second):
		t.Fatal("canceled RPC did not return")
	}
	require.Eventually(t, func() bool { return workers("0") }, 5*time.Second, time.Millisecond, "canceled work must eventually release its CPU permit")
	ctx, stop := context.WithTimeout(context.Background(), time.Second)
	defer stop()
	reply, err := m.client.Validate(ctx, &api.ValidateRequest{Document: `<VAST version="2.0"/>`, SummaryOnly: true})
	require.NoError(t, err)
	require.True(t, reply.SummaryOnly)
}
