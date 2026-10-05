package tracing

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

const (
	tracedPartner   = "traced-partner"
	untracedPartner = "untraced-partner"
	testBidderName  = "appnexus"
	otherEndpoint   = "/openrtb2/amp"
	auctionEndpoint = hookexecution.EndpointAuction
)

func TestBuilder(t *testing.T) {
	// Act
	result, err := Builder(nil, moduledeps.ModuleDeps{})

	// Assert
	assert.NoError(t, err)
	module, ok := result.(Module)
	assert.True(t, ok)
	assert.NotNil(t, module.tracer)
	assert.NotNil(t, module.output)
	assert.NotNil(t, module.now)
	assert.NotNil(t, module.mu)
}

func newTestModule(buf *bytes.Buffer, now time.Time) Module {
	return Module{
		tracer: NewTracer([]Rule{
			{PartnerID: tracedPartner, Duration: time.Hour, TracePacketsAmount: 100},
		}),
		output: buf,
		now:    func() time.Time { return now },
		mu:     &sync.Mutex{},
	}
}

func TestHandleProcessedAuctionHook(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bidRequest := &openrtb2.BidRequest{ID: "req-1"}

	testCases := []struct {
		description  string
		accountID    string
		endpoint     string
		payload      hookstage.ProcessedAuctionRequestPayload
		expectOutput bool
	}{
		{
			description:  "traced partner on the auction endpoint emits a packet",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: bidRequest}},
			expectOutput: true,
		},
		{
			description:  "untraced partner emits nothing",
			accountID:    untracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: bidRequest}},
			expectOutput: false,
		},
		{
			description:  "traced partner on a different endpoint emits nothing",
			accountID:    tracedPartner,
			endpoint:     otherEndpoint,
			payload:      hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: bidRequest}},
			expectOutput: false,
		},
		{
			description:  "nil request wrapper emits nothing",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.ProcessedAuctionRequestPayload{Request: nil},
			expectOutput: false,
		},
		{
			description:  "nil bid request emits nothing",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.ProcessedAuctionRequestPayload{Request: &openrtb_ext.RequestWrapper{}},
			expectOutput: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			// Arrange
			var buf bytes.Buffer
			module := newTestModule(&buf, now)
			miCtx := hookstage.ModuleInvocationContext{AccountID: test.accountID, Endpoint: test.endpoint}

			// Act
			result, err := module.HandleProcessedAuctionHook(context.Background(), miCtx, test.payload)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}, result)
			if test.expectOutput {
				assert.Contains(t, buf.String(), `"type":"incoming_bid_request"`)
				assert.Contains(t, buf.String(), `"partner_id":"`+test.accountID+`"`)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestHandleBidderRequestHook(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bidRequest := &openrtb2.BidRequest{ID: "req-1"}

	testCases := []struct {
		description  string
		accountID    string
		endpoint     string
		payload      hookstage.BidderRequestPayload
		expectOutput bool
	}{
		{
			description:  "traced partner on the auction endpoint emits a packet with bidder name",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.BidderRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: bidRequest}, Bidder: testBidderName},
			expectOutput: true,
		},
		{
			description:  "untraced partner emits nothing",
			accountID:    untracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.BidderRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: bidRequest}, Bidder: testBidderName},
			expectOutput: false,
		},
		{
			description:  "traced partner on a different endpoint emits nothing",
			accountID:    tracedPartner,
			endpoint:     otherEndpoint,
			payload:      hookstage.BidderRequestPayload{Request: &openrtb_ext.RequestWrapper{BidRequest: bidRequest}, Bidder: testBidderName},
			expectOutput: false,
		},
		{
			description:  "nil bid request emits nothing",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.BidderRequestPayload{Request: &openrtb_ext.RequestWrapper{}, Bidder: testBidderName},
			expectOutput: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			// Arrange
			var buf bytes.Buffer
			module := newTestModule(&buf, now)
			miCtx := hookstage.ModuleInvocationContext{AccountID: test.accountID, Endpoint: test.endpoint}

			// Act
			result, err := module.HandleBidderRequestHook(context.Background(), miCtx, test.payload)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, hookstage.HookResult[hookstage.BidderRequestPayload]{}, result)
			if test.expectOutput {
				assert.Contains(t, buf.String(), `"type":"outgoing_bid_request"`)
				assert.Contains(t, buf.String(), `"bidder":"`+testBidderName+`"`)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestHandleRawBidderResponseHook(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bidderResponse := &adapters.BidderResponse{Currency: "USD"}

	testCases := []struct {
		description  string
		accountID    string
		endpoint     string
		payload      hookstage.RawBidderResponsePayload
		expectOutput bool
	}{
		{
			description:  "traced partner on the auction endpoint emits a packet with bidder name",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.RawBidderResponsePayload{BidderResponse: bidderResponse, Bidder: testBidderName},
			expectOutput: true,
		},
		{
			description:  "untraced partner emits nothing",
			accountID:    untracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.RawBidderResponsePayload{BidderResponse: bidderResponse, Bidder: testBidderName},
			expectOutput: false,
		},
		{
			description:  "traced partner on a different endpoint emits nothing",
			accountID:    tracedPartner,
			endpoint:     otherEndpoint,
			payload:      hookstage.RawBidderResponsePayload{BidderResponse: bidderResponse, Bidder: testBidderName},
			expectOutput: false,
		},
		{
			description:  "nil bidder response emits nothing",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.RawBidderResponsePayload{BidderResponse: nil, Bidder: testBidderName},
			expectOutput: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			// Arrange
			var buf bytes.Buffer
			module := newTestModule(&buf, now)
			miCtx := hookstage.ModuleInvocationContext{AccountID: test.accountID, Endpoint: test.endpoint}

			// Act
			result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, test.payload)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, hookstage.HookResult[hookstage.RawBidderResponsePayload]{}, result)
			if test.expectOutput {
				assert.Contains(t, buf.String(), `"type":"incoming_bid_response"`)
				assert.Contains(t, buf.String(), `"bidder":"`+testBidderName+`"`)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestHandleAuctionResponseHook(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	bidResponse := &openrtb2.BidResponse{ID: "resp-1"}

	testCases := []struct {
		description  string
		accountID    string
		endpoint     string
		payload      hookstage.AuctionResponsePayload
		expectOutput bool
	}{
		{
			description:  "traced partner on the auction endpoint emits a packet",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.AuctionResponsePayload{BidResponse: bidResponse},
			expectOutput: true,
		},
		{
			description:  "untraced partner emits nothing",
			accountID:    untracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.AuctionResponsePayload{BidResponse: bidResponse},
			expectOutput: false,
		},
		{
			description:  "traced partner on a different endpoint emits nothing",
			accountID:    tracedPartner,
			endpoint:     otherEndpoint,
			payload:      hookstage.AuctionResponsePayload{BidResponse: bidResponse},
			expectOutput: false,
		},
		{
			description:  "nil bid response emits nothing",
			accountID:    tracedPartner,
			endpoint:     auctionEndpoint,
			payload:      hookstage.AuctionResponsePayload{BidResponse: nil},
			expectOutput: false,
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			// Arrange
			var buf bytes.Buffer
			module := newTestModule(&buf, now)
			miCtx := hookstage.ModuleInvocationContext{AccountID: test.accountID, Endpoint: test.endpoint}

			// Act
			result, err := module.HandleAuctionResponseHook(context.Background(), miCtx, test.payload)

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, hookstage.HookResult[hookstage.AuctionResponsePayload]{}, result)
			if test.expectOutput {
				assert.Contains(t, buf.String(), `"type":"auction_response"`)
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}

func TestModuleTraceStopsAfterAmountLimitReached(t *testing.T) {
	// Arrange
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	module := Module{
		tracer: NewTracer([]Rule{{PartnerID: tracedPartner, Duration: time.Hour, TracePacketsAmount: 1}}),
		output: &buf,
		now:    func() time.Time { return now },
		mu:     &sync.Mutex{},
	}

	// Act
	module.trace(tracedPartner, PacketTypeAuctionResponse, "", map[string]string{"a": "b"})
	firstOutput := buf.String()
	buf.Reset()
	module.trace(tracedPartner, PacketTypeAuctionResponse, "", map[string]string{"a": "b"})

	// Assert
	assert.NotEmpty(t, firstOutput)
	assert.Empty(t, buf.String())
}

func TestModuleTraceHandlesMarshalError(t *testing.T) {
	// Arrange
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	module := newTestModule(&buf, now)

	// Act
	module.trace(tracedPartner, PacketTypeAuctionResponse, "", unmarshalable{})

	// Assert
	assert.Empty(t, buf.String())
}

func TestModuleTraceHandlesWriteError(t *testing.T) {
	// Arrange
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	module := Module{
		tracer: NewTracer([]Rule{{PartnerID: tracedPartner, Duration: time.Hour, TracePacketsAmount: 100}}),
		output: failingWriter{},
		now:    func() time.Time { return now },
		mu:     &sync.Mutex{},
	}

	// Act & Assert (must not panic)
	assert.NotPanics(t, func() {
		module.trace(tracedPartner, PacketTypeAuctionResponse, "", map[string]string{"a": "b"})
	})
}

// TestModuleTraceConcurrentWritesDoNotInterleave guards against the bidder_request and
// raw_bidder_response stages (invoked once per bidder, concurrently, by the exchange) tearing
// each other's JSON lines apart when writing to the shared output.
func TestModuleTraceConcurrentWritesDoNotInterleave(t *testing.T) {
	// Arrange
	const workers = 100
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	module := Module{
		tracer: NewTracer([]Rule{{PartnerID: tracedPartner, Duration: time.Hour, TracePacketsAmount: workers}}),
		output: &buf,
		now:    func() time.Time { return now },
		mu:     &sync.Mutex{},
	}

	// large-ish payload to make interleaving likely if writes aren't serialized
	largeValue := map[string]string{"padding": strings.Repeat("x", 8192)}

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(bidder string) {
			defer wg.Done()
			module.trace(tracedPartner, PacketTypeOutgoingBidRequest, bidder, largeValue)
		}(testBidderName)
	}
	wg.Wait()

	// Assert: every line is valid, independently parseable JSON
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	assert.Len(t, lines, workers)
	for _, line := range lines {
		var packet Packet
		assert.NoError(t, json.Unmarshal([]byte(line), &packet))
	}
}
