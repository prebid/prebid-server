package exchange

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/config"
	"github.com/stretchr/testify/assert"
)

func TestGetBidderTmax(t *testing.T) {
	var (
		requestTmaxMS               int64 = 700
		bidderNetworkLatencyBuffer  uint  = 50
		responsePreparationDuration uint  = 60
	)
	requestTmaxNS := requestTmaxMS * int64(time.Millisecond)
	startTime := time.Date(2023, 5, 30, 1, 0, 0, 0, time.UTC)
	deadline := time.Date(2023, 5, 30, 1, 0, 0, int(requestTmaxNS), time.UTC)
	ctx := &mockBidderTmaxCtx{startTime: startTime, deadline: deadline, ok: true}
	tests := []struct {
		description     string
		ctx             bidderTmaxContext
		requestTmax     int64
		expectedTmax    int64
		tmaxAdjustments TmaxAdjustmentsPreprocessed
	}{
		{
			description:     "returns-requestTmax-when-IsEnforced-is-false",
			ctx:             ctx,
			requestTmax:     requestTmaxMS,
			tmaxAdjustments: TmaxAdjustmentsPreprocessed{IsEnforced: false},
			expectedTmax:    requestTmaxMS,
		},
		{
			description:     "returns-requestTmax-when-BidderResponseDurationMin-is-not-set",
			ctx:             ctx,
			requestTmax:     requestTmaxMS,
			tmaxAdjustments: TmaxAdjustmentsPreprocessed{IsEnforced: true, BidderResponseDurationMin: 0},
			expectedTmax:    requestTmaxMS,
		},
		{
			description:     "returns-requestTmax-when-BidderNetworkLatencyBuffer-and-PBSResponsePreparationDuration-is-not-set",
			ctx:             ctx,
			requestTmax:     requestTmaxMS,
			tmaxAdjustments: TmaxAdjustmentsPreprocessed{IsEnforced: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 0, PBSResponsePreparationDuration: 0},
			expectedTmax:    requestTmaxMS,
		},
		{
			description:     "returns-requestTmax-when-context-deadline-is-not-set",
			ctx:             &mockBidderTmaxCtx{ok: false},
			requestTmax:     requestTmaxMS,
			tmaxAdjustments: TmaxAdjustmentsPreprocessed{IsEnforced: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 50, PBSResponsePreparationDuration: 60},
			expectedTmax:    requestTmaxMS,
		},
		{
			description:     "returns-remaing-duration-by-subtracting-BidderNetworkLatencyBuffer-and-PBSResponsePreparationDuration",
			ctx:             ctx,
			requestTmax:     requestTmaxMS,
			tmaxAdjustments: TmaxAdjustmentsPreprocessed{IsEnforced: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: bidderNetworkLatencyBuffer, PBSResponsePreparationDuration: responsePreparationDuration},
			expectedTmax:    ctx.RemainingDurationMS(deadline) - int64(bidderNetworkLatencyBuffer) - int64(responsePreparationDuration),
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			assert.Equal(t, test.expectedTmax, getBidderTmax(test.ctx, test.requestTmax, test.tmaxAdjustments))
		})
	}
}

func TestProcessTMaxAdjustments(t *testing.T) {
	tests := []struct {
		description     string
		expected        *TmaxAdjustmentsPreprocessed
		tmaxAdjustments config.TmaxAdjustments
	}{
		{
			description:     "returns-nil-when-tmax-is-not-enabled",
			tmaxAdjustments: config.TmaxAdjustments{Enabled: false},
			expected:        nil,
		},
		{
			description:     "BidderResponseDurationMin-is-not-set",
			tmaxAdjustments: config.TmaxAdjustments{Enabled: true, BidderResponseDurationMin: 0, BidderNetworkLatencyBuffer: 10, PBSResponsePreparationDuration: 20},
			expected:        &TmaxAdjustmentsPreprocessed{IsEnforced: false, BidderResponseDurationMin: 0, BidderNetworkLatencyBuffer: 10, PBSResponsePreparationDuration: 20},
		},
		{
			description:     "BidderNetworkLatencyBuffer-and-PBSResponsePreparationDuration-are-not-set",
			tmaxAdjustments: config.TmaxAdjustments{Enabled: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 0, PBSResponsePreparationDuration: 0},
			expected:        &TmaxAdjustmentsPreprocessed{IsEnforced: false, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 0, PBSResponsePreparationDuration: 0},
		},
		{
			description:     "BidderNetworkLatencyBuffer-is-not-set",
			tmaxAdjustments: config.TmaxAdjustments{Enabled: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 0, PBSResponsePreparationDuration: 10},
			expected:        &TmaxAdjustmentsPreprocessed{IsEnforced: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 0, PBSResponsePreparationDuration: 10},
		},
		{
			description:     "PBSResponsePreparationDuration-is-not-set",
			tmaxAdjustments: config.TmaxAdjustments{Enabled: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 10, PBSResponsePreparationDuration: 0},
			expected:        &TmaxAdjustmentsPreprocessed{IsEnforced: true, BidderResponseDurationMin: 100, BidderNetworkLatencyBuffer: 10, PBSResponsePreparationDuration: 0},
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			assert.Equal(t, test.expected, ProcessTMaxAdjustments(test.tmaxAdjustments))
		})
	}
}
