//go:build cgo

package vastlint

import (
	"context"
	"os"
	"testing"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/stretchr/testify/require"
)

func TestBuilderValidatesVideoStormFixture(t *testing.T) {
	body, err := os.ReadFile("testdata/videostorm_simid_4.2.xml")
	require.NoError(t, err)

	built, err := Builder([]byte(`{"enabled":true}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	m := built.(Module)
	tally := &memTally{}
	m.record = tally

	payload := hookstage.RawBidderResponsePayload{
		Bidder:         "videostorm",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid(string(body))}},
	}
	result, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	require.Empty(t, result.Errors)
	require.Empty(t, result.ChangeSet.Mutations())
	for _, got := range tally.got {
		require.Contains(t, got, "|true")
	}
}

func TestBuilderRejectsMissingImpression(t *testing.T) {
	built, err := Builder([]byte(`{"enabled":true,"reject_revenue":true}`), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	m := built.(Module)
	tally := &memTally{}
	m.record = tally

	payload := hookstage.RawBidderResponsePayload{
		Bidder:         "dsp",
		BidderResponse: &adapters.BidderResponse{Bids: []*adapters.TypedBid{videoBid(missingImpression)}},
	}
	result, err := m.HandleRawBidderResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, payload)
	require.NoError(t, err)
	require.Contains(t, tally.got, "dsp|VAST-2.0-inline-impression|true")
	require.Empty(t, applyBids(t, payload, result).BidderResponse.Bids)
}
