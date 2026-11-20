package tracing

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
)

// Builder builds the tracing module. Tracing rules are hardcoded (see rules.go) per the
// task requirement; the config argument is intentionally unused.
func Builder(_ json.RawMessage, _ moduledeps.ModuleDeps) (interface{}, error) {
	return Module{
		tracer: NewTracer(Rules),
		output: os.Stdout,
		now:    time.Now,
		mu:     &sync.Mutex{},
	}, nil
}

// Module traces auction lifecycle data (incoming/outgoing bid requests and responses) for
// accounts matching a hardcoded set of rules, printing each collected trace packet as JSON
// directly to stdout. It only acts on the /openrtb2/auction endpoint.
//
// The bidder_request and raw_bidder_response stages run once per bidder, concurrently, so
// writes to output are serialized through mu to keep each printed JSON line intact.
type Module struct {
	tracer *Tracer
	output io.Writer
	now    func() time.Time
	mu     *sync.Mutex
}

// HandleProcessedAuctionHook traces the incoming BidRequest.
func (m Module) HandleProcessedAuctionHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{}
	if !inScope(miCtx) || payload.Request == nil || payload.Request.BidRequest == nil {
		return result, nil
	}

	m.trace(miCtx.AccountID, PacketTypeIncomingBidRequest, "", payload.Request.BidRequest)
	return result, nil
}

// HandleBidderRequestHook traces the outgoing BidRequest sent to a specific bidder.
func (m Module) HandleBidderRequestHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	result := hookstage.HookResult[hookstage.BidderRequestPayload]{}
	if !inScope(miCtx) || payload.Request == nil || payload.Request.BidRequest == nil {
		return result, nil
	}

	m.trace(miCtx.AccountID, PacketTypeOutgoingBidRequest, payload.Bidder, payload.Request.BidRequest)
	return result, nil
}

// HandleRawBidderResponseHook traces the incoming BidResponse from a specific bidder.
func (m Module) HandleRawBidderResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}
	if !inScope(miCtx) || payload.BidderResponse == nil {
		return result, nil
	}

	m.trace(miCtx.AccountID, PacketTypeIncomingBidResponse, payload.Bidder, payload.BidderResponse)
	return result, nil
}

// HandleAuctionResponseHook traces the final auction response sent back to the client.
func (m Module) HandleAuctionResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	result := hookstage.HookResult[hookstage.AuctionResponsePayload]{}
	if !inScope(miCtx) || payload.BidResponse == nil {
		return result, nil
	}

	m.trace(miCtx.AccountID, PacketTypeAuctionResponse, "", payload.BidResponse)
	return result, nil
}

// inScope reports whether the hook is executing for the /openrtb2/auction endpoint, the
// only endpoint this module is allowed to affect.
func inScope(miCtx hookstage.ModuleInvocationContext) bool {
	return miCtx.Endpoint == hookexecution.EndpointAuction
}

// trace collects and prints a single trace packet for partnerID, if the tracer's hardcoded
// rules still allow it.
func (m Module) trace(partnerID string, packetType PacketType, bidder string, value any) {
	now := m.now()
	if !m.tracer.Allow(partnerID, now) {
		return
	}

	packet, err := newPacket(partnerID, packetType, bidder, now, value)
	if err != nil {
		logger.Warnf("intentiq.tracing: failed to build trace packet: %s", err)
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if err := emit(m.output, packet); err != nil {
		logger.Warnf("intentiq.tracing: failed to write trace packet: %s", err)
	}
}
