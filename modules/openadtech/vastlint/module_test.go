package vastlint

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

const missingImpression = `<VAST version="2.0"><Ad id="1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Creatives><Creative><Linear><Duration>00:00:30</Duration><MediaFiles><MediaFile delivery="progressive" type="video/mp4" width="640" height="360">https://cdn.example.com/ad.mp4</MediaFile></MediaFiles></Linear></Creative></Creatives></InLine></Ad></VAST>`

func TestLooksLikeVAST(t *testing.T) {
	require.False(t, looksLikeVAST(""))
	require.False(t, looksLikeVAST("https://vast.example/tag.xml"))
	require.True(t, looksLikeVAST(`<?xml version="1.0"?><VAST version="4.2"></VAST>`))
	require.True(t, looksLikeVAST("<vast version=\"3.0\"></vast>"))
}

func TestRejectRevenueDropsBid(t *testing.T) {
	tally := &memTally{}
	m := Module{
		cfg: config{RejectRevenue: true},
		validate: func(string) ([]finding, error) {
			return []finding{{ID: "VAST-2.0-inline-impression"}, {ID: "VAST-2.0-some-other"}}, nil
		},
		record: tally,
	}
	payload := hookstage.RawBidderResponsePayload{
		Bidder: "dsp",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{
			videoBid(missingImpression),
			{BidType: openrtb_ext.BidTypeBanner, Bid: &openrtb2.Bid{ID: "banner", AdM: "<vast></vast>"}},
		}},
	}

	result, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	require.Equal(t, []string{"dsp|VAST-2.0-inline-impression|true"}, tally.got)

	updated := applyBids(t, payload, result)
	require.Len(t, updated.BidderResponse.Bids, 1)
	require.Equal(t, "banner", updated.BidderResponse.Bids[0].Bid.ID)
	require.Contains(t, result.DebugMessages[0], "VAST-2.0-inline-impression")
}

func TestCountingKeepsBid(t *testing.T) {
	tally := &memTally{}
	m := Module{
		cfg: config{RejectRevenue: false},
		validate: func(string) ([]finding, error) {
			return []finding{{ID: "VAST-2.0-inline-impression"}}, nil
		},
		record: tally,
	}
	payload := hookstage.RawBidderResponsePayload{
		Bidder:         "dsp",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid(missingImpression)}},
	}

	result, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	require.Empty(t, result.ChangeSet.Mutations())
	require.Equal(t, []string{"dsp|VAST-2.0-inline-impression|true"}, tally.got)
}

func TestSkipsNonVASTAndValidatorErrors(t *testing.T) {
	tally := &memTally{}
	m := Module{
		validate: func(string) ([]finding, error) {
			return nil, errors.New("vastlint down")
		},
		record: tally,
	}
	payload := hookstage.RawBidderResponsePayload{
		Bidder: "dsp",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{
			{BidType: openrtb_ext.BidTypeVideo, Bid: &openrtb2.Bid{ID: "url", AdM: "https://vast.example/tag.xml"}},
			videoBid(missingImpression),
		}},
	}

	result, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	require.Empty(t, result.ChangeSet.Mutations())
	require.Equal(t, []string{"vastlint down"}, result.Errors)
	require.Empty(t, tally.got)
}

func TestAccountConfigOverridesReject(t *testing.T) {
	tally := &memTally{}
	m := Module{
		cfg: config{RejectRevenue: false},
		validate: func(string) ([]finding, error) {
			return []finding{{ID: "VAST-2.0-wrapper-vastadtaguri"}}, nil
		},
		record: tally,
	}
	payload := hookstage.RawBidderResponsePayload{
		Bidder:         "seat",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid("<VAST></VAST>")}},
	}

	result, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{
		AccountConfig: []byte(`{"reject_revenue":true}`),
	}, payload)
	require.NoError(t, err)
	require.Len(t, result.ChangeSet.Mutations(), 1)
	require.Empty(t, applyBids(t, payload, result).BidderResponse.Bids)
}

func TestRegisterRecordsOnPrometheus(t *testing.T) {
	reg := prometheus.NewRegistry()
	Register(reg, "", "")
	t.Cleanup(resetRecorder)

	m := Module{
		validate: func(string) ([]finding, error) {
			return []finding{{ID: "VAST-2.0-inline-impression"}}, nil
		},
	}
	payload := hookstage.RawBidderResponsePayload{
		Bidder:         "dsp",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid(missingImpression)}},
	}
	_, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)

	families, err := reg.Gather()
	require.NoError(t, err)
	require.Equal(t, 1.0, counterValue(t, families, "vastlint_findings_total", map[string]string{
		"caller":         "dsp",
		"rule_id":        "VAST-2.0-inline-impression",
		"revenue_impact": "true",
	}))
}

func TestVideoStormFixtureShapeIsVAST(t *testing.T) {
	body, err := os.ReadFile("testdata/videostorm_simid_4.2.xml")
	require.NoError(t, err)
	require.Contains(t, string(body), `version="4.2"`)
	require.Contains(t, string(body), `apiFramework="SIMID"`)
	require.True(t, looksLikeVAST(string(body)))
	require.NotContains(t, string(body), "videostorm.com")
	require.NotContains(t, string(body), "linkstorm.net")
}

func videoBid(adm string) *adapters.TypedBid {
	return &adapters.TypedBid{
		BidType: openrtb_ext.BidTypeVideo,
		Bid:     &openrtb2.Bid{ID: "bid-1", ImpID: "imp-1", AdM: adm},
	}
}

func applyBids(t *testing.T, payload hookstage.RawBidderResponsePayload, result hookstage.HookResult[hookstage.RawBidderResponsePayload]) hookstage.RawBidderResponsePayload {
	t.Helper()
	for _, mutation := range result.ChangeSet.Mutations() {
		next, err := mutation.Apply(payload)
		require.NoError(t, err)
		payload = next
	}
	return payload
}

type memTally struct {
	got []string
}

func (m *memTally) Finding(caller, ruleID string, revenue bool) {
	flag := "false"
	if revenue {
		flag = "true"
	}
	m.got = append(m.got, caller+"|"+ruleID+"|"+flag)
}

func counterValue(t *testing.T, families []*dto.MetricFamily, name string, labels map[string]string) float64 {
	t.Helper()
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			if labelsMatch(metric, labels) {
				return metric.GetCounter().GetValue()
			}
		}
	}
	t.Fatalf("metric %s with labels %v not found", name, labels)
	return 0
}

func labelsMatch(metric *dto.Metric, want map[string]string) bool {
	if len(metric.Label) != len(want) {
		return false
	}
	for _, label := range metric.Label {
		if want[label.GetName()] != label.GetValue() {
			return false
		}
	}
	return true
}
