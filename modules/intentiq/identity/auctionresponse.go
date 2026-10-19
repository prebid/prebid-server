package identity

import (
	"context"
	"time"

	iiqreporting "github.com/Intent-IQ/identity-go/reporting"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

func (m *Module) HandleAuctionResponseHook(
	_ context.Context,
	invocation hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (result hookstage.HookResult[hookstage.AuctionResponsePayload], _ error) {
	start := time.Now()
	config := m.config.resolve(invocation.AccountConfig)
	auction, ok := loadAuctionContext(invocation.ModuleContext)

	tr := &flowTrace{collect: ok && auction.TraceEnabled}
	defer func() {
		result.DebugMessages = tr.msgs
		if !tr.enabled() {
			return
		}
		combined := make([]string, 0, len(auction.Trace)+len(tr.msgs))
		combined = append(combined, auction.Trace...)
		combined = append(combined, tr.msgs...)
		result.ChangeSet.AddMutation(
			setIdentityTraceMutation(combined),
			hookstage.MutationUpdate, "ext", "trace", traceExtKey,
		)
	}()

	if tr.enabled() {
		tr.tracef("[response] start — flow latency (enrich → response) %s", fmtDur(time.Since(auction.Start)))
	}

	if config.ReportsEndpoint == "" {
		if tr.enabled() {
			tr.tracef("[response] no reports_endpoint configured — reporting skipped; response hook took %s", fmtDur(time.Since(start)))
		}
		return result, nil
	}
	if payload.BidResponse == nil {
		if tr.enabled() {
			tr.tracef("[response] nil bid response — reporting skipped; response hook took %s", fmtDur(time.Since(start)))
		}
		return result, nil
	}

	accepted, rejected := 0, 0
	for seatIndex := range payload.BidResponse.SeatBid {
		seatBid := payload.BidResponse.SeatBid[seatIndex]
		for bidIndex := range seatBid.Bid {
			request := iiqreporting.Request{
				PartnerID: config.PartnerID, Endpoint: config.ReportsEndpoint, Timeout: config.timeout(),
				Bid: seatBid.Bid[bidIndex], BidderCode: seatBid.Seat, Currency: payload.BidResponse.Cur,
				AuctionID: auction.AuctionID, Reference: auction.Reference, IP: auction.IP,
				UserAgent: auction.UserAgent, ABTestUUID: auction.ABTestUUID,
				TerminationCause: auction.TerminationCause,
			}
			if m.reporter == nil {
				continue
			}
			if m.reporter.TryReport(request) == iiqreporting.ReportAccepted {
				accepted++
			} else {
				rejected++
			}
		}
	}
	if tr.enabled() {
		tr.tracef("[response] queued %d impression report(s) → %s (fire-and-forget), %d rejected; tc=%s carried from enrich",
			accepted, config.ReportsEndpoint, rejected, tcStr(auction.TerminationCause))
	}
	if tr.enabled() {
		tr.tracef("[response] response hook took %s", fmtDur(time.Since(start)))
	}
	return result, nil
}
