package analytics

import (
	"encoding/json"
	"time"
)

// LifecycleModule is an optional interface implemented by analytics modules that record auction lifecycles.
type LifecycleModule interface {
	// AuctionLifecycleOptions is called once near the start of each auction.
	// Returning nil means "do not record this auction".
	AuctionLifecycleOptions(passthrough json.RawMessage) *LifecycleOptions
}

// LifecycleOptions configures lifecycle capture for one module on one auction.
type LifecycleOptions struct {
	// RequestBodyBidders lists bidder codes whose outgoing request bodies should be kept.
	RequestBodyBidders []string
}

// LifecycleRunner is optionally implemented by a Runner whose modules can record auction lifecycles.
type LifecycleRunner interface {
	// BeginAuctionLifecycle returns nil unless at least one module records this auction.
	// passthrough is the request's ext.prebid.passthrough and must not be modified.
	BeginAuctionLifecycle(passthrough json.RawMessage) *AuctionLifecycle
}

// AuctionLifecycle is filled by Prebid Server during one recorded auction. It is never added to any
// response and is delivered only to modules that returned LifecycleOptions for the auction, via
// AuctionObject.Lifecycle in their existing LogAuctionObject call.
type AuctionLifecycle struct {
	// FilteredBidderImpDetails lists bidder x impression pairs that were removed before being placed in a bidder request.
	FilteredBidderImpDetails []FilteredBidderImpDetail
	// BidderRequestDetails lists every bidder request that was built, whether or not it was sent.
	BidderRequestDetails []BidderRequestDetail
	// Bids lists every bid decoded from a bidder response and whether core later filtered it.
	Bids []BidResult
	// RequestBodiesWithheld is true when privacy rules removed BidderRequestDetails[].RequestBody before delivery.
	RequestBodiesWithheld bool

	requestBodyBidders map[string]struct{}
	subscriptions      map[string]map[string]struct{}
}

// FilteredBidderImpDetail is a bidder x impression pair removed before it was placed in a bidder request.
type FilteredBidderImpDetail struct {
	Bidder string
	ImpID  string
	// Scope is "bidder" (whole bidder dropped) or "impression" (only this impression dropped).
	Scope string
	// ReasonCode is an IAB seat-non-bid status code, used here only as a reason vocabulary.
	ReasonCode int
	// ReasonDetail narrows ReasonCode, e.g. "gdpr" or a rules engine analyticsvalue.
	ReasonDetail string
}

// Scopes for FilteredBidderImpDetail.Scope.
const (
	FilterScopeBidder     = "bidder"
	FilterScopeImpression = "impression"
)

// IAB seat-non-bid status codes used for FilteredBidderImpDetail.ReasonCode, BidderRequestDetail.ErrorCode, and BidResult.ReasonCode.
const (
	ErrorGeneral                       = 100
	ErrorTimeout                       = 101
	RequestBlockedGeneral              = 200
	RequestBlockedUnsupportedChannel   = 201
	RequestBlockedUnsupportedMediaType = 202
	RequestBlockedOptimized            = 203
	RequestBlockedPrivacy              = 204
	ResponseRejectedGeneral            = 300
	ResponseRejectedBelowFloor         = 301
	ResponseRejectedCategoryMapping    = 303
	ResponseRejectedBelowDealFloor     = 304
	ResponseRejectedSizeNotAllowed     = 351
	ResponseRejectedNotSecure          = 352
)

// Reasons a built bidder request was not sent over HTTP.
const (
	NotSentThrottled        = "throttled"
	NotSentTmaxInsufficient = "tmaxInsufficient"
	NotSentRequestError     = "requestError"
	NotSentStoredResponse   = "storedResponse"
)

// BidderRequestDetail is one request built by an adapter, whether sent or not.
type BidderRequestDetail struct {
	// BidderRequestID is unique within the auction.
	BidderRequestID string
	Bidder          string
	// OpenRTBVersion is the OpenRTB version the bidder request was built for (e.g. "2.5" or "2.6").
	OpenRTBVersion string
	// TMaxMs is the timeout in milliseconds on the bidder request after any PBS tmax adjustment.
	TMaxMs int64
	// HTTPStatus is zero when no response was received.
	HTTPStatus int
	// Duration covers sending the request and reading the response.
	Duration time.Duration
	// ErrorCode is 0 when no error occurred.
	ErrorCode int
	// ErrorReason describes why the request was not sent or why it failed.
	ErrorReason string
	// NotSentReason is empty when the request was sent over HTTP.
	NotSentReason string
	// Err is the transport or HTTP status error, if any.
	Err error
	// ResponseErrs are the errors the adapter reported while decoding the response.
	ResponseErrs []error
	// RequestBody is set only for bidders requested in LifecycleOptions.RequestBodyBidders.
	// It is shared with Prebid Server and must not be modified.
	RequestBody []byte
	ImpDetails  []ImpDetail
}

// ImpDetail is one impression carried by a bidder request.
type ImpDetail struct {
	ImpID string
	// MediaTypes are the media types the impression carried when sent, e.g. "banner".
	MediaTypes []string
	// RemovedMediaTypes are media types removed from this impression while other media types remained.
	RemovedMediaTypes []string
	// BidIDs are the IDs of the bids decoded for this impression from this request's response.
	BidIDs []string
}

// BidResult is one bid decoded from a bidder response.
type BidResult struct {
	BidderRequestID string
	BidID           string
	ImpID           string
	IsFiltered      bool
	ReasonCode      int
	ReasonDetail    string
}

// NewAuctionLifecycle returns nil when no module records the auction.
func NewAuctionLifecycle(optionsByModule map[string]*LifecycleOptions) *AuctionLifecycle {
	if len(optionsByModule) == 0 {
		return nil
	}
	lifecycle := &AuctionLifecycle{
		subscriptions: make(map[string]map[string]struct{}, len(optionsByModule)),
	}
	for module, opts := range optionsByModule {
		if opts == nil {
			continue
		}
		var moduleBidders map[string]struct{}
		if len(opts.RequestBodyBidders) > 0 {
			moduleBidders = make(map[string]struct{}, len(opts.RequestBodyBidders))
			if lifecycle.requestBodyBidders == nil {
				lifecycle.requestBodyBidders = make(map[string]struct{}, len(opts.RequestBodyBidders))
			}
			for _, bidder := range opts.RequestBodyBidders {
				moduleBidders[bidder] = struct{}{}
				lifecycle.requestBodyBidders[bidder] = struct{}{}
			}
		}
		lifecycle.subscriptions[module] = moduleBidders
	}
	if len(lifecycle.subscriptions) == 0 {
		return nil
	}
	return lifecycle
}

// KeepsRequestBody reports whether any recording module asked for bidder's outgoing request bodies.
func (l *AuctionLifecycle) KeepsRequestBody(bidder string) bool {
	if l == nil {
		return false
	}
	_, ok := l.requestBodyBidders[bidder]
	return ok
}

// RecordFiltered records one filtered bidder x impression pair. It is a no-op on a nil lifecycle.
func (l *AuctionLifecycle) RecordFiltered(bidder, impID string, reasonCode int, reasonDetail string) {
	l.RecordFilteredWithScope(bidder, impID, defaultFilterScope(reasonCode), reasonCode, reasonDetail)
}

// AddFiltered is an alias for RecordFiltered.
func (l *AuctionLifecycle) AddFiltered(bidder, impID string, reasonCode int, reasonDetail string) {
	l.RecordFiltered(bidder, impID, reasonCode, reasonDetail)
}

// RecordFilteredWithScope records one filtered bidder x impression pair with an explicit scope ("bidder" or "impression").
func (l *AuctionLifecycle) RecordFilteredWithScope(bidder, impID, scope string, reasonCode int, reasonDetail string) {
	if l == nil {
		return
	}
	l.FilteredBidderImpDetails = append(l.FilteredBidderImpDetails, FilteredBidderImpDetail{
		Bidder:       bidder,
		ImpID:        impID,
		Scope:        scope,
		ReasonCode:   reasonCode,
		ReasonDetail: reasonDetail,
	})
}

func defaultFilterScope(code int) string {
	switch code {
	case ErrorGeneral, RequestBlockedUnsupportedMediaType:
		return FilterScopeImpression
	default:
		return FilterScopeBidder
	}
}

// Merge appends other's entries to l. It is not safe for concurrent use.
func (l *AuctionLifecycle) Merge(other *AuctionLifecycle) {
	if l == nil || other == nil {
		return
	}
	l.FilteredBidderImpDetails = append(l.FilteredBidderImpDetails, other.FilteredBidderImpDetails...)
	l.BidderRequestDetails = append(l.BidderRequestDetails, other.BidderRequestDetails...)
	l.Bids = append(l.Bids, other.Bids...)
}

// ForModule returns the lifecycle view for module, or nil when module did not opt in to recording this auction.
// When withholdRequestBodies is true, all request bodies are removed and RequestBodiesWithheld is set.
// Otherwise, only request bodies for bidders selected by module are kept.
func (l *AuctionLifecycle) ForModule(module string, withholdRequestBodies bool) *AuctionLifecycle {
	if l == nil {
		return nil
	}
	moduleBidders, ok := l.subscriptions[module]
	if !ok {
		return nil
	}
	if withholdRequestBodies {
		return l.withBodySelection(nil, true)
	}
	if len(moduleBidders) == len(l.requestBodyBidders) {
		return l
	}
	return l.withBodySelection(moduleBidders, false)
}

func (l *AuctionLifecycle) withBodySelection(allowedBidders map[string]struct{}, withheld bool) *AuctionLifecycle {
	copied := *l
	if len(l.BidderRequestDetails) > 0 {
		copied.BidderRequestDetails = make([]BidderRequestDetail, len(l.BidderRequestDetails))
		for i, call := range l.BidderRequestDetails {
			if _, ok := allowedBidders[call.Bidder]; !ok {
				call.RequestBody = nil
			}
			copied.BidderRequestDetails[i] = call
		}
	}
	copied.RequestBodiesWithheld = withheld
	return &copied
}
