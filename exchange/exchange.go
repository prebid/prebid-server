package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/adservertargeting"
	"github.com/prebid/prebid-server/v3/bidadjustment"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/dsa"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/experiment/adscert"
	"github.com/prebid/prebid-server/v3/firstpartydata"
	"github.com/prebid/prebid-server/v3/floors"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/hooks/hookexecution"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/stored_responses"
	"github.com/prebid/prebid-server/v3/usersync"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/maputil"

	"github.com/buger/jsonparser"
	"github.com/gofrs/uuid"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/openrtb/v20/openrtb3"
)

type extCacheInstructions struct {
	cacheBids, cacheVAST, returnCreative bool
}

// Exchange runs Auctions. Implementations must be threadsafe, and will be shared across many goroutines.
type Exchange interface {
	// HoldAuction executes an OpenRTB v2.5 Auction.
	HoldAuction(ctx context.Context, r *AuctionRequest, debugLog *DebugLog) (*AuctionResponse, error)
}

// IdFetcher can find the user's ID for a specific Bidder.
type IdFetcher interface {
	GetUID(key string) (uid string, exists bool, notExpired bool)
	HasAnyLiveSyncs() bool
}

type exchange struct {
	adapterMap               map[openrtb_ext.BidderName]AdaptedBidder
	bidderInfo               config.BidderInfos
	bidderToSyncerKey        map[string]string
	me                       metrics.MetricsEngine
	cache                    prebid_cache_client.Client
	cacheTime                time.Duration
	gdprPermsBuilder         gdpr.PermissionsBuilder
	currencyConverter        *currency.RateConverter
	externalURL              string
	gdprDefaultValue         gdpr.Signal
	privacyConfig            config.Privacy
	categoriesFetcher        stored_requests.CategoryFetcher
	bidIDGenerator           BidIDGenerator
	hostSChainNode           *openrtb2.SupplyChainNode
	adsCertSigner            adscert.Signer
	server                   config.Server
	bidValidationEnforcement config.Validations
	requestSplitter          requestSplitter
	macroReplacer            macros.Replacer
	priceFloorEnabled        bool
	priceFloorFetcher        floors.FloorFetcher
	singleFormatBidders      map[openrtb_ext.BidderName]struct{}
}

// Container to pass out response ext data from the GetAllBids goroutines back into the main thread
type seatResponseExtra struct {
	ResponseTimeMillis int
	Errors             []openrtb_ext.ExtBidderMessage
	Warnings           []openrtb_ext.ExtBidderMessage
	// httpCalls is the list of debugging info. It should only be populated if the request.test == 1.
	// This will become response.ext.debug.httpcalls.{bidder} on the final Response.
	HttpCalls []*openrtb_ext.ExtHttpCall
	// NonBid contains non bid reason information
	NonBid *openrtb_ext.NonBid
}

type bidResponseWrapper struct {
	adapterSeatBids         []*entities.PbsOrtbSeatBid
	adapterExtra            *seatResponseExtra
	bidder                  openrtb_ext.BidderName
	adapter                 openrtb_ext.BidderName
	bidderResponseStartTime time.Time
	seatNonBidBuilder       SeatNonBidBuilder
}

type BidIDGenerator interface {
	New(bidder string) (string, error)
	Enabled() bool
}

type bidIDGenerator struct {
	enabled bool
}

func (big *bidIDGenerator) Enabled() bool {
	return big.enabled
}

func (big *bidIDGenerator) New(bidder string) (string, error) {
	rawUuid, err := uuid.NewV4()
	return rawUuid.String(), err
}

type deduplicateChanceGenerator interface {
	Generate() bool
}

type randomDeduplicateBidBooleanGenerator struct{}

func (randomDeduplicateBidBooleanGenerator) Generate() bool {
	return rand.Intn(100) < 50
}

func NewExchange(adapters map[openrtb_ext.BidderName]AdaptedBidder, cache prebid_cache_client.Client, cfg *config.Configuration, requestValidator ortb.RequestValidator, syncersByBidder map[string]usersync.Syncer, metricsEngine metrics.MetricsEngine, infos config.BidderInfos, gdprPermsBuilder gdpr.PermissionsBuilder, currencyConverter *currency.RateConverter, categoriesFetcher stored_requests.CategoryFetcher, adsCertSigner adscert.Signer, macroReplacer macros.Replacer, priceFloorFetcher floors.FloorFetcher, singleFormatBidders map[openrtb_ext.BidderName]struct{}) Exchange {
	bidderToSyncerKey := map[string]string{}
	for bidder, syncer := range syncersByBidder {
		bidderToSyncerKey[bidder] = syncer.Key()
	}

	gdprDefaultValue := gdpr.SignalYes
	if cfg.GDPR.DefaultValue == "0" {
		gdprDefaultValue = gdpr.SignalNo
	}

	privacyConfig := config.Privacy{
		CCPA: cfg.CCPA,
		GDPR: cfg.GDPR,
		LMT:  cfg.LMT,
	}
	requestSplitter := requestSplitter{
		bidderToSyncerKey: bidderToSyncerKey,
		me:                metricsEngine,
		privacyConfig:     privacyConfig,
		gdprPermsBuilder:  gdprPermsBuilder,
		hostSChainNode:    cfg.HostSChainNode,
		bidderInfo:        infos,
		requestValidator:  requestValidator,
	}

	return &exchange{
		adapterMap:               adapters,
		bidderInfo:               infos,
		bidderToSyncerKey:        bidderToSyncerKey,
		cache:                    cache,
		cacheTime:                time.Duration(cfg.CacheURL.ExpectedTimeMillis) * time.Millisecond,
		categoriesFetcher:        categoriesFetcher,
		currencyConverter:        currencyConverter,
		externalURL:              cfg.ExternalURL,
		gdprPermsBuilder:         gdprPermsBuilder,
		me:                       metricsEngine,
		gdprDefaultValue:         gdprDefaultValue,
		privacyConfig:            privacyConfig,
		bidIDGenerator:           &bidIDGenerator{cfg.GenerateBidID},
		hostSChainNode:           cfg.HostSChainNode,
		adsCertSigner:            adsCertSigner,
		server:                   config.Server{ExternalUrl: cfg.ExternalURL, GvlID: cfg.GDPR.HostVendorID, DataCenter: cfg.DataCenter},
		bidValidationEnforcement: cfg.Validations,
		requestSplitter:          requestSplitter,
		macroReplacer:            macroReplacer,
		priceFloorEnabled:        cfg.PriceFloors.Enabled,
		priceFloorFetcher:        priceFloorFetcher,
		singleFormatBidders:      singleFormatBidders,
	}
}

type ImpExtInfo struct {
	EchoVideoAttrs bool
	StoredImp      []byte
	Passthrough    json.RawMessage
}

// AuctionRequest holds the bid request for the auction
// and all other information needed to process that request
type AuctionRequest struct {
	BidRequestWrapper          *openrtb_ext.RequestWrapper
	ResolvedBidRequest         json.RawMessage
	Account                    config.Account
	UserSyncs                  IdFetcher
	RequestType                metrics.RequestType
	StartTime                  time.Time
	Warnings                   []error
	GlobalPrivacyControlHeader string
	ImpExtInfoMap              map[string]ImpExtInfo
	TCF2Config                 gdpr.TCF2ConfigReader
	Activities                 privacy.ActivityControl

	// LegacyLabels is included here for temporary compatibility with cleanOpenRTBRequests
	// in HoldAuction until we get to factoring it away. Do not use for anything new.
	LegacyLabels   metrics.Labels
	FirstPartyData map[openrtb_ext.BidderName]*firstpartydata.ResolvedFirstPartyData
	// map of imp id to stored response
	StoredAuctionResponses stored_responses.ImpsWithBidResponses
	// map of imp id to bidder to stored response
	StoredBidResponses      stored_responses.ImpBidderStoredResp
	BidderImpReplaceImpID   stored_responses.BidderImpReplaceImpID
	PubID                   string
	HookExecutor            hookexecution.StageExecutor
	QueryParams             url.Values
	BidderResponseStartTime time.Time
	TmaxAdjustments         *TmaxAdjustmentsPreprocessed
}

// BidderRequest holds the bidder specific request and all other
// information needed to process that bidder request.
type BidderRequest struct {
	BidRequest            *openrtb2.BidRequest
	BidderName            openrtb_ext.BidderName
	BidderCoreName        openrtb_ext.BidderName
	BidderLabels          metrics.AdapterLabels
	BidderStoredResponses map[string]json.RawMessage
	IsRequestAlias        bool
	ImpReplaceImpId       map[string]bool
}

func (e *exchange) HoldAuction(ctx context.Context, r *AuctionRequest, debugLog *DebugLog) (*AuctionResponse, error) {
	if r == nil {
		return nil, nil
	}

	err := r.HookExecutor.ExecuteProcessedAuctionStage(r.BidRequestWrapper)
	if err != nil {
		return nil, err
	}

	requestExt, err := r.BidRequestWrapper.GetRequestExt()
	if err != nil {
		return nil, err
	}

	// ensure prebid object always exists
	requestExtPrebid := requestExt.GetPrebid()
	if requestExtPrebid == nil {
		requestExtPrebid = &openrtb_ext.ExtRequestPrebid{}
	}

	if !e.server.Empty() {
		requestExtPrebid.Server = &openrtb_ext.ExtRequestPrebidServer{
			ExternalUrl: e.server.ExternalUrl,
			GvlID:       e.server.GvlID,
			DataCenter:  e.server.DataCenter}
		requestExt.SetPrebid(requestExtPrebid)
	}

	cacheInstructions := getExtCacheInstructions(requestExtPrebid)

	targData, warning := getExtTargetData(requestExtPrebid, cacheInstructions, r.Account)
	if targData != nil {
		_, targData.cacheHost, targData.cachePath = e.cache.GetExtCacheData()
	}

	for _, w := range warning {
		r.Warnings = append(r.Warnings, w)
	}

	// Get currency rates conversions for the auction
	conversions := currency.GetAuctionCurrencyRates(e.currencyConverter, requestExtPrebid.CurrencyConversions)

	var floorErrs []error
	if e.priceFloorEnabled {
		floorErrs = floors.EnrichWithPriceFloors(r.BidRequestWrapper, r.Account, conversions, e.priceFloorFetcher)
	}

	responseDebugAllow, accountDebugAllow, debugLog := getDebugInfo(r.BidRequestWrapper.Test, requestExtPrebid, r.Account.DebugAllow, debugLog)

	// save incoming request with stored requests (if applicable) to return in debug logs
	if responseDebugAllow || len(requestExtPrebid.AdServerTargeting) > 0 {
		if err := r.BidRequestWrapper.RebuildRequest(); err != nil {
			return nil, err
		}
		resolvedBidReq, err := jsonutil.Marshal(r.BidRequestWrapper.BidRequest)
		if err != nil {
			return nil, err
		}
		r.ResolvedBidRequest = resolvedBidReq
	}
	e.me.RecordDebugRequest(responseDebugAllow || accountDebugAllow, r.PubID)

	if r.RequestType == metrics.ReqTypeORTB2Web ||
		r.RequestType == metrics.ReqTypeORTB2App ||
		r.RequestType == metrics.ReqTypeAMP {
		//Extract First party data for auction endpoint only
		resolvedFPD, fpdErrors := firstpartydata.ExtractFPDForBidders(r.BidRequestWrapper)
		if len(fpdErrors) > 0 {
			var errMessages []string
			for _, fpdError := range fpdErrors {
				errMessages = append(errMessages, fpdError.Error())
			}
			return nil, &errortypes.BadInput{
				Message: strings.Join(errMessages, ","),
			}
		}
		r.FirstPartyData = resolvedFPD
	}

	bidAdjustmentFactors := getExtBidAdjustmentFactors(requestExtPrebid)

	recordImpMetrics(r.BidRequestWrapper, e.me)

	// Retrieve EEA countries configuration from either host or account settings
	eeaCountries := selectEEACountries(e.privacyConfig.GDPR.EEACountries, r.Account.GDPR.EEACountries)

	// Make our best guess if GDPR applies
	gdprDefaultValue := e.parseGDPRDefaultValue(r.BidRequestWrapper, eeaCountries)
	gdprSignal, err := getGDPR(r.BidRequestWrapper)
	if err != nil {
		return nil, err
	}
	channelEnabled := r.TCF2Config.ChannelEnabled(channelTypeMap[r.LegacyLabels.RType])
	gdprEnforced := enforceGDPR(gdprSignal, gdprDefaultValue, channelEnabled)
	dsaWriter := dsa.Writer{
		Config:      r.Account.Privacy.DSA,
		GDPRInScope: gdprEnforced,
	}
	if err := dsaWriter.Write(r.BidRequestWrapper); err != nil {
		return nil, err
	}

	// rebuild/resync the request in the request wrapper.
	if err := r.BidRequestWrapper.RebuildRequest(); err != nil {
		return nil, err
	}

	// Slice of BidRequests, each a copy of the original cleaned to only contain bidder data for the named bidder
	requestExtLegacy := &openrtb_ext.ExtRequest{
		Prebid: *requestExtPrebid,
		SChain: requestExt.GetSChain(),
	}
	bidderRequests, privacyLabels, errs := e.requestSplitter.cleanOpenRTBRequests(ctx, *r, requestExtLegacy, gdprSignal, gdprEnforced, bidAdjustmentFactors)
	for _, err := range errs {
		if errortypes.ReadCode(err) == errortypes.InvalidImpFirstPartyDataErrorCode {
			return nil, err
		}
	}
	errs = append(errs, floorErrs...)

	mergedBidAdj, err := bidadjustment.Merge(r.BidRequestWrapper, r.Account.BidAdjustments)
	if err != nil {
		if errortypes.ContainsFatalError([]error{err}) {
			return nil, err
		}
		errs = append(errs, err)
	}
	bidAdjustmentRules := bidadjustment.BuildRules(mergedBidAdj)

	e.me.RecordRequestPrivacy(privacyLabels)

	if len(r.StoredAuctionResponses) > 0 || len(r.StoredBidResponses) > 0 {
		e.me.RecordStoredResponse(r.PubID)
	}

	// If we need to cache bids, then it will take some time to call prebid cache.
	// We should reduce the amount of time the bidders have, to compensate.
	auctionCtx, cancel := e.makeAuctionContext(ctx, cacheInstructions.cacheBids)
	defer cancel()

	var (
		adapterBids     map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		adapterExtra    map[openrtb_ext.BidderName]*seatResponseExtra
		fledge          *openrtb_ext.Fledge
		anyBidsReturned bool
		// List of bidders we have requests for.
		liveAdapters      []openrtb_ext.BidderName
		seatNonBidBuilder SeatNonBidBuilder = SeatNonBidBuilder{}
	)

	if len(r.StoredAuctionResponses) > 0 {
		adapterBids, fledge, liveAdapters, err = buildStoredAuctionResponse(r.StoredAuctionResponses)
		if err != nil {
			return nil, err
		}
		anyBidsReturned = true

	} else {
		// List of bidders we have requests for.
		liveAdapters = listBiddersWithRequests(bidderRequests)

		//This will be used to validate bids
		var alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes
		if requestExtPrebid.AlternateBidderCodes != nil {
			alternateBidderCodes = *requestExtPrebid.AlternateBidderCodes
		} else if r.Account.AlternateBidderCodes != nil {
			alternateBidderCodes = *r.Account.AlternateBidderCodes
		}

		liveAdaptersPreferredMediaType := getBidderPreferredMediaTypeMap(requestExtPrebid, &r.Account, liveAdapters, e.singleFormatBidders)

		var extraRespInfo extraAuctionResponseInfo
		adapterBids, adapterExtra, extraRespInfo = e.getAllBids(auctionCtx, bidderRequests, bidAdjustmentFactors, conversions, accountDebugAllow, r.GlobalPrivacyControlHeader, debugLog.DebugOverride, alternateBidderCodes, requestExtLegacy.Prebid.Experiment, r.HookExecutor, r.StartTime, bidAdjustmentRules, r.TmaxAdjustments, responseDebugAllow, liveAdaptersPreferredMediaType)
		fledge = extraRespInfo.fledge
		anyBidsReturned = extraRespInfo.bidsFound
		r.BidderResponseStartTime = extraRespInfo.bidderResponseStartTime
		if extraRespInfo.seatNonBidBuilder != nil {
			seatNonBidBuilder = extraRespInfo.seatNonBidBuilder
		}
	}

	var (
		auc            *auction
		cacheErrs      []error
		bidResponseExt *openrtb_ext.ExtBidResponse
	)

	if anyBidsReturned {
		if e.priceFloorEnabled {
			var rejectedBids []*entities.PbsOrtbSeatBid
			var enforceErrs []error

			adapterBids, enforceErrs, rejectedBids = floors.Enforce(r.BidRequestWrapper, adapterBids, r.Account, conversions)
			errs = append(errs, enforceErrs...)
			for _, rejectedBid := range rejectedBids {
				errs = append(errs, &errortypes.Warning{
					Message:     fmt.Sprintf("%s bid id %s rejected - bid price %.4f %s is less than bid floor %.4f %s for imp %s", rejectedBid.Seat, rejectedBid.Bids[0].Bid.ID, rejectedBid.Bids[0].Bid.Price, rejectedBid.Currency, rejectedBid.Bids[0].BidFloors.FloorValue, rejectedBid.Bids[0].BidFloors.FloorCurrency, rejectedBid.Bids[0].Bid.ImpID),
					WarningCode: errortypes.FloorBidRejectionWarningCode})
				rejectionReason := ResponseRejectedBelowFloor
				if rejectedBid.Bids[0].Bid.DealID != "" {
					rejectionReason = ResponseRejectedBelowDealFloor
				}
				seatNonBidBuilder.rejectBid(rejectedBid.Bids[0], int(rejectionReason), rejectedBid.Seat)
			}
		}

		var bidCategory map[string]string
		//If includebrandcategory is present in ext then CE feature is on.
		if requestExtPrebid.Targeting != nil && requestExtPrebid.Targeting.IncludeBrandCategory != nil {
			var rejections []string
			bidCategory, adapterBids, rejections, err = applyCategoryMapping(ctx, *requestExtPrebid.Targeting, adapterBids, e.categoriesFetcher, targData, &randomDeduplicateBidBooleanGenerator{}, &seatNonBidBuilder)
			if err != nil {
				return nil, fmt.Errorf("Error in category mapping : %s", err.Error())
			}
			for _, message := range rejections {
				errs = append(errs, errors.New(message))
			}
		}

		if e.bidIDGenerator.Enabled() {
			for bidder, seatBid := range adapterBids {
				for i := range seatBid.Bids {
					if bidID, err := e.bidIDGenerator.New(bidder.String()); err == nil {
						seatBid.Bids[i].GeneratedBidID = bidID
					} else {
						errs = append(errs, errors.New("Error generating bid.ext.prebid.bidid"))
					}
				}
			}
		}

		evTracking := getEventTracking(requestExtPrebid, r.StartTime, &r.Account, e.bidderInfo, e.externalURL)
		adapterBids = evTracking.modifyBidsForEvents(adapterBids)

		r.HookExecutor.ExecuteAllProcessedBidResponsesStage(adapterBids)

		if targData != nil {
			multiBidMap := buildMultiBidMap(requestExtPrebid)

			// A non-nil auction is only needed if targeting is active. (It is used below this block to extract cache keys)
			auc = newAuction(adapterBids, len(r.BidRequestWrapper.Imp), targData.preferDeals)
			auc.validateAndUpdateMultiBid(adapterBids, targData.preferDeals, r.Account.DefaultBidLimit)
			auc.setRoundedPrices(*targData)

			if requestExtPrebid.SupportDeals {
				dealErrs := applyDealSupport(r.BidRequestWrapper.BidRequest, auc, bidCategory, multiBidMap)
				errs = append(errs, dealErrs...)
			}

			bidResponseExt = e.makeExtBidResponse(adapterBids, adapterExtra, *r, responseDebugAllow, requestExtPrebid.Passthrough, fledge, errs)
			if debugLog.DebugEnabledOrOverridden {
				if bidRespExtBytes, err := jsonutil.Marshal(bidResponseExt); err == nil {
					debugLog.Data.Response = string(bidRespExtBytes)
				} else {
					debugLog.Data.Response = "Unable to marshal response ext for debugging"
					errs = append(errs, err)
				}
			}

			cacheErrs = auc.doCache(ctx, e.cache, targData, evTracking, r.BidRequestWrapper.BidRequest, 60, &r.Account.CacheTTL, bidCategory, debugLog)
			if len(cacheErrs) > 0 {
				errs = append(errs, cacheErrs...)
			}

			if targData.includeWinners || targData.includeBidderKeys || targData.includeFormat {
				targData.setTargeting(auc, r.BidRequestWrapper.BidRequest.App != nil, bidCategory, r.Account.TruncateTargetAttribute, multiBidMap)
			}
		}
		bidResponseExt = e.makeExtBidResponse(adapterBids, adapterExtra, *r, responseDebugAllow, requestExtPrebid.Passthrough, fledge, errs)
	} else {
		bidResponseExt = e.makeExtBidResponse(adapterBids, adapterExtra, *r, responseDebugAllow, requestExtPrebid.Passthrough, fledge, errs)

		if debugLog.DebugEnabledOrOverridden {

			if bidRespExtBytes, err := jsonutil.Marshal(bidResponseExt); err == nil {
				debugLog.Data.Response = string(bidRespExtBytes)
			} else {
				debugLog.Data.Response = "Unable to marshal response ext for debugging"
				errs = append(errs, err)
			}
		}
	}

	if !accountDebugAllow && !debugLog.DebugOverride {
		accountDebugDisabledWarning := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.AccountLevelDebugDisabledWarningCode,
			Message: "debug turned off for account",
		}
		bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral] = append(bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral], accountDebugDisabledWarning)
	}

	for _, warning := range r.Warnings {
		if errortypes.ReadScope(warning) == errortypes.ScopeDebug && !responseDebugAllow {
			continue
		}
		generalWarning := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.ReadCode(warning),
			Message: warning.Error(),
		}
		bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral] = append(bidResponseExt.Warnings[openrtb_ext.BidderReservedGeneral], generalWarning)
	}

	e.bidValidationEnforcement.SetBannerCreativeMaxSize(r.Account.Validations)

	// Build the response
	bidResponse := e.buildBidResponse(ctx, liveAdapters, adapterBids, r.BidRequestWrapper, adapterExtra, auc, bidResponseExt, cacheInstructions.returnCreative, r.ImpExtInfoMap, r.PubID, errs, &seatNonBidBuilder)
	bidResponse = adservertargeting.Apply(r.BidRequestWrapper, r.ResolvedBidRequest, bidResponse, r.QueryParams, bidResponseExt, r.Account.TruncateTargetAttribute)

	bidResponse.Ext, err = encodeBidResponseExt(bidResponseExt)
	if err != nil {
		return nil, err
	}
	bidResponseExt = setSeatNonBid(bidResponseExt, seatNonBidBuilder)

	return &AuctionResponse{
		BidResponse:    bidResponse,
		ExtBidResponse: bidResponseExt,
	}, nil
}

// getBidderPreferredMediaType reads the preferred media type from the request and account and returns a map of bidder to preferred media type. Preference given to the request over account.
func getBidderPreferredMediaTypeMap(prebid *openrtb_ext.ExtRequestPrebid, account *config.Account, liveAdapters []openrtb_ext.BidderName, singleFormatBidders map[openrtb_ext.BidderName]struct{}) openrtb_ext.PreferredMediaType {
	preferredMediaType := make(openrtb_ext.PreferredMediaType)

	// Skip if no bidders are present in singleFormatBidders
	if len(singleFormatBidders) == 0 {
		return nil
	}

	for _, bidder := range liveAdapters {

		// Skip if the bidder is not present in singleFormatBidders
		if _, found := singleFormatBidders[bidder]; !found {
			continue
		}

		//read preferred media type from request
		if prebid != nil && prebid.BidderControls != nil {
			if bidderControl, found := prebid.BidderControls[bidder]; found && bidderControl.PreferredMediaType != "" {
				preferredMediaType[bidder] = bidderControl.PreferredMediaType
				continue
			}
		}

		// if preferred media type not present in the request, read from account config
		if account != nil && account.PreferredMediaType != nil {
			if preferredType, found := account.PreferredMediaType[bidder]; found {
				preferredMediaType[bidder] = preferredType
			}
		}
	}

	return preferredMediaType
}

func buildMultiBidMap(prebid *openrtb_ext.ExtRequestPrebid) map[string]openrtb_ext.ExtMultiBid {
	if prebid == nil || prebid.MultiBid == nil {
		return nil
	}

	// validation already done in validateRequestExt(), directly build a map here for downstream processing
	multiBidMap := make(map[string]openrtb_ext.ExtMultiBid)
	for _, multiBid := range prebid.MultiBid {
		if multiBid.Bidder != "" {
			if bidderNormalized, bidderFound := openrtb_ext.NormalizeBidderName(multiBid.Bidder); bidderFound {
				multiBidMap[string(bidderNormalized)] = *multiBid
			}
		} else {
			for _, bidder := range multiBid.Bidders {
				if bidderNormalized, bidderFound := openrtb_ext.NormalizeBidderName(bidder); bidderFound {
					multiBidMap[string(bidderNormalized)] = *multiBid
				}
			}
		}
	}

	return multiBidMap
}

func (e *exchange) parseGDPRDefaultValue(r *openrtb_ext.RequestWrapper, eeaCountries []string) gdpr.Signal {
	gdprDefaultValue := e.gdprDefaultValue

	var geo *openrtb2.Geo
	if r.User != nil && r.User.Geo != nil {
		geo = r.User.Geo
	} else if r.Device != nil && r.Device.Geo != nil {
		geo = r.Device.Geo
	}

	if geo != nil {
		// If the country is in the EEA list, GDPR applies.
		// Otherwise, if the country code is properly formatted (3 characters), GDPR does not apply.
		if isEEACountry(geo.Country, eeaCountries) {
			gdprDefaultValue = gdpr.SignalYes
		} else if len(geo.Country) == 3 {
			gdprDefaultValue = gdpr.SignalNo
		}
	}

	return gdprDefaultValue
}

func recordImpMetrics(r *openrtb_ext.RequestWrapper, metricsEngine metrics.MetricsEngine) {
	for _, impInRequest := range r.GetImp() {
		var impLabels metrics.ImpLabels = metrics.ImpLabels{
			BannerImps: impInRequest.Banner != nil,
			VideoImps:  impInRequest.Video != nil,
			AudioImps:  impInRequest.Audio != nil,
			NativeImps: impInRequest.Native != nil,
		}
		metricsEngine.RecordImps(impLabels)
	}
}

// applyDealSupport updates targeting keys with deal prefixes if minimum deal tier exceeded
func applyDealSupport(bidRequest *openrtb2.BidRequest, auc *auction, bidCategory map[string]string, multiBid map[string]openrtb_ext.ExtMultiBid) []error {
	errs := []error{}
	impDealMap := getDealTiers(bidRequest)

	for impID, topBidsPerImp := range auc.allBidsByBidder {
		impDeal := impDealMap[impID]
		for bidder, topBidsPerBidder := range topBidsPerImp {
			bidderNormalized, bidderFound := openrtb_ext.NormalizeBidderName(bidder.String())
			if !bidderFound {
				bidderNormalized = bidder
			}

			maxBid := bidsToUpdate(multiBid, bidderNormalized.String())
			for i, topBid := range topBidsPerBidder {
				if i == maxBid {
					break
				}
				if topBid.DealPriority > 0 {
					if validateDealTier(impDeal[bidderNormalized]) {
						updateHbPbCatDur(topBid, impDeal[bidderNormalized], bidCategory)
					} else {
						errs = append(errs, fmt.Errorf("dealTier configuration invalid for bidder '%s', imp ID '%s'", string(bidder), impID))
					}
				}
			}
		}
	}

	return errs
}

// By default, update 1 bid,
// For 2nd and the following bids, updateHbPbCatDur only if this bidder's multibid config is fully defined.
func bidsToUpdate(multiBid map[string]openrtb_ext.ExtMultiBid, bidder string) int {
	if multiBid != nil {
		if bidderMultiBid, ok := multiBid[bidder]; ok && bidderMultiBid.TargetBidderCodePrefix != "" {
			return *bidderMultiBid.MaxBids
		}
	}

	return openrtb_ext.DefaultBidLimit
}

// getDealTiers creates map of impression to bidder deal tier configuration
func getDealTiers(bidRequest *openrtb2.BidRequest) map[string]openrtb_ext.DealTierBidderMap {
	impDealMap := make(map[string]openrtb_ext.DealTierBidderMap)

	for _, imp := range bidRequest.Imp {
		dealTierBidderMap, err := openrtb_ext.ReadDealTiersFromImp(imp)
		if err != nil {
			continue
		}
		impDealMap[imp.ID] = dealTierBidderMap
	}

	return impDealMap
}

func validateDealTier(dealTier openrtb_ext.DealTier) bool {
	return len(dealTier.Prefix) > 0 && dealTier.MinDealTier > 0
}

func updateHbPbCatDur(bid *entities.PbsOrtbBid, dealTier openrtb_ext.DealTier, bidCategory map[string]string) {
	if bid.DealPriority >= dealTier.MinDealTier {
		prefixTier := fmt.Sprintf("%s%d_", dealTier.Prefix, bid.DealPriority)
		bid.DealTierSatisfied = true

		if oldCatDur, ok := bidCategory[bid.Bid.ID]; ok {
			oldCatDurSplit := strings.SplitAfterN(oldCatDur, "_", 2)
			oldCatDurSplit[0] = prefixTier

			newCatDur := strings.Join(oldCatDurSplit, "")
			bidCategory[bid.Bid.ID] = newCatDur
		}
	}
}

func (e *exchange) makeAuctionContext(ctx context.Context, needsCache bool) (auctionCtx context.Context, cancel context.CancelFunc) {
	auctionCtx = ctx
	cancel = func() {}
	if needsCache {
		if deadline, ok := ctx.Deadline(); ok {
			auctionCtx, cancel = context.WithDeadline(ctx, deadline.Add(-e.cacheTime))
		}
	}
	return
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) getAllBids(
	ctx context.Context,
	bidderRequests []BidderRequest,
	bidAdjustments map[string]float64,
	conversions currency.Conversions,
	accountDebugAllowed bool,
	globalPrivacyControlHeader string,
	headerDebugAllowed bool,
	alternateBidderCodes openrtb_ext.ExtAlternateBidderCodes,
	experiment *openrtb_ext.Experiment,
	hookExecutor hookexecution.StageExecutor,
	pbsRequestStartTime time.Time,
	bidAdjustmentRules map[string][]openrtb_ext.Adjustment,
	tmaxAdjustments *TmaxAdjustmentsPreprocessed,
	responseDebugAllowed bool,
	liveAdaptersPreferredMediaType openrtb_ext.PreferredMediaType) (
	map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
	map[openrtb_ext.BidderName]*seatResponseExtra,
	extraAuctionResponseInfo) {
	// Set up pointers to the bid results
	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, len(bidderRequests))
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, len(bidderRequests))
	chBids := make(chan *bidResponseWrapper, len(bidderRequests))
	extraRespInfo := extraAuctionResponseInfo{seatNonBidBuilder: SeatNonBidBuilder{}}

	e.me.RecordOverheadTime(metrics.MakeBidderRequests, time.Since(pbsRequestStartTime))

	for _, bidder := range bidderRequests {
		// Here we actually call the adapters and collect the bids.
		bidderRunner := e.recoverSafely(bidderRequests, func(bidderRequest BidderRequest, conversions currency.Conversions) {
			// Passing in aName so a doesn't change out from under the go routine
			if bidderRequest.BidderLabels.Adapter == "" {
				glog.Errorf("Exchange: bidlables for %s (%s) missing adapter string", bidderRequest.BidderName, bidderRequest.BidderCoreName)
				bidderRequest.BidderLabels.Adapter = bidderRequest.BidderCoreName
			}
			brw := new(bidResponseWrapper)
			brw.bidder = bidderRequest.BidderName
			brw.adapter = bidderRequest.BidderCoreName
			// Defer basic metrics to insure we capture them after all the values have been set
			defer func() {
				e.me.RecordAdapterRequest(bidderRequest.BidderLabels)
			}()
			start := time.Now()

			reqInfo := adapters.NewExtraRequestInfo(conversions)
			reqInfo.PbsEntryPoint = bidderRequest.BidderLabels.RType
			reqInfo.GlobalPrivacyControlHeader = globalPrivacyControlHeader

			if len(liveAdaptersPreferredMediaType) > 0 {
				if mtype, found := liveAdaptersPreferredMediaType[bidder.BidderName]; found {
					reqInfo.PreferredMediaType = mtype
				}
			}

			bidReqOptions := bidRequestOptions{
				accountDebugAllowed:    accountDebugAllowed,
				headerDebugAllowed:     headerDebugAllowed,
				addCallSignHeader:      isAdsCertEnabled(experiment, e.bidderInfo[string(bidderRequest.BidderName)]),
				bidAdjustments:         bidAdjustments,
				tmaxAdjustments:        tmaxAdjustments,
				bidderRequestStartTime: start,
				responseDebugAllowed:   responseDebugAllowed,
			}
			seatBids, extraBidderRespInfo, err := e.adapterMap[bidderRequest.BidderCoreName].requestBid(ctx, bidderRequest, conversions, &reqInfo, e.adsCertSigner, bidReqOptions, alternateBidderCodes, hookExecutor, bidAdjustmentRules)
			brw.bidderResponseStartTime = extraBidderRespInfo.respProcessingStartTime

			// Add in time reporting
			elapsed := time.Since(start)
			brw.adapterSeatBids = seatBids
			brw.seatNonBidBuilder = extraBidderRespInfo.seatNonBidBuilder
			// Structure to record extra tracking data generated during bidding
			ae := new(seatResponseExtra)
			ae.ResponseTimeMillis = int(elapsed / time.Millisecond)
			if len(seatBids) != 0 {
				ae.HttpCalls = seatBids[0].HttpCalls
			}
			// Timing statistics
			e.me.RecordAdapterTime(bidderRequest.BidderLabels, elapsed)
			bidderRequest.BidderLabels.AdapterBids = bidsToMetric(brw.adapterSeatBids)
			bidderRequest.BidderLabels.AdapterErrors = errorsToMetric(err)
			// Append any bid validation errors to the error list
			ae.Errors = errsToBidderErrors(err)
			ae.Warnings = errsToBidderWarnings(err)
			brw.adapterExtra = ae
			for _, seatBid := range seatBids {
				if seatBid != nil {
					for _, bid := range seatBid.Bids {
						var cpm = float64(bid.Bid.Price * 1000)
						e.me.RecordAdapterPrice(bidderRequest.BidderLabels, cpm)
						e.me.RecordAdapterBidReceived(bidderRequest.BidderLabels, bid.BidType, bid.Bid.AdM != "")
					}
				}
			}
			chBids <- brw
		}, chBids)
		go bidderRunner(bidder, conversions)
	}

	// Wait for the bidders to do their thing
	for i := 0; i < len(bidderRequests); i++ {
		brw := <-chBids
		if !brw.bidderResponseStartTime.IsZero() {
			extraRespInfo.bidderResponseStartTime = brw.bidderResponseStartTime
		}
		//if bidder returned no bids back - remove bidder from further processing
		for _, seatBid := range brw.adapterSeatBids {
			if seatBid != nil {
				bidderName := openrtb_ext.BidderName(seatBid.Seat)
				if len(seatBid.Bids) != 0 {
					if val, ok := adapterBids[bidderName]; ok {
						adapterBids[bidderName].Bids = append(val.Bids, seatBid.Bids...)
					} else {
						adapterBids[bidderName] = seatBid
					}
					extraRespInfo.bidsFound = true
				}
				// collect fledgeAuctionConfigs separately from bids, as empty seatBids may be discarded
				extraRespInfo.fledge = collectFledgeFromSeatBid(extraRespInfo.fledge, bidderName, brw.adapter, seatBid)
			}
		}
		//but we need to add all bidders data to adapterExtra to have metrics and other metadata
		adapterExtra[brw.bidder] = brw.adapterExtra

		// collect adapter non bids
		extraRespInfo.seatNonBidBuilder.append(brw.seatNonBidBuilder)

	}

	return adapterBids, adapterExtra, extraRespInfo
}

func collectFledgeFromSeatBid(fledge *openrtb_ext.Fledge, bidderName openrtb_ext.BidderName, adapterName openrtb_ext.BidderName, seatBid *entities.PbsOrtbSeatBid) *openrtb_ext.Fledge {
	if seatBid.FledgeAuctionConfigs != nil {
		if fledge == nil {
			fledge = &openrtb_ext.Fledge{
				AuctionConfigs: make([]*openrtb_ext.FledgeAuctionConfig, 0, len(seatBid.FledgeAuctionConfigs)),
			}
		}
		for _, config := range seatBid.FledgeAuctionConfigs {
			fledge.AuctionConfigs = append(fledge.AuctionConfigs, &openrtb_ext.FledgeAuctionConfig{
				Bidder:  bidderName.String(),
				Adapter: config.Bidder,
				ImpId:   config.ImpId,
				Config:  config.Config,
			})
		}
	}
	return fledge
}

func (e *exchange) recoverSafely(bidderRequests []BidderRequest,
	inner func(BidderRequest, currency.Conversions),
	chBids chan *bidResponseWrapper) func(BidderRequest, currency.Conversions) {
	return func(bidderRequest BidderRequest, conversions currency.Conversions) {
		defer func() {
			if r := recover(); r != nil {

				allBidders := ""
				sb := strings.Builder{}
				for _, bidder := range bidderRequests {
					sb.WriteString(bidder.BidderName.String())
					sb.WriteString(",")
				}
				if sb.Len() > 0 {
					allBidders = sb.String()[:sb.Len()-1]
				}

				glog.Errorf("OpenRTB auction recovered panic from Bidder %s: %v. "+
					"Account id: %s, All Bidders: %s, Stack trace is: %v",
					bidderRequest.BidderCoreName, r, bidderRequest.BidderLabels.PubID, allBidders, string(debug.Stack()))
				e.me.RecordAdapterPanic(bidderRequest.BidderLabels)
				// Let the master request know that there is no data here
				brw := new(bidResponseWrapper)
				brw.adapterExtra = new(seatResponseExtra)
				chBids <- brw
			}
		}()
		inner(bidderRequest, conversions)
	}
}

func bidsToMetric(seatBids []*entities.PbsOrtbSeatBid) metrics.AdapterBid {
	for _, seatBid := range seatBids {
		if seatBid != nil && len(seatBid.Bids) != 0 {
			return metrics.AdapterBidPresent
		}
	}
	return metrics.AdapterBidNone
}

func errorsToMetric(errs []error) map[metrics.AdapterError]struct{} {
	if len(errs) == 0 {
		return nil
	}
	ret := make(map[metrics.AdapterError]struct{}, len(errs))
	var s struct{}
	for _, err := range errs {
		switch errortypes.ReadCode(err) {
		case errortypes.TimeoutErrorCode:
			ret[metrics.AdapterErrorTimeout] = s
		case errortypes.BadInputErrorCode:
			ret[metrics.AdapterErrorBadInput] = s
		case errortypes.BadServerResponseErrorCode:
			ret[metrics.AdapterErrorBadServerResponse] = s
		case errortypes.FailedToRequestBidsErrorCode:
			ret[metrics.AdapterErrorFailedToRequestBids] = s
		case errortypes.AlternateBidderCodeWarningCode:
			ret[metrics.AdapterErrorValidation] = s
		case errortypes.TmaxTimeoutErrorCode:
			ret[metrics.AdapterErrorTmaxTimeout] = s
		default:
			ret[metrics.AdapterErrorUnknown] = s
		}
	}
	return ret
}

func errsToBidderErrors(errs []error) []openrtb_ext.ExtBidderMessage {
	sErr := make([]openrtb_ext.ExtBidderMessage, 0)
	for _, err := range errortypes.FatalOnly(errs) {
		newErr := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.ReadCode(err),
			Message: err.Error(),
		}
		sErr = append(sErr, newErr)
	}

	return sErr
}

func errsToBidderWarnings(errs []error) []openrtb_ext.ExtBidderMessage {
	sWarn := make([]openrtb_ext.ExtBidderMessage, 0)
	for _, warn := range errortypes.WarningOnly(errs) {
		newErr := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.ReadCode(warn),
			Message: warn.Error(),
		}
		sWarn = append(sWarn, newErr)
	}
	return sWarn
}

// This piece takes all the bids supplied by the adapters and crafts an openRTB response to send back to the requester
func (e *exchange) buildBidResponse(ctx context.Context, liveAdapters []openrtb_ext.BidderName, adapterSeatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, bidRequest *openrtb_ext.RequestWrapper, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, auc *auction, bidResponseExt *openrtb_ext.ExtBidResponse, returnCreative bool, impExtInfoMap map[string]ImpExtInfo, pubID string, errList []error, seatNonBidBuilder *SeatNonBidBuilder) *openrtb2.BidResponse {
	bidResponse := new(openrtb2.BidResponse)

	bidResponse.ID = bidRequest.ID
	if len(liveAdapters) == 0 {
		// signal "Invalid Request" if no valid bidders.
		bidResponse.NBR = openrtb3.NoBidInvalidRequest.Ptr()
	}

	// Create the SeatBids. We use a zero sized slice so that we can append non-zero seat bids, and not include seatBid
	// objects for seatBids without any bids. Preallocate the max possible size to avoid reallocating the array as we go.
	seatBids := make([]openrtb2.SeatBid, 0, len(liveAdapters))
	for a, adapterSeatBids := range adapterSeatBids {
		//while processing every single bib, do we need to handle categories here?
		if adapterSeatBids != nil && len(adapterSeatBids.Bids) > 0 {
			sb := e.makeSeatBid(adapterSeatBids, a, adapterExtra, auc, returnCreative, impExtInfoMap, bidRequest, bidResponseExt, pubID, seatNonBidBuilder)
			seatBids = append(seatBids, *sb)
			bidResponse.Cur = adapterSeatBids.Currency
		}
	}
	bidResponse.SeatBid = seatBids

	return bidResponse
}

func encodeBidResponseExt(bidResponseExt *openrtb_ext.ExtBidResponse) ([]byte, error) {
	buffer := &bytes.Buffer{}
	enc := json.NewEncoder(buffer)

	enc.SetEscapeHTML(false)
	err := enc.Encode(bidResponseExt)

	return buffer.Bytes(), err
}

func applyCategoryMapping(ctx context.Context, targeting openrtb_ext.ExtRequestTargeting, seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, categoriesFetcher stored_requests.CategoryFetcher, targData *targetData, booleanGenerator deduplicateChanceGenerator, seatNonBidBuilder *SeatNonBidBuilder) (map[string]string, map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, []string, error) {
	res := make(map[string]string)

	type bidDedupe struct {
		bidderName openrtb_ext.BidderName
		bidIndex   int
		bidID      string
		bidPrice   string
	}

	dedupe := make(map[string]bidDedupe)

	// applyCategoryMapping doesn't get called unless
	brandCatExt := targeting.IncludeBrandCategory

	//If ext.prebid.targeting.includebrandcategory is present in ext then competitive exclusion feature is on.
	var includeBrandCategory = brandCatExt != nil //if not present - category will no be appended
	appendBidderNames := targeting.AppendBidderNames

	var primaryAdServer string
	var publisher string
	var err error
	var rejections []string
	var translateCategories = true

	if includeBrandCategory && brandCatExt.WithCategory {
		if brandCatExt.TranslateCategories != nil {
			translateCategories = *brandCatExt.TranslateCategories
		}
		//if translateCategories is set to false, ignore checking primaryAdServer and publisher
		if translateCategories {
			//if ext.prebid.targeting.includebrandcategory present but primaryadserver/publisher not present then error out the request right away.
			primaryAdServer, err = getPrimaryAdServer(brandCatExt.PrimaryAdServer) //1-Freewheel 2-DFP
			if err != nil {
				return res, seatBids, rejections, err
			}
			publisher = brandCatExt.Publisher
		}
	}

	seatBidsToRemove := make([]openrtb_ext.BidderName, 0)

	for bidderName, seatBid := range seatBids {
		bidsToRemove := make([]int, 0)
		for bidInd := range seatBid.Bids {
			bid := seatBid.Bids[bidInd]
			bidID := bid.Bid.ID
			var duration int
			var category string
			var priceBucket string

			if bid.BidVideo != nil {
				duration = bid.BidVideo.Duration
				category = bid.BidVideo.PrimaryCategory
			}
			if brandCatExt.WithCategory && category == "" {
				bidIabCat := bid.Bid.Cat
				if len(bidIabCat) != 1 {
					//TODO: add metrics
					//on receiving bids from adapters if no unique IAB category is returned  or if no ad server category is returned discard the bid
					bidsToRemove = append(bidsToRemove, bidInd)
					rejections = updateRejections(rejections, bidID, "Bid did not contain a category")
					seatNonBidBuilder.rejectBid(bid, int(ResponseRejectedCategoryMappingInvalid), string(bidderName))
					continue
				}
				if translateCategories {
					//if unique IAB category is present then translate it to the adserver category based on mapping file
					category, err = categoriesFetcher.FetchCategories(ctx, primaryAdServer, publisher, bidIabCat[0])
					if err != nil || category == "" {
						//TODO: add metrics
						//if mapping required but no mapping file is found then discard the bid
						bidsToRemove = append(bidsToRemove, bidInd)
						reason := fmt.Sprintf("Category mapping file for primary ad server: '%s', publisher: '%s' not found", primaryAdServer, publisher)
						rejections = updateRejections(rejections, bidID, reason)
						continue
					}
				} else {
					//category translation is disabled, continue with IAB category
					category = bidIabCat[0]
				}
			}

			// TODO: consider should we remove bids with zero duration here?

			priceBucket = GetPriceBucket(*bid.Bid, *targData)

			newDur, err := findDurationRange(duration, targeting.DurationRangeSec)
			if err != nil {
				bidsToRemove = append(bidsToRemove, bidInd)
				rejections = updateRejections(rejections, bidID, err.Error())
				continue
			}

			var categoryDuration string
			var dupeKey string
			if brandCatExt.WithCategory {
				categoryDuration = fmt.Sprintf("%s_%s_%ds", priceBucket, category, newDur)
				dupeKey = category
			} else {
				categoryDuration = fmt.Sprintf("%s_%ds", priceBucket, newDur)
				dupeKey = categoryDuration
			}

			if appendBidderNames {
				categoryDuration = fmt.Sprintf("%s_%s", categoryDuration, bidderName.String())
			}

			if dupe, ok := dedupe[dupeKey]; ok {

				dupeBidPrice, err := strconv.ParseFloat(dupe.bidPrice, 64)
				if err != nil {
					dupeBidPrice = 0
				}
				currBidPrice, err := strconv.ParseFloat(priceBucket, 64)
				if err != nil {
					currBidPrice = 0
				}
				if dupeBidPrice == currBidPrice {
					if booleanGenerator.Generate() {
						dupeBidPrice = -1
					} else {
						currBidPrice = -1
					}
				}

				if dupeBidPrice < currBidPrice {
					if dupe.bidderName == bidderName {
						// An older bid from the current bidder
						bidsToRemove = append(bidsToRemove, dupe.bidIndex)
						rejections = updateRejections(rejections, dupe.bidID, "Bid was deduplicated")
					} else {
						// An older bid from a different seatBid we've already finished with
						oldSeatBid := (seatBids)[dupe.bidderName]
						rejections = updateRejections(rejections, dupe.bidID, "Bid was deduplicated")
						if len(oldSeatBid.Bids) == 1 {
							seatBidsToRemove = append(seatBidsToRemove, dupe.bidderName)
						} else {
							// This is a very rare, but still possible case where bid needs to be removed from already processed bidder
							// This happens when current processing bidder has a bid that has same deduplication key as a bid from already processed bidder
							// and already processed bid was selected to be removed
							// See example of input data in unit test `TestCategoryMappingTwoBiddersManyBidsEachNoCategorySamePrice`
							// Need to remove bid by name, not index in this case
							removeBidById(oldSeatBid, dupe.bidID)
						}
					}
					delete(res, dupe.bidID)
				} else {
					// Remove this bid
					bidsToRemove = append(bidsToRemove, bidInd)
					rejections = updateRejections(rejections, bidID, "Bid was deduplicated")
					continue
				}
			}
			res[bidID] = categoryDuration
			dedupe[dupeKey] = bidDedupe{bidderName: bidderName, bidIndex: bidInd, bidID: bidID, bidPrice: priceBucket}
		}

		if len(bidsToRemove) > 0 {
			sort.Ints(bidsToRemove)
			if len(bidsToRemove) == len(seatBid.Bids) {
				//if all bids are invalid - remove entire seat bid
				seatBidsToRemove = append(seatBidsToRemove, bidderName)
			} else {
				bids := seatBid.Bids
				for i := len(bidsToRemove) - 1; i >= 0; i-- {
					remInd := bidsToRemove[i]
					bids = append(bids[:remInd], bids[remInd+1:]...)
				}
				seatBid.Bids = bids
			}
		}

	}
	for _, seatBidInd := range seatBidsToRemove {
		seatBids[seatBidInd].Bids = nil
	}

	return res, seatBids, rejections, nil
}

// findDurationRange returns the element in the array 'durationRanges' that is both greater than 'dur' and closest
// in value to 'dur' unless a value equal to 'dur' is found. Returns an error if all elements in 'durationRanges'
// are less than 'dur'.
func findDurationRange(dur int, durationRanges []int) (int, error) {
	newDur := dur
	madeSelection := false
	var err error

	for i := range durationRanges {
		if dur > durationRanges[i] {
			continue
		}
		if dur == durationRanges[i] {
			return durationRanges[i], nil
		}
		// dur < durationRanges[i]
		if durationRanges[i] < newDur || !madeSelection {
			newDur = durationRanges[i]
			madeSelection = true
		}
	}
	if !madeSelection && len(durationRanges) > 0 {
		err = errors.New("bid duration exceeds maximum allowed")
	}
	return newDur, err
}

func removeBidById(seatBid *entities.PbsOrtbSeatBid, bidID string) {
	//Find index of bid to remove
	dupeBidIndex := -1
	for i, bid := range seatBid.Bids {
		if bid.Bid.ID == bidID {
			dupeBidIndex = i
			break
		}
	}
	if dupeBidIndex != -1 {
		if dupeBidIndex < len(seatBid.Bids)-1 {
			seatBid.Bids = append(seatBid.Bids[:dupeBidIndex], seatBid.Bids[dupeBidIndex+1:]...)
		} else if dupeBidIndex == len(seatBid.Bids)-1 {
			seatBid.Bids = seatBid.Bids[:len(seatBid.Bids)-1]
		}
	}
}

func updateRejections(rejections []string, bidID string, reason string) []string {
	message := fmt.Sprintf("bid rejected [bid ID: %s] reason: %s", bidID, reason)
	return append(rejections, message)
}

func getPrimaryAdServer(adServerId int) (string, error) {
	switch adServerId {
	case 1:
		return "freewheel", nil
	case 2:
		return "dfp", nil
	default:
		return "", fmt.Errorf("Primary ad server %d not recognized", adServerId)
	}
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) makeExtBidResponse(adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, r AuctionRequest, debugInfo bool, passthrough json.RawMessage, fledge *openrtb_ext.Fledge, errList []error) *openrtb_ext.ExtBidResponse {
	bidResponseExt := &openrtb_ext.ExtBidResponse{
		Errors:               make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage, len(adapterBids)),
		Warnings:             make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderMessage, len(adapterBids)),
		ResponseTimeMillis:   make(map[openrtb_ext.BidderName]int, len(adapterBids)),
		RequestTimeoutMillis: r.BidRequestWrapper.BidRequest.TMax,
	}
	if debugInfo {
		bidResponseExt.Debug = &openrtb_ext.ExtResponseDebug{
			HttpCalls:       make(map[openrtb_ext.BidderName][]*openrtb_ext.ExtHttpCall),
			ResolvedRequest: r.ResolvedBidRequest,
		}
	}

	var auctionTimestamp int64
	if !r.StartTime.IsZero() {
		auctionTimestamp = r.StartTime.UnixMilli()
	}

	if auctionTimestamp > 0 ||
		passthrough != nil ||
		fledge != nil {
		bidResponseExt.Prebid = &openrtb_ext.ExtResponsePrebid{
			AuctionTimestamp: auctionTimestamp,
			Passthrough:      passthrough,
			Fledge:           fledge,
		}
	}

	for bidderName, responseExtra := range adapterExtra {

		if debugInfo && len(responseExtra.HttpCalls) > 0 {
			bidResponseExt.Debug.HttpCalls[bidderName] = responseExtra.HttpCalls
		}
		if len(responseExtra.Warnings) > 0 {
			bidResponseExt.Warnings[bidderName] = responseExtra.Warnings
		}
		// Only make an entry for bidder errors if the bidder reported any.
		if len(responseExtra.Errors) > 0 {
			bidResponseExt.Errors[bidderName] = responseExtra.Errors
		}
		if len(errList) > 0 {
			bidResponseExt.Errors[openrtb_ext.PrebidExtKey] = errsToBidderErrors(errList)
			if prebidWarn := errsToBidderWarnings(errList); len(prebidWarn) > 0 {
				bidResponseExt.Warnings[openrtb_ext.PrebidExtKey] = prebidWarn
			}
		}
		bidResponseExt.ResponseTimeMillis[bidderName] = responseExtra.ResponseTimeMillis
		// Defering the filling of bidResponseExt.Usersync[bidderName] until later

	}

	return bidResponseExt
}

// Return an openrtb seatBid for a bidder
// buildBidResponse is responsible for ensuring nil bid seatbids are not included
func (e *exchange) makeSeatBid(adapterBid *entities.PbsOrtbSeatBid, adapter openrtb_ext.BidderName, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, auc *auction, returnCreative bool, impExtInfoMap map[string]ImpExtInfo, bidRequest *openrtb_ext.RequestWrapper, bidResponseExt *openrtb_ext.ExtBidResponse, pubID string, seatNonBidBuilder *SeatNonBidBuilder) *openrtb2.SeatBid {
	seatBid := &openrtb2.SeatBid{
		Seat:  adapter.String(),
		Group: 0, // Prebid cannot support roadblocking
	}

	var errList []error
	seatBid.Bid, errList = e.makeBid(adapterBid.Bids, auc, returnCreative, impExtInfoMap, bidRequest, bidResponseExt, adapter, pubID, seatNonBidBuilder)
	if len(errList) > 0 {
		adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, errsToBidderErrors(errList)...)
	}

	return seatBid
}

func (e *exchange) makeBid(bids []*entities.PbsOrtbBid, auc *auction, returnCreative bool, impExtInfoMap map[string]ImpExtInfo, bidRequest *openrtb_ext.RequestWrapper, bidResponseExt *openrtb_ext.ExtBidResponse, adapter openrtb_ext.BidderName, pubID string, seatNonBidBuilder *SeatNonBidBuilder) ([]openrtb2.Bid, []error) {
	result := make([]openrtb2.Bid, 0, len(bids))
	errs := make([]error, 0, 1)

	for _, bid := range bids {
		if err := dsa.Validate(bidRequest, bid); err != nil {
			dsaMessage := openrtb_ext.ExtBidderMessage{
				Code:    errortypes.InvalidBidResponseDSAWarningCode,
				Message: fmt.Sprintf("bid rejected: %s", err.Error()),
			}
			bidResponseExt.Warnings[adapter] = append(bidResponseExt.Warnings[adapter], dsaMessage)

			seatNonBidBuilder.rejectBid(bid, int(ResponseRejectedGeneral), adapter.String())
			continue // Don't add bid to result
		}
		if e.bidValidationEnforcement.BannerCreativeMaxSize == config.ValidationEnforce && bid.BidType == openrtb_ext.BidTypeBanner {
			if !e.validateBannerCreativeSize(bid, bidResponseExt, adapter, pubID, e.bidValidationEnforcement.BannerCreativeMaxSize) {
				seatNonBidBuilder.rejectBid(bid, int(ResponseRejectedCreativeSizeNotAllowed), adapter.String())
				continue // Don't add bid to result
			}
		} else if e.bidValidationEnforcement.BannerCreativeMaxSize == config.ValidationWarn && bid.BidType == openrtb_ext.BidTypeBanner {
			e.validateBannerCreativeSize(bid, bidResponseExt, adapter, pubID, e.bidValidationEnforcement.BannerCreativeMaxSize)
		}
		if _, ok := impExtInfoMap[bid.Bid.ImpID]; ok {
			if e.bidValidationEnforcement.SecureMarkup == config.ValidationEnforce && (bid.BidType == openrtb_ext.BidTypeBanner || bid.BidType == openrtb_ext.BidTypeVideo) {
				if !e.validateBidAdM(bid, bidResponseExt, adapter, pubID, e.bidValidationEnforcement.SecureMarkup) {
					seatNonBidBuilder.rejectBid(bid, int(ResponseRejectedCreativeNotSecure), adapter.String())
					continue // Don't add bid to result
				}
			} else if e.bidValidationEnforcement.SecureMarkup == config.ValidationWarn && (bid.BidType == openrtb_ext.BidTypeBanner || bid.BidType == openrtb_ext.BidTypeVideo) {
				e.validateBidAdM(bid, bidResponseExt, adapter, pubID, e.bidValidationEnforcement.SecureMarkup)
			}

		}
		bidExtPrebid := &openrtb_ext.ExtBidPrebid{
			DealPriority:      bid.DealPriority,
			DealTierSatisfied: bid.DealTierSatisfied,
			Events:            bid.BidEvents,
			Targeting:         bid.BidTargets,
			Floors:            bid.BidFloors,
			Type:              bid.BidType,
			Meta:              bid.BidMeta,
			Video:             bid.BidVideo,
			BidId:             bid.GeneratedBidID,
			TargetBidderCode:  bid.TargetBidderCode,
		}

		if cacheInfo, found := e.getBidCacheInfo(bid, auc); found {
			bidExtPrebid.Cache = &openrtb_ext.ExtBidPrebidCache{
				Bids: &cacheInfo,
			}
		}

		if bidExtJSON, err := makeBidExtJSON(bid.Bid.Ext, bidExtPrebid, impExtInfoMap, bid.Bid.ImpID, bid.OriginalBidCPM, bid.OriginalBidCur, bid.AdapterCode); err != nil {
			errs = append(errs, err)
		} else {
			result = append(result, *bid.Bid)
			resultBid := &result[len(result)-1]
			resultBid.Ext = bidExtJSON
			if !returnCreative {
				resultBid.AdM = ""
			}
		}
	}
	return result, errs
}

func makeBidExtJSON(ext json.RawMessage, prebid *openrtb_ext.ExtBidPrebid, impExtInfoMap map[string]ImpExtInfo, impId string, originalBidCpm float64, originalBidCur string, adapter openrtb_ext.BidderName) (json.RawMessage, error) {
	var extMap map[string]interface{}

	if len(ext) != 0 {
		if err := jsonutil.Unmarshal(ext, &extMap); err != nil {
			return nil, err
		}
	} else {
		extMap = make(map[string]interface{})
	}

	//ext.origbidcpm
	if originalBidCpm >= 0 {
		extMap[openrtb_ext.OriginalBidCpmKey] = originalBidCpm
	}

	//ext.origbidcur
	if originalBidCur != "" {
		extMap[openrtb_ext.OriginalBidCurKey] = originalBidCur
	}

	// ext.prebid
	if prebid.Meta == nil && maputil.HasElement(extMap, "prebid", "meta") {
		metaContainer := struct {
			Prebid struct {
				Meta openrtb_ext.ExtBidPrebidMeta `json:"meta"`
			} `json:"prebid"`
		}{}
		if err := jsonutil.Unmarshal(ext, &metaContainer); err != nil {
			return nil, fmt.Errorf("error validating response from server, %s", err)
		}
		prebid.Meta = &metaContainer.Prebid.Meta
	}

	if prebid.Meta == nil {
		prebid.Meta = &openrtb_ext.ExtBidPrebidMeta{}
	}

	prebid.Meta.AdapterCode = adapter.String()

	// ext.prebid.storedrequestattributes and ext.prebid.passthrough
	if impExtInfo, ok := impExtInfoMap[impId]; ok {
		prebid.Passthrough = impExtInfoMap[impId].Passthrough
		if impExtInfo.EchoVideoAttrs {
			videoData, _, _, err := jsonparser.Get(impExtInfo.StoredImp, "video")
			if err != nil && err != jsonparser.KeyPathNotFoundError {
				return nil, err
			}
			//handler for case where EchoVideoAttrs is true, but video data is not found
			if len(videoData) > 0 {
				extMap[openrtb_ext.StoredRequestAttributes] = json.RawMessage(videoData)
			}
		}
	}
	extMap[openrtb_ext.PrebidExtKey] = prebid
	return jsonutil.Marshal(extMap)
}

// If bid got cached inside `(a *auction) doCache(ctx context.Context, cache prebid_cache_client.Client, targData *targetData, bidRequest *openrtb2.BidRequest, ttlBuffer int64, defaultTTLs *config.DefaultTTLs, bidCategory map[string]string)`,
// a UUID should be found inside `a.cacheIds` or `a.vastCacheIds`. This function returns the UUID along with the internal cache URL
func (e *exchange) getBidCacheInfo(bid *entities.PbsOrtbBid, auction *auction) (cacheInfo openrtb_ext.ExtBidPrebidCacheBids, found bool) {
	uuid, found := findCacheID(bid, auction)

	if found {
		cacheInfo.CacheId = uuid
		cacheInfo.Url = buildCacheURL(e.cache, uuid)
	}

	return
}

func findCacheID(bid *entities.PbsOrtbBid, auction *auction) (string, bool) {
	if bid != nil && bid.Bid != nil && auction != nil {
		if id, found := auction.cacheIds[bid.Bid]; found {
			return id, true
		}

		if id, found := auction.vastCacheIds[bid.Bid]; found {
			return id, true
		}
	}

	return "", false
}

func buildCacheURL(cache prebid_cache_client.Client, uuid string) string {
	scheme, host, path := cache.GetExtCacheData()

	if host == "" || path == "" {
		return ""
	}

	query := url.Values{"uuid": []string{uuid}}
	cacheURL := url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: query.Encode(),
	}
	cacheURL.Query()

	// URLs without a scheme will begin with //, in which case we
	// want to trim it off to keep compatbile with current behavior.
	return strings.TrimPrefix(cacheURL.String(), "//")
}

func listBiddersWithRequests(bidderRequests []BidderRequest) []openrtb_ext.BidderName {
	liveAdapters := make([]openrtb_ext.BidderName, len(bidderRequests))
	i := 0
	for _, bidderRequest := range bidderRequests {
		liveAdapters[i] = bidderRequest.BidderName
		i++
	}
	// Randomize the list of adapters to make the auction more fair
	randomizeList(liveAdapters)

	return liveAdapters
}

func buildStoredAuctionResponse(storedAuctionResponses map[string]json.RawMessage) (
	map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid,
	*openrtb_ext.Fledge,
	[]openrtb_ext.BidderName,
	error) {

	adapterBids := make(map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, 0)
	var fledge *openrtb_ext.Fledge
	liveAdapters := make([]openrtb_ext.BidderName, 0)
	for impId, storedResp := range storedAuctionResponses {
		var seatBids []openrtb2.SeatBid

		if err := jsonutil.UnmarshalValid(storedResp, &seatBids); err != nil {
			return nil, nil, nil, err
		}
		for _, seat := range seatBids {
			var bidsToAdd []*entities.PbsOrtbBid
			//set imp id from request
			for i := range seat.Bid {
				seat.Bid[i].ImpID = impId
				bidType, err := getMediaTypeForBid(seat.Bid[i])
				if err != nil {
					return nil, nil, nil, err
				}
				bidsToAdd = append(bidsToAdd, &entities.PbsOrtbBid{Bid: &seat.Bid[i], BidType: bidType})
			}

			bidderName := openrtb_ext.BidderName(seat.Seat)

			if seat.Ext != nil {
				var seatExt openrtb_ext.ExtBidResponse
				if err := jsonutil.Unmarshal(seat.Ext, &seatExt); err != nil {
					return nil, nil, nil, err
				}
				// add in FLEDGE response with impId substituted
				if seatExt.Prebid != nil &&
					seatExt.Prebid.Fledge != nil &&
					seatExt.Prebid.Fledge.AuctionConfigs != nil {
					auctionConfigs := seatExt.Prebid.Fledge.AuctionConfigs
					if fledge == nil {
						fledge = &openrtb_ext.Fledge{
							AuctionConfigs: make([]*openrtb_ext.FledgeAuctionConfig, 0, len(auctionConfigs)),
						}
					}
					for _, config := range auctionConfigs {
						newConfig := &openrtb_ext.FledgeAuctionConfig{
							ImpId:   impId,
							Bidder:  string(bidderName),
							Adapter: string(bidderName),
							Config:  config.Config,
						}
						fledge.AuctionConfigs = append(fledge.AuctionConfigs, newConfig)
					}
				}
			}

			if _, ok := adapterBids[bidderName]; ok {
				adapterBids[bidderName].Bids = append(adapterBids[bidderName].Bids, bidsToAdd...)

			} else {
				//create new seat bid and add it to live adapters
				liveAdapters = append(liveAdapters, bidderName)
				newSeatBid := entities.PbsOrtbSeatBid{
					Bids:     bidsToAdd,
					Currency: "",
					Seat:     "",
				}
				adapterBids[bidderName] = &newSeatBid

			}
		}
	}

	return adapterBids, fledge, liveAdapters, nil
}

func isAdsCertEnabled(experiment *openrtb_ext.Experiment, info config.BidderInfo) bool {
	requestAdsCertEnabled := experiment != nil && experiment.AdsCert != nil && experiment.AdsCert.Enabled
	bidderAdsCertEnabled := info.Experiment.AdsCert.Enabled
	return requestAdsCertEnabled && bidderAdsCertEnabled
}

func (e exchange) validateBannerCreativeSize(bid *entities.PbsOrtbBid, bidResponseExt *openrtb_ext.ExtBidResponse, adapter openrtb_ext.BidderName, pubID string, validationType string) bool {
	if bid.Bid.W > e.bidValidationEnforcement.MaxCreativeWidth || bid.Bid.H > e.bidValidationEnforcement.MaxCreativeHeight {
		// Add error to debug array
		errorMessage := setErrorMessageCreativeSize(validationType)
		bidCreativeMaxSizeError := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.BadServerResponseErrorCode,
			Message: errorMessage,
		}
		bidResponseExt.Errors[adapter] = append(bidResponseExt.Errors[adapter], bidCreativeMaxSizeError)

		// Log Metrics
		e.me.RecordBidValidationCreativeSizeError(adapter, pubID)

		return false
	}
	return true
}

func (e exchange) validateBidAdM(bid *entities.PbsOrtbBid, bidResponseExt *openrtb_ext.ExtBidResponse, adapter openrtb_ext.BidderName, pubID string, validationType string) bool {
	invalidAdM := []string{"http:", "http%3A"}
	requiredAdM := []string{"https:", "https%3A"}

	if (strings.Contains(bid.Bid.AdM, invalidAdM[0]) || strings.Contains(bid.Bid.AdM, invalidAdM[1])) && (!strings.Contains(bid.Bid.AdM, requiredAdM[0]) && !strings.Contains(bid.Bid.AdM, requiredAdM[1])) {
		// Add error to debug array
		errorMessage := setErrorMessageSecureMarkup(validationType)
		bidSecureMarkupError := openrtb_ext.ExtBidderMessage{
			Code:    errortypes.BadServerResponseErrorCode,
			Message: errorMessage,
		}
		bidResponseExt.Errors[adapter] = append(bidResponseExt.Errors[adapter], bidSecureMarkupError)

		// Log Metrics
		e.me.RecordBidValidationSecureMarkupError(adapter, pubID)

		return false
	}
	return true
}

func setErrorMessageCreativeSize(validationType string) string {
	if validationType == config.ValidationEnforce {
		return "bidResponse rejected: size WxH"
	} else if validationType == config.ValidationWarn {
		return "bidResponse creative size warning: size WxH larger than AdUnit sizes"
	}
	return ""
}

func setErrorMessageSecureMarkup(validationType string) string {
	if validationType == config.ValidationEnforce {
		return "bidResponse rejected: insecure creative in secure context"
	} else if validationType == config.ValidationWarn {
		return "bidResponse secure markup warning: insecure creative in secure contexts"
	}
	return ""
}

// setSeatNonBid adds SeatNonBids within bidResponse.Ext.Prebid.SeatNonBid
func setSeatNonBid(bidResponseExt *openrtb_ext.ExtBidResponse, seatNonBidBuilder SeatNonBidBuilder) *openrtb_ext.ExtBidResponse {
	if len(seatNonBidBuilder) == 0 {
		return bidResponseExt
	}
	if bidResponseExt == nil {
		bidResponseExt = &openrtb_ext.ExtBidResponse{}
	}
	if bidResponseExt.Prebid == nil {
		bidResponseExt.Prebid = &openrtb_ext.ExtResponsePrebid{}
	}

	bidResponseExt.Prebid.SeatNonBid = seatNonBidBuilder.Slice()
	return bidResponseExt
}

func isEEACountry(country string, eeaCountries []string) bool {
	if len(eeaCountries) == 0 {
		return false
	}

	country = strings.ToUpper(country)
	for _, c := range eeaCountries {
		if strings.ToUpper(c) == country {
			return true
		}
	}
	return false
}
