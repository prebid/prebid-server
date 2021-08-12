package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/gofrs/uuid"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb/v15/native1"
	nativeRequests "github.com/mxmCherry/openrtb/v15/native1/request"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	accountService "github.com/prebid/prebid-server/account"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/lmt"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/httputil"
	"github.com/prebid/prebid-server/util/iputil"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/text/currency"
)

const storedRequestTimeoutMillis = 50

var (
	dntKey      string = http.CanonicalHeaderKey("DNT")
	dntDisabled int8   = 0
	dntEnabled  int8   = 1
)

func NewEndpoint(
	ex exchange.Exchange,
	validator openrtb_ext.BidderParamValidator,
	requestsById stored_requests.Fetcher,
	accounts stored_requests.AccountFetcher,
	cfg *config.Configuration,
	met metrics.MetricsEngine,
	pbsAnalytics analytics.PBSAnalyticsModule,
	disabledBidders map[string]string,
	defReqJSON []byte,
	bidderMap map[string]openrtb_ext.BidderName,
) (httprouter.Handle, error) {
	if ex == nil || validator == nil || requestsById == nil || accounts == nil || cfg == nil || met == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}

	defRequest := defReqJSON != nil && len(defReqJSON) > 0

	ipValidator := iputil.PublicNetworkIPValidator{
		IPv4PrivateNetworks: cfg.RequestValidation.IPv4PrivateNetworksParsed,
		IPv6PrivateNetworks: cfg.RequestValidation.IPv6PrivateNetworksParsed,
	}

	return httprouter.Handle((&endpointDeps{
		ex,
		validator,
		requestsById,
		empty_fetcher.EmptyFetcher{},
		accounts,
		cfg,
		met,
		pbsAnalytics,
		disabledBidders,
		defRequest,
		defReqJSON,
		bidderMap,
		nil,
		nil,
		ipValidator}).Auction), nil
}

type endpointDeps struct {
	ex                        exchange.Exchange
	paramsValidator           openrtb_ext.BidderParamValidator
	storedReqFetcher          stored_requests.Fetcher
	videoFetcher              stored_requests.Fetcher
	accounts                  stored_requests.AccountFetcher
	cfg                       *config.Configuration
	metricsEngine             metrics.MetricsEngine
	analytics                 analytics.PBSAnalyticsModule
	disabledBidders           map[string]string
	defaultRequest            bool
	defReqJSON                []byte
	bidderMap                 map[string]openrtb_ext.BidderName
	cache                     prebid_cache_client.Client
	debugLogRegexp            *regexp.Regexp
	privateNetworkIPValidator iputil.IPValidator
}

func (deps *endpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()

	ao := analytics.AuctionObject{
		Status:    http.StatusOK,
		Errors:    make([]error, 0),
		StartTime: start,
	}

	labels := metrics.Labels{
		Source:        metrics.DemandUnknown,
		RType:         metrics.ReqTypeORTB2Web,
		PubID:         metrics.PublisherUnknown,
		CookieFlag:    metrics.CookieFlagUnknown,
		RequestStatus: metrics.RequestStatusOK,
	}
	defer func() {
		deps.metricsEngine.RecordRequest(labels)
		deps.metricsEngine.RecordRequestTime(labels, time.Since(start))
		deps.analytics.LogAuctionObject(&ao)
	}()

	req, impExtInfoMap, errL := deps.parseRequest(r)

	if errortypes.ContainsFatalError(errL) && writeError(errL, w, &labels) {
		return
	}
	warnings := errortypes.WarningOnly(errL)

	ctx := context.Background()

	timeout := deps.cfg.AuctionTimeouts.LimitAuctionTimeout(time.Duration(req.TMax) * time.Millisecond)
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, start.Add(timeout))
		defer cancel()
	}

	usersyncs := usersync.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie))
	if req.App != nil {
		labels.Source = metrics.DemandApp
		labels.RType = metrics.ReqTypeORTB2App
		labels.PubID = getAccountID(req.App.Publisher)
	} else { //req.Site != nil
		labels.Source = metrics.DemandWeb
		if usersyncs.LiveSyncCount() == 0 {
			labels.CookieFlag = metrics.CookieFlagNo
		} else {
			labels.CookieFlag = metrics.CookieFlagYes
		}
		labels.PubID = getAccountID(req.Site.Publisher)
	}

	// Look up account now that we have resolved the pubID value
	account, acctIDErrs := accountService.GetAccount(ctx, deps.cfg, deps.accounts, labels.PubID)
	if len(acctIDErrs) > 0 {
		errL = append(errL, acctIDErrs...)
		writeError(errL, w, &labels)
		return
	}

	// rebuild/resync the request in the request wrapper.
	if err := req.RebuildRequest(); err != nil {
		errL = append(errL, err)
		writeError(errL, w, &labels)
		return
	}

	secGPC := r.Header.Get("Sec-GPC")

	auctionRequest := exchange.AuctionRequest{
		BidRequest:                 req.BidRequest,
		Account:                    *account,
		UserSyncs:                  usersyncs,
		RequestType:                labels.RType,
		StartTime:                  start,
		LegacyLabels:               labels,
		Warnings:                   warnings,
		GlobalPrivacyControlHeader: secGPC,
		ImpExtInfoMap:              impExtInfoMap,
	}

	response, err := deps.ex.HoldAuction(ctx, auctionRequest, nil)
	ao.Request = req.BidRequest
	ao.Response = response
	ao.Account = account
	if err != nil {
		labels.RequestStatus = metrics.RequestStatusErr
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/auction Critical error: %v", err)
		ao.Status = http.StatusInternalServerError
		ao.Errors = append(ao.Errors, err)
		return
	}

	// Fixes #231
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	// Fixes #328
	w.Header().Set("Content-Type", "application/json")

	// If an error happens when encoding the response, there isn't much we can do.
	// If we've sent _any_ bytes, then Go would have sent the 200 status code first.
	// That status code can't be un-sent... so the best we can do is log the error.
	if err := enc.Encode(response); err != nil {
		labels.RequestStatus = metrics.RequestStatusNetworkErr
		ao.Errors = append(ao.Errors, fmt.Errorf("/openrtb2/auction Failed to send response: %v", err))
	}
}

// parseRequest turns the HTTP request into an OpenRTB request. This is guaranteed to return:
//
//   - A context which times out appropriately, given the request.
//   - A cancellation function which should be called if the auction finishes early.
//
// If the errors list is empty, then the returned request will be valid according to the OpenRTB 2.5 spec.
// In case of "strong recommendations" in the spec, it tends to be restrictive. If a better workaround is
// possible, it will return errors with messages that suggest improvements.
//
// If the errors list has at least one element, then no guarantees are made about the returned request.
func (deps *endpointDeps) parseRequest(httpRequest *http.Request) (req *openrtb_ext.RequestWrapper, impExtInfoMap map[string]exchange.ImpExtInfo, errs []error) {
	req = &openrtb_ext.RequestWrapper{}
	req.BidRequest = &openrtb2.BidRequest{}
	errs = nil

	// Pull the request body into a buffer, so we have it for later usage.
	lr := &io.LimitedReader{
		R: httpRequest.Body,
		N: deps.cfg.MaxRequestSize,
	}
	requestJson, err := ioutil.ReadAll(lr)
	if err != nil {
		errs = []error{err}
		return
	}
	// If the request size was too large, read through the rest of the request body so that the connection can be reused.
	if lr.N <= 0 {
		if written, err := io.Copy(ioutil.Discard, httpRequest.Body); written > 0 || err != nil {
			errs = []error{fmt.Errorf("Request size exceeded max size of %d bytes.", deps.cfg.MaxRequestSize)}
			return
		}
	}

	timeout := parseTimeout(requestJson, time.Duration(storedRequestTimeoutMillis)*time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Fetch the Stored Request data and merge it into the HTTP request.
	if requestJson, impExtInfoMap, errs = deps.processStoredRequests(ctx, requestJson); len(errs) > 0 {
		return
	}

	if err := json.Unmarshal(requestJson, req.BidRequest); err != nil {
		errs = []error{err}
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(httpRequest, req.BidRequest)

	if err := processInterstitials(req); err != nil {
		errs = []error{err}
		return
	}

	lmt.ModifyForIOS(req.BidRequest)

	errL := deps.validateRequest(req)
	if len(errL) > 0 {
		errs = append(errs, errL...)
	}

	return
}

// parseTimeout returns parses tmax from the requestJson, or returns the default if it doesn't exist.
//
// requestJson should be the content of the POST body.
//
// If the request defines tmax explicitly, then this will return that duration in milliseconds.
// If not, it will return the default timeout.
func parseTimeout(requestJson []byte, defaultTimeout time.Duration) time.Duration {
	if tmax, dataType, _, err := jsonparser.Get(requestJson, "tmax"); dataType != jsonparser.NotExist && err == nil {
		if tmaxInt, err := strconv.Atoi(string(tmax)); err == nil && tmaxInt > 0 {
			return time.Duration(tmaxInt) * time.Millisecond
		}
	}
	return defaultTimeout
}

func (deps *endpointDeps) validateRequest(req *openrtb_ext.RequestWrapper) []error {
	errL := []error{}
	if req.ID == "" {
		return []error{errors.New("request missing required field: \"id\"")}
	}

	if req.TMax < 0 {
		return []error{fmt.Errorf("request.tmax must be nonnegative. Got %d", req.TMax)}
	}

	if len(req.Imp) < 1 {
		return []error{errors.New("request.imp must contain at least one element.")}
	}

	if len(req.Cur) > 1 {
		req.Cur = req.Cur[0:1]
		errL = append(errL, &errortypes.Warning{Message: fmt.Sprintf("A prebid request can only process one currency. Taking the first currency in the list, %s, as the active currency", req.Cur[0])})
	}

	// If automatically filling source TID is enabled then validate that
	// source.TID exists and If it doesn't, fill it with a randomly generated UUID
	if deps.cfg.AutoGenSourceTID {
		if err := validateAndFillSourceTID(req.BidRequest); err != nil {
			return []error{err}
		}
	}

	var aliases map[string]string
	reqExt, err := req.GetRequestExt()
	if err != nil {
		return []error{fmt.Errorf("request.ext is invalid: %v", err)}
	}
	reqPrebid := reqExt.GetPrebid()
	if err := deps.parseBidExt(req); err != nil {
		return []error{err}
	} else if reqPrebid != nil {
		aliases = reqPrebid.Aliases

		if err := deps.validateAliases(aliases); err != nil {
			return []error{err}
		}

		if err := deps.validateBidAdjustmentFactors(reqPrebid.BidAdjustmentFactors, aliases); err != nil {
			return []error{err}
		}

		if err := validateSChains(reqPrebid.SChains); err != nil {
			return []error{err}
		}

		if err := deps.validateEidPermissions(reqPrebid.Data, aliases); err != nil {
			return []error{err}
		}

		if err := validateCustomRates(reqPrebid.CurrencyConversions); err != nil {
			return []error{err}
		}
	}

	if (req.Site == nil && req.App == nil) || (req.Site != nil && req.App != nil) {
		return append(errL, errors.New("request.site or request.app must be defined, but not both."))
	}

	if err := deps.validateSite(req); err != nil {
		return append(errL, err)
	}

	if err := deps.validateApp(req); err != nil {
		return append(errL, err)
	}

	if err := deps.validateUser(req, aliases); err != nil {
		return append(errL, err)
	}

	if err := validateRegs(req); err != nil {
		return append(errL, err)
	}

	if err := validateDevice(req.Device); err != nil {
		return append(errL, err)
	}

	if ccpaPolicy, err := ccpa.ReadFromRequestWrapper(req); err != nil {
		return append(errL, err)
	} else if _, err := ccpaPolicy.Parse(exchange.GetValidBidders(aliases)); err != nil {
		if _, invalidConsent := err.(*errortypes.Warning); invalidConsent {
			errL = append(errL, &errortypes.Warning{
				Message:     fmt.Sprintf("CCPA consent is invalid and will be ignored. (%v)", err),
				WarningCode: errortypes.InvalidPrivacyConsentWarningCode})
			regsExt, err := req.GetRegExt()
			if err != nil {
				return append(errL, err)
			}
			regsExt.SetUSPrivacy("")
		} else {
			return append(errL, err)
		}
	}

	impIDs := make(map[string]int, len(req.Imp))
	for index := range req.Imp {
		imp := &req.Imp[index]
		if firstIndex, ok := impIDs[imp.ID]; ok {
			errL = append(errL, fmt.Errorf(`request.imp[%d].id and request.imp[%d].id are both "%s". Imp IDs must be unique.`, firstIndex, index, imp.ID))
		}
		impIDs[imp.ID] = index
		errs := deps.validateImp(imp, aliases, index)
		if len(errs) > 0 {
			errL = append(errL, errs...)
		}
		if errortypes.ContainsFatalError(errs) {
			return errL
		}
	}

	return errL
}

func validateAndFillSourceTID(req *openrtb2.BidRequest) error {
	if req.Source == nil {
		req.Source = &openrtb2.Source{}
	}
	if req.Source.TID == "" {
		if rawUUID, err := uuid.NewV4(); err == nil {
			req.Source.TID = rawUUID.String()
		} else {
			return errors.New("req.Source.TID missing in the req and error creating a random UID")
		}
	}
	return nil
}

func (deps *endpointDeps) validateBidAdjustmentFactors(adjustmentFactors map[string]float64, aliases map[string]string) error {
	for bidderToAdjust, adjustmentFactor := range adjustmentFactors {
		if adjustmentFactor <= 0 {
			return fmt.Errorf("request.ext.prebid.bidadjustmentfactors.%s must be a positive number. Got %f", bidderToAdjust, adjustmentFactor)
		}
		if _, isBidder := deps.bidderMap[bidderToAdjust]; !isBidder {
			if _, isAlias := aliases[bidderToAdjust]; !isAlias {
				return fmt.Errorf("request.ext.prebid.bidadjustmentfactors.%s is not a known bidder or alias", bidderToAdjust)
			}
		}
	}
	return nil
}

func validateSChains(sChains []*openrtb_ext.ExtRequestPrebidSChain) error {
	_, err := exchange.BidderToPrebidSChains(sChains)
	return err
}

// validateCustomRates throws a bad input error if any of the 3-digit currency codes found in
// the bidRequest.ext.prebid.currency field is invalid, malfomed or does not represent any actual
// currency. No error is thrown if bidRequest.ext.prebid.currency is invalid or empty.
func validateCustomRates(bidReqCurrencyRates *openrtb_ext.ExtRequestCurrency) error {
	if bidReqCurrencyRates == nil {
		return nil
	}

	for fromCurrency, rates := range bidReqCurrencyRates.ConversionRates {
		// Check if fromCurrency is a valid 3-letter currency code
		if _, err := currency.ParseISO(fromCurrency); err != nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("currency code %s is not recognized or malformed", fromCurrency)}
		}

		// Check if currencies mapped to fromCurrency are valid 3-letter currency codes
		for toCurrency := range rates {
			if _, err := currency.ParseISO(toCurrency); err != nil {
				return &errortypes.BadInput{Message: fmt.Sprintf("currency code %s is not recognized or malformed", toCurrency)}
			}
		}
	}
	return nil
}

func (deps *endpointDeps) validateEidPermissions(prebid *openrtb_ext.ExtRequestPrebidData, aliases map[string]string) error {
	if prebid == nil {
		return nil
	}

	uniqueSources := make(map[string]struct{}, len(prebid.EidPermissions))
	for i, eid := range prebid.EidPermissions {
		if len(eid.Source) == 0 {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] missing required field: "source"`, i)
		}

		if _, exists := uniqueSources[eid.Source]; exists {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] duplicate entry with field: "source"`, i)
		}
		uniqueSources[eid.Source] = struct{}{}

		if len(eid.Bidders) == 0 {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] missing or empty required field: "bidders"`, i)
		}

		if err := validateBidders(eid.Bidders, deps.bidderMap, aliases); err != nil {
			return fmt.Errorf(`request.ext.prebid.data.eidpermissions[%d] contains %v`, i, err)
		}
	}

	return nil
}

func validateBidders(bidders []string, knownBidders map[string]openrtb_ext.BidderName, knownAliases map[string]string) error {
	for _, bidder := range bidders {
		if bidder == "*" {
			if len(bidders) > 1 {
				return errors.New(`bidder wildcard "*" mixed with specific bidders`)
			}
		} else {
			_, isCoreBidder := knownBidders[bidder]
			_, isAlias := knownAliases[bidder]
			if !isCoreBidder && !isAlias {
				return fmt.Errorf(`unrecognized bidder "%v"`, bidder)
			}
		}
	}
	return nil
}

func (deps *endpointDeps) validateImp(imp *openrtb2.Imp, aliases map[string]string, index int) []error {
	if imp.ID == "" {
		return []error{fmt.Errorf("request.imp[%d] missing required field: \"id\"", index)}
	}

	if len(imp.Metric) != 0 {
		return []error{fmt.Errorf("request.imp[%d].metric is not yet supported by prebid-server. Support may be added in the future", index)}
	}

	if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
		return []error{fmt.Errorf("request.imp[%d] must contain at least one of \"banner\", \"video\", \"audio\", or \"native\"", index)}
	}

	if err := validateBanner(imp.Banner, index); err != nil {
		return []error{err}
	}

	if err := validateVideo(imp.Video, index); err != nil {
		return []error{err}
	}

	if err := validateAudio(imp.Audio, index); err != nil {
		return []error{err}
	}

	if err := fillAndValidateNative(imp.Native, index); err != nil {
		return []error{err}
	}

	if err := validatePmp(imp.PMP, index); err != nil {
		return []error{err}
	}

	errL := deps.validateImpExt(imp, aliases, index)
	if len(errL) != 0 {
		return errL
	}

	return nil
}

func validateBanner(banner *openrtb2.Banner, impIndex int) error {
	if banner == nil {
		return nil
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if banner.W != nil && *banner.W < 0 {
		return fmt.Errorf("request.imp[%d].banner.w must be a positive number", impIndex)
	}
	if banner.H != nil && *banner.H < 0 {
		return fmt.Errorf("request.imp[%d].banner.h must be a positive number", impIndex)
	}

	// The following fields are deprecated in the OpenRTB 2.5 spec but are still present
	// in the OpenRTB library we use. Enforce they are not specified.
	if banner.WMin != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"wmin\". Use the \"format\" array instead.", impIndex)
	}
	if banner.WMax != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"wmax\". Use the \"format\" array instead.", impIndex)
	}
	if banner.HMin != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"hmin\". Use the \"format\" array instead.", impIndex)
	}
	if banner.HMax != 0 {
		return fmt.Errorf("request.imp[%d].banner uses unsupported property: \"hmax\". Use the \"format\" array instead.", impIndex)
	}

	hasRootSize := banner.H != nil && banner.W != nil && *banner.H > 0 && *banner.W > 0
	if !hasRootSize && len(banner.Format) == 0 {
		return fmt.Errorf("request.imp[%d].banner has no sizes. Define \"w\" and \"h\", or include \"format\" elements.", impIndex)
	}

	for i, format := range banner.Format {
		if err := validateFormat(&format, impIndex, i); err != nil {
			return err
		}
	}

	return nil
}

func validateVideo(video *openrtb2.Video, impIndex int) error {
	if video == nil {
		return nil
	}

	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].video.mimes must contain at least one supported MIME type", impIndex)
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if video.W < 0 {
		return fmt.Errorf("request.imp[%d].video.w must be a positive number", impIndex)
	}
	if video.H < 0 {
		return fmt.Errorf("request.imp[%d].video.h must be a positive number", impIndex)
	}
	if video.MinBitRate < 0 {
		return fmt.Errorf("request.imp[%d].video.minbitrate must be a positive number", impIndex)
	}
	if video.MaxBitRate < 0 {
		return fmt.Errorf("request.imp[%d].video.maxbitrate must be a positive number", impIndex)
	}

	return nil
}

func validateAudio(audio *openrtb2.Audio, impIndex int) error {
	if audio == nil {
		return nil
	}

	if len(audio.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].audio.mimes must contain at least one supported MIME type", impIndex)
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if audio.Sequence < 0 {
		return fmt.Errorf("request.imp[%d].audio.sequence must be a positive number", impIndex)
	}
	if audio.MaxSeq < 0 {
		return fmt.Errorf("request.imp[%d].audio.maxseq must be a positive number", impIndex)
	}
	if audio.MinBitrate < 0 {
		return fmt.Errorf("request.imp[%d].audio.minbitrate must be a positive number", impIndex)
	}
	if audio.MaxBitrate < 0 {
		return fmt.Errorf("request.imp[%d].audio.maxbitrate must be a positive number", impIndex)
	}

	return nil
}

// fillAndValidateNative validates the request, and assigns the Asset IDs as recommended by the Native v1.2 spec.
func fillAndValidateNative(n *openrtb2.Native, impIndex int) error {
	if n == nil {
		return nil
	}

	if len(n.Request) == 0 {
		return fmt.Errorf("request.imp[%d].native missing required property \"request\"", impIndex)
	}
	var nativePayload nativeRequests.Request
	if err := json.Unmarshal(json.RawMessage(n.Request), &nativePayload); err != nil {
		return err
	}

	if err := validateNativeContextTypes(nativePayload.Context, nativePayload.ContextSubType, impIndex); err != nil {
		return err
	}
	if err := validateNativePlacementType(nativePayload.PlcmtType, impIndex); err != nil {
		return err
	}
	if err := fillAndValidateNativeAssets(nativePayload.Assets, impIndex); err != nil {
		return err
	}
	if err := validateNativeEventTrackers(nativePayload.EventTrackers, impIndex); err != nil {
		return err
	}

	serialized, err := json.Marshal(nativePayload)
	if err != nil {
		return err
	}
	n.Request = string(serialized)
	return nil
}

func validateNativeContextTypes(cType native1.ContextType, cSubtype native1.ContextSubType, impIndex int) error {
	if cType == 0 {
		// Context is only recommended, so none is a valid type.
		return nil
	}
	if cType < native1.ContextTypeContent || (cType > native1.ContextTypeProduct && cType < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.context is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
	}
	if cSubtype < 0 {
		return fmt.Errorf("request.imp[%d].native.request.contextsubtype value can't be less than 0. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
	}
	if cSubtype == 0 {
		return nil
	}
	if cSubtype >= native1.ContextSubTypeGeneral && cSubtype <= native1.ContextSubTypeUserGenerated {
		if cType != native1.ContextTypeContent && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= native1.ContextSubTypeSocial && cSubtype <= native1.ContextSubTypeChat {
		if cType != native1.ContextTypeSocial && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= native1.ContextSubTypeSelling && cSubtype <= native1.ContextSubTypeProductReview {
		if cType != native1.ContextTypeProduct && cType < openrtb_ext.NativeExchangeSpecificLowerBound {
			return fmt.Errorf("request.imp[%d].native.request.context is %d, but contextsubtype is %d. This is an invalid combination. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex, cType, cSubtype)
		}
		return nil
	}
	if cSubtype >= openrtb_ext.NativeExchangeSpecificLowerBound {
		return nil
	}

	return fmt.Errorf("request.imp[%d].native.request.contextsubtype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=39", impIndex)
}

func validateNativePlacementType(pt native1.PlacementType, impIndex int) error {
	if pt == 0 {
		// Placement Type is only reccomended, not required.
		return nil
	}
	if pt < native1.PlacementTypeFeed || (pt > native1.PlacementTypeRecommendationWidget && pt < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.plcmttype is invalid. See https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40", impIndex)
	}
	return nil
}

func fillAndValidateNativeAssets(assets []nativeRequests.Asset, impIndex int) error {
	if len(assets) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets must be an array containing at least one object", impIndex)
	}

	assetIDs := make(map[int64]struct{}, len(assets))

	// If none of the asset IDs are defined by the caller, then prebid server should assign its own unique IDs. But
	// if the caller did assign its own asset IDs, then prebid server will respect those IDs
	assignAssetIDs := true
	for i := 0; i < len(assets); i++ {
		assignAssetIDs = assignAssetIDs && (assets[i].ID == 0)
	}

	for i := 0; i < len(assets); i++ {
		if err := validateNativeAsset(assets[i], impIndex, i); err != nil {
			return err
		}

		if assignAssetIDs {
			assets[i].ID = int64(i)
			continue
		}

		// Each asset should have a unique ID thats assigned by the caller
		if _, ok := assetIDs[assets[i].ID]; ok {
			return fmt.Errorf("request.imp[%d].native.request.assets[%d].id is already being used by another asset. Each asset ID must be unique.", impIndex, i)
		}

		assetIDs[assets[i].ID] = struct{}{}
	}

	return nil
}

func validateNativeAsset(asset nativeRequests.Asset, impIndex int, assetIndex int) error {
	assetErr := "request.imp[%d].native.request.assets[%d] must define exactly one of {title, img, video, data}"
	foundType := false

	if asset.Title != nil {
		foundType = true
		if err := validateNativeAssetTitle(asset.Title, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Img != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetImage(asset.Img, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Video != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetVideo(asset.Video, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Data != nil {
		if foundType {
			return fmt.Errorf(assetErr, impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetData(asset.Data, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if !foundType {
		return fmt.Errorf(assetErr, impIndex, assetIndex)
	}

	return nil
}

func validateNativeEventTrackers(trackers []nativeRequests.EventTracker, impIndex int) error {
	for i := 0; i < len(trackers); i++ {
		if err := validateNativeEventTracker(trackers[i], impIndex, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeAssetTitle(title *nativeRequests.Title, impIndex int, assetIndex int) error {
	if title.Len < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].title.len must be a positive number", impIndex, assetIndex)
	}
	return nil
}

func validateNativeEventTracker(tracker nativeRequests.EventTracker, impIndex int, eventIndex int) error {
	if tracker.Event < native1.EventTypeImpression || (tracker.Event > native1.EventTypeViewableVideo50 && tracker.Event < openrtb_ext.NativeExchangeSpecificLowerBound) {
		return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].event is invalid. See section 7.6: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex)
	}
	if len(tracker.Methods) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].method is required. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex)
	}
	for methodIndex, method := range tracker.Methods {
		if method < native1.EventTrackingMethodImage || (method > native1.EventTrackingMethodJS && method < openrtb_ext.NativeExchangeSpecificLowerBound) {
			return fmt.Errorf("request.imp[%d].native.request.eventtrackers[%d].methods[%d] is invalid. See section 7.7: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=43", impIndex, eventIndex, methodIndex)
		}
	}

	return nil
}

func validateNativeAssetImage(img *nativeRequests.Image, impIndex int, assetIndex int) error {
	if img.W < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.w must be a positive integer", impIndex, assetIndex)
	}
	if img.H < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.h must be a positive integer", impIndex, assetIndex)
	}
	if img.WMin < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.wmin must be a positive integer", impIndex, assetIndex)
	}
	if img.HMin < 0 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].img.hmin must be a positive integer", impIndex, assetIndex)
	}
	return nil
}

func validateNativeAssetVideo(video *nativeRequests.Video, impIndex int, assetIndex int) error {
	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.mimes must be an array with at least one MIME type", impIndex, assetIndex)
	}
	if video.MinDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.minduration must be a positive integer", impIndex, assetIndex)
	}
	if video.MaxDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.maxduration must be a positive integer", impIndex, assetIndex)
	}
	if err := validateNativeVideoProtocols(video.Protocols, impIndex, assetIndex); err != nil {
		return err
	}

	return nil
}

func validateNativeAssetData(data *nativeRequests.Data, impIndex int, assetIndex int) error {
	if data.Type < native1.DataAssetTypeSponsored || (data.Type > native1.DataAssetTypeCTAText && data.Type < 500) {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].data.type is invalid. See section 7.4: https://iabtechlab.com/wp-content/uploads/2016/07/OpenRTB-Native-Ads-Specification-Final-1.2.pdf#page=40", impIndex, assetIndex)
	}

	return nil
}

func validateNativeVideoProtocols(protocols []native1.Protocol, impIndex int, assetIndex int) error {
	if len(protocols) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols must be an array with at least one element", impIndex, assetIndex)
	}
	for i := 0; i < len(protocols); i++ {
		if err := validateNativeVideoProtocol(protocols[i], impIndex, assetIndex, i); err != nil {
			return err
		}
	}
	return nil
}

func validateNativeVideoProtocol(protocol native1.Protocol, impIndex int, assetIndex int, protocolIndex int) error {
	if protocol < native1.ProtocolVAST10 || protocol > native1.ProtocolDAAST10Wrapper {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols[%d] is invalid. See Section 5.8: https://www.iab.com/wp-content/uploads/2016/03/OpenRTB-API-Specification-Version-2-5-FINAL.pdf#page=52", impIndex, assetIndex, protocolIndex)
	}
	return nil
}

func validateFormat(format *openrtb2.Format, impIndex, formatIndex int) error {
	usesHW := format.W != 0 || format.H != 0
	usesRatios := format.WMin != 0 || format.WRatio != 0 || format.HRatio != 0

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if format.W < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].w must be a positive number", impIndex, formatIndex)
	}
	if format.H < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].h must be a positive number", impIndex, formatIndex)
	}
	if format.WRatio < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].wratio must be a positive number", impIndex, formatIndex)
	}
	if format.HRatio < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].hratio must be a positive number", impIndex, formatIndex)
	}
	if format.WMin < 0 {
		return fmt.Errorf("request.imp[%d].banner.format[%d].wmin must be a positive number", impIndex, formatIndex)
	}

	if usesHW && usesRatios {
		return fmt.Errorf("Request imp[%d].banner.format[%d] should define *either* {w, h} *or* {wmin, wratio, hratio}, but not both. If both are valid, send two \"format\" objects in the request.", impIndex, formatIndex)
	}
	if !usesHW && !usesRatios {
		return fmt.Errorf("Request imp[%d].banner.format[%d] should define *either* {w, h} (for static size requirements) *or* {wmin, wratio, hratio} (for flexible sizes) to be non-zero.", impIndex, formatIndex)
	}
	if usesHW && (format.W == 0 || format.H == 0) {
		return fmt.Errorf("Request imp[%d].banner.format[%d] must define non-zero \"h\" and \"w\" properties.", impIndex, formatIndex)
	}
	if usesRatios && (format.WMin == 0 || format.WRatio == 0 || format.HRatio == 0) {
		return fmt.Errorf("Request imp[%d].banner.format[%d] must define non-zero \"wmin\", \"wratio\", and \"hratio\" properties.", impIndex, formatIndex)
	}
	return nil
}

func validatePmp(pmp *openrtb2.PMP, impIndex int) error {
	if pmp == nil {
		return nil
	}

	for dealIndex, deal := range pmp.Deals {
		if deal.ID == "" {
			return fmt.Errorf("request.imp[%d].pmp.deals[%d] missing required field: \"id\"", impIndex, dealIndex)
		}
	}
	return nil
}

func (deps *endpointDeps) validateImpExt(imp *openrtb2.Imp, aliases map[string]string, impIndex int) []error {
	errL := []error{}
	if len(imp.Ext) == 0 {
		return []error{fmt.Errorf("request.imp[%d].ext is required", impIndex)}
	}

	var bidderExts map[string]json.RawMessage
	if err := json.Unmarshal(imp.Ext, &bidderExts); err != nil {
		return []error{err}
	}

	// Prefer bidder params from request.imp.ext.prebid.bidder.BIDDER over request.imp.ext.BIDDER
	// to avoid confusion beteween prebid specific adapter config and other ext protocols.
	if extPrebidJSON, ok := bidderExts[openrtb_ext.PrebidExtKey]; ok {
		var extPrebid openrtb_ext.ExtImpPrebid
		if err := json.Unmarshal(extPrebidJSON, &extPrebid); err == nil && extPrebid.Bidder != nil {
			for bidder, ext := range extPrebid.Bidder {
				if ext == nil {
					continue
				}
				bidderExts[bidder] = ext
			}
		}
	}

	/* Process all the bidder exts in the request */
	disabledBidders := []string{}
	otherExtElements := 0
	for bidder, ext := range bidderExts {
		if isBidderToValidate(bidder) {
			coreBidder := bidder
			if tmp, isAlias := aliases[bidder]; isAlias {
				coreBidder = tmp
			}
			if bidderName, isValid := deps.bidderMap[coreBidder]; isValid {
				if err := deps.paramsValidator.Validate(bidderName, ext); err != nil {
					return []error{fmt.Errorf("request.imp[%d].ext.%s failed validation.\n%v", impIndex, coreBidder, err)}
				}
			} else {
				if msg, isDisabled := deps.disabledBidders[bidder]; isDisabled {
					errL = append(errL, &errortypes.BidderTemporarilyDisabled{Message: msg})
					disabledBidders = append(disabledBidders, bidder)
				} else {
					return []error{fmt.Errorf("request.imp[%d].ext contains unknown bidder: %s. Did you forget an alias in request.ext.prebid.aliases?", impIndex, bidder)}
				}
			}
		} else {
			otherExtElements++
		}
	}

	// defer deleting disabled bidders so we don't disrupt the loop
	if len(disabledBidders) > 0 {
		for _, bidder := range disabledBidders {
			delete(bidderExts, bidder)
		}
		extJSON, err := json.Marshal(bidderExts)
		if err != nil {
			return []error{err}
		}
		imp.Ext = extJSON
	}

	if len(bidderExts)-otherExtElements == 0 {
		errL = append(errL, fmt.Errorf("request.imp[%d].ext must contain at least one bidder", impIndex))
	}

	return errL
}

// isBidderToValidate determines if the bidder name in request.imp[].prebid should be validated.
func isBidderToValidate(bidder string) bool {
	switch openrtb_ext.BidderName(bidder) {
	case openrtb_ext.BidderReservedContext:
		return false
	case openrtb_ext.BidderReservedData:
		return false
	case openrtb_ext.BidderReservedPrebid:
		return false
	case openrtb_ext.BidderReservedSKAdN:
		return false
	default:
		return true
	}
}

func (deps *endpointDeps) parseBidExt(req *openrtb_ext.RequestWrapper) error {
	if _, err := req.GetRequestExt(); err != nil {
		return fmt.Errorf("request.ext is invalid: %v", err)
	}
	return nil
}

func (deps *endpointDeps) validateAliases(aliases map[string]string) error {
	for alias, coreBidder := range aliases {
		if _, isCoreBidderDisabled := deps.disabledBidders[coreBidder]; isCoreBidderDisabled {
			return fmt.Errorf("request.ext.prebid.aliases.%s refers to disabled bidder: %s", alias, coreBidder)
		}

		if _, isCoreBidder := deps.bidderMap[coreBidder]; !isCoreBidder {
			return fmt.Errorf("request.ext.prebid.aliases.%s refers to unknown bidder: %s", alias, coreBidder)
		}

		if alias == coreBidder {
			return fmt.Errorf("request.ext.prebid.aliases.%s defines a no-op alias. Choose a different alias, or remove this entry.", alias)
		}
	}
	return nil
}

func (deps *endpointDeps) validateSite(req *openrtb_ext.RequestWrapper) error {
	if req.Site == nil {
		return nil
	}

	if req.Site.ID == "" && req.Site.Page == "" {
		return errors.New("request.site should include at least one of request.site.id or request.site.page.")
	}
	siteExt, err := req.GetSiteExt()
	if err != nil {
		return err
	}
	siteAmp := siteExt.GetAmp()
	if siteAmp < 0 || siteAmp > 1 {
		return errors.New(`request.site.ext.amp must be either 1, 0, or undefined`)
	}

	return nil
}

func (deps *endpointDeps) validateApp(req *openrtb_ext.RequestWrapper) error {
	if req.App == nil {
		return nil
	}

	if req.App.ID != "" {
		if _, found := deps.cfg.BlacklistedAppMap[req.App.ID]; found {
			return &errortypes.BlacklistedApp{Message: fmt.Sprintf("Prebid-server does not process requests from App ID: %s", req.App.ID)}
		}
	}

	_, err := req.GetAppExt()
	return err
}

func (deps *endpointDeps) validateUser(req *openrtb_ext.RequestWrapper, aliases map[string]string) error {
	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if req != nil && req.BidRequest != nil && req.User != nil {
		if req.User.Geo != nil && req.User.Geo.Accuracy < 0 {
			return errors.New("request.user.geo.accuracy must be a positive number")
		}
	}

	userExt, err := req.GetUserExt()
	if err != nil {
		return fmt.Errorf("request.user.ext object is not valid: %v", err)
	}
	// Check if the buyeruids are valid
	prebid := userExt.GetPrebid()
	if prebid != nil {
		if len(prebid.BuyerUIDs) < 1 {
			return errors.New(`request.user.ext.prebid requires a "buyeruids" property with at least one ID defined. If none exist, then request.user.ext.prebid should not be defined.`)
		}
		for bidderName := range prebid.BuyerUIDs {
			if _, ok := deps.bidderMap[bidderName]; !ok {
				if _, ok := aliases[bidderName]; !ok {
					return fmt.Errorf("request.user.ext.%s is neither a known bidder name nor an alias in request.ext.prebid.aliases.", bidderName)
				}
			}
		}
	}
	// Check Universal User ID
	eids := userExt.GetEid()
	if eids != nil {
		if len(*eids) == 0 {
			return errors.New("request.user.ext.eids must contain at least one element or be undefined")
		}
		uniqueSources := make(map[string]struct{}, len(*eids))
		for eidIndex, eid := range *eids {
			if eid.Source == "" {
				return fmt.Errorf("request.user.ext.eids[%d] missing required field: \"source\"", eidIndex)
			}
			if _, ok := uniqueSources[eid.Source]; ok {
				return errors.New("request.user.ext.eids must contain unique sources")
			}
			uniqueSources[eid.Source] = struct{}{}

			if eid.ID == "" && eid.Uids == nil {
				return fmt.Errorf("request.user.ext.eids[%d] must contain either \"id\" or \"uids\" field", eidIndex)
			}
			if eid.ID == "" {
				if len(eid.Uids) == 0 {
					return fmt.Errorf("request.user.ext.eids[%d].uids must contain at least one element or be undefined", eidIndex)
				}
				for uidIndex, uid := range eid.Uids {
					if uid.ID == "" {
						return fmt.Errorf("request.user.ext.eids[%d].uids[%d] missing required field: \"id\"", eidIndex, uidIndex)
					}
				}
			}
		}
	}

	return nil
}

func validateRegs(req *openrtb_ext.RequestWrapper) error {
	regsExt, err := req.GetRegExt()
	if err != nil {
		return fmt.Errorf("request.regs.ext is invalid: %v", err)
	}
	regExt := regsExt.GetExt()
	gdprJSON, hasGDPR := regExt["gdpr"]
	if hasGDPR && (string(gdprJSON) != "0" && string(gdprJSON) != "1") {
		return errors.New("request.regs.ext.gdpr must be either 0 or 1.")
	}
	return nil
}

func validateDevice(device *openrtb2.Device) error {
	if device == nil {
		return nil
	}

	// The following fields were previously uints in the OpenRTB library we use, but have
	// since been changed to ints. We decided to maintain the non-negative check.
	if device.W < 0 {
		return errors.New("request.device.w must be a positive number")
	}
	if device.H < 0 {
		return errors.New("request.device.h must be a positive number")
	}
	if device.PPI < 0 {
		return errors.New("request.device.ppi must be a positive number")
	}
	if device.Geo != nil && device.Geo.Accuracy < 0 {
		return errors.New("request.device.geo.accuracy must be a positive number")
	}

	return nil
}

func sanitizeRequest(r *openrtb2.BidRequest, ipValidator iputil.IPValidator) {
	if r.Device != nil {
		if ip, ver := iputil.ParseIP(r.Device.IP); ip == nil || ver != iputil.IPv4 || !ipValidator.IsValid(ip, ver) {
			r.Device.IP = ""
		}

		if ip, ver := iputil.ParseIP(r.Device.IPv6); ip == nil || ver != iputil.IPv6 || !ipValidator.IsValid(ip, ver) {
			r.Device.IPv6 = ""
		}
	}
}

// setFieldsImplicitly uses _implicit_ information from the httpReq to set values on bidReq.
// This function does not consume the request body, which was set explicitly, but infers certain
// OpenRTB properties from the headers and other implicit info.
//
// This function _should not_ override any fields which were defined explicitly by the caller in the request.
func (deps *endpointDeps) setFieldsImplicitly(httpReq *http.Request, bidReq *openrtb2.BidRequest) {
	sanitizeRequest(bidReq, deps.privateNetworkIPValidator)

	setDeviceImplicitly(httpReq, bidReq, deps.privateNetworkIPValidator)

	// Per the OpenRTB spec: A bid request must not contain both a Site and an App object.
	if bidReq.App == nil {
		setSiteImplicitly(httpReq, bidReq)
	}
	setImpsImplicitly(httpReq, bidReq.Imp)

	setAuctionTypeImplicitly(bidReq)
}

// setDeviceImplicitly uses implicit info from httpReq to populate bidReq.Device
func setDeviceImplicitly(httpReq *http.Request, bidReq *openrtb2.BidRequest, ipValidtor iputil.IPValidator) {
	setIPImplicitly(httpReq, bidReq, ipValidtor)
	setUAImplicitly(httpReq, bidReq)
	setDoNotTrackImplicitly(httpReq, bidReq)

}

// setAuctionTypeImplicitly sets the auction type to 1 if it wasn't on the request,
// since header bidding is generally a first-price auction.
func setAuctionTypeImplicitly(bidReq *openrtb2.BidRequest) {
	if bidReq.AT == 0 {
		bidReq.AT = 1
	}
	return
}

// setSiteImplicitly uses implicit info from httpReq to populate bidReq.Site
func setSiteImplicitly(httpReq *http.Request, bidReq *openrtb2.BidRequest) {
	if bidReq.Site == nil || bidReq.Site.Page == "" || bidReq.Site.Domain == "" {
		referrerCandidate := httpReq.Referer()
		if parsedUrl, err := url.Parse(referrerCandidate); err == nil {
			if domain, err := publicsuffix.EffectiveTLDPlusOne(parsedUrl.Host); err == nil {
				if bidReq.Site == nil {
					bidReq.Site = &openrtb2.Site{}
				}
				if bidReq.Site.Domain == "" {
					bidReq.Site.Domain = domain
				}

				// This looks weird... but is not a bug. The site which called prebid-server (the "referer"), is
				// (almost certainly) the page where the ad will be hosted. In the OpenRTB spec, this is *page*, not *ref*.
				if bidReq.Site.Page == "" {
					bidReq.Site.Page = referrerCandidate
				}
			}
		}
	}
	if bidReq.Site != nil {
		setAmpExt(bidReq.Site, "0")
	}
}

func setImpsImplicitly(httpReq *http.Request, imps []openrtb2.Imp) {
	secure := int8(1)
	for i := 0; i < len(imps); i++ {
		if imps[i].Secure == nil && httputil.IsSecure(httpReq) {
			imps[i].Secure = &secure
		}
	}
}

func getJsonSyntaxError(testJSON []byte) (bool, string) {
	type JsonNode struct {
		raw   *json.RawMessage
		doc   map[string]*JsonNode
		ary   []*JsonNode
		which int
	}
	type jNode map[string]*JsonNode
	docErrdoc := &jNode{}
	docErr := json.Unmarshal(testJSON, docErrdoc)
	if uerror, ok := docErr.(*json.SyntaxError); ok {
		err := fmt.Sprintf("%s at offset %v", uerror.Error(), uerror.Offset)
		return true, err
	}
	return false, ""
}

func (deps *endpointDeps) processStoredRequests(ctx context.Context, requestJson []byte) ([]byte, map[string]exchange.ImpExtInfo, []error) {
	// Parse the Stored Request IDs from the BidRequest and Imps.
	storedBidRequestId, hasStoredBidRequest, err := getStoredRequestId(requestJson)
	if err != nil {
		return nil, nil, []error{err}
	}
	imps, impIds, idIndices, errs := parseImpInfo(requestJson)
	if len(errs) > 0 {
		return nil, nil, errs
	}

	// Fetch the Stored Request data
	var storedReqIds []string
	if hasStoredBidRequest {
		storedReqIds = []string{storedBidRequestId}
	}
	storedRequests, storedImps, errs := deps.storedReqFetcher.FetchRequests(ctx, storedReqIds, impIds)
	if len(errs) != 0 {
		return nil, nil, errs
	}

	// Apply the Stored BidRequest, if it exists
	resolvedRequest := requestJson
	if hasStoredBidRequest {
		resolvedRequest, err = jsonpatch.MergePatch(storedRequests[storedBidRequestId], requestJson)
		if err != nil {
			hasErr, Err := getJsonSyntaxError(requestJson)
			if hasErr {
				err = fmt.Errorf("Invalid JSON in Incoming Request: %s", Err)
			} else {
				hasErr, Err = getJsonSyntaxError(storedRequests[storedBidRequestId])
				if hasErr {
					err = fmt.Errorf("Invalid JSON in Stored Request with ID %s: %s", storedBidRequestId, Err)
					err = fmt.Errorf("ext.prebid.storedrequest.id refers to Stored Request %s which contains Invalid JSON: %s", storedBidRequestId, Err)
				}
			}
			return nil, nil, []error{err}
		}
	}

	// Apply default aliases, if they are provided
	if deps.defaultRequest {
		aliasedRequest, err := jsonpatch.MergePatch(deps.defReqJSON, resolvedRequest)
		if err != nil {
			hasErr, Err := getJsonSyntaxError(resolvedRequest)
			if hasErr {
				err = fmt.Errorf("Invalid JSON in Incoming Request: %s", Err)
			} else {
				hasErr, Err = getJsonSyntaxError(deps.defReqJSON)
				if hasErr {
					err = fmt.Errorf("Invalid JSON in Default Request Settings: %s", Err)
				}
			}
			return nil, nil, []error{err}
		}
		resolvedRequest = aliasedRequest
	}

	// Apply any Stored Imps, if they exist. Since the JSON Merge Patch overrides arrays,
	// and Prebid Server defers to the HTTP Request to resolve conflicts, it's safe to
	// assume that the request.imp data did not change when applying the Stored BidRequest.
	impExtInfoMap := make(map[string]exchange.ImpExtInfo, len(impIds))
	for i := 0; i < len(impIds); i++ {
		resolvedImp, err := jsonpatch.MergePatch(storedImps[impIds[i]], imps[idIndices[i]])

		if err != nil {
			hasErr, Err := getJsonSyntaxError(imps[idIndices[i]])
			if hasErr {
				err = fmt.Errorf("Invalid JSON in Imp[%d] of Incoming Request: %s", i, Err)
			} else {
				hasErr, Err = getJsonSyntaxError(storedImps[impIds[i]])
				if hasErr {
					err = fmt.Errorf("imp.ext.prebid.storedrequest.id %s: Stored Imp has Invalid JSON: %s", impIds[i], Err)
				}
			}
			return nil, nil, []error{err}
		}
		imps[idIndices[i]] = resolvedImp

		impId, err := jsonparser.GetString(resolvedImp, "id")
		if err != nil {
			return nil, nil, []error{err}
		}
		// This is substantially faster for reading values from a json blob than the Go json package,
		// but keep in mind that each jsonparser.GetXXX call re-parses the entire json.
		// Based on performance measurements, the tipping point of efficiency is around 4 calls.
		// At that point, please consider switching to EachKey to use a single pass.
		includeVideoAttributes, err := jsonparser.GetBoolean(resolvedImp, "ext", "prebid", "options", "echovideoattrs")
		if err != nil && err != jsonparser.KeyPathNotFoundError {
			return nil, nil, []error{err}
		}
		if storedImps[impIds[i]] != nil {
			impExtInfoMap[impId] = exchange.ImpExtInfo{EchoVideoAttrs: includeVideoAttributes, StoredImp: storedImps[impIds[i]]}
		}
	}
	if len(impIds) > 0 {
		newImpJson, err := json.Marshal(imps)
		if err != nil {
			return nil, nil, []error{err}
		}
		resolvedRequest, err = jsonparser.Set(resolvedRequest, newImpJson, "imp")
		if err != nil {
			return nil, nil, []error{err}
		}
	}

	return resolvedRequest, impExtInfoMap, nil
}

// parseImpInfo parses the request JSON and returns several things about the Imps
//
// 1. A list of the JSON for every Imp.
// 2. A list of all IDs which appear at `imp[i].ext.prebid.storedrequest.id`.
// 3. A list intended to parallel "ids". Each element tells which index of "imp[index]" the corresponding element of "ids" should modify.
// 4. Any errors which occur due to bad requests. These should warrant an HTTP 4xx response.
func parseImpInfo(requestJson []byte) (imps []json.RawMessage, ids []string, impIdIndices []int, errs []error) {
	if impArray, dataType, _, err := jsonparser.Get(requestJson, "imp"); err == nil && dataType == jsonparser.Array {
		i := 0
		jsonparser.ArrayEach(impArray, func(imp []byte, _ jsonparser.ValueType, _ int, err error) {
			if storedImpId, hasStoredImp, err := getStoredRequestId(imp); err != nil {
				errs = append(errs, err)
			} else if hasStoredImp {
				ids = append(ids, storedImpId)
				impIdIndices = append(impIdIndices, i)
			}
			imps = append(imps, imp)
			i++
		})
	}
	return
}

// getStoredRequestId parses a Stored Request ID from some json, without doing a full (slow) unmarshal.
// It returns the ID, true/false whether a stored request key existed, and an error if anything went wrong
// (e.g. malformed json, id not a string, etc).
func getStoredRequestId(data []byte) (string, bool, error) {
	// These keys must be kept in sync with openrtb_ext.ExtStoredRequest
	value, dataType, _, err := jsonparser.Get(data, "ext", openrtb_ext.PrebidExtKey, "storedrequest", "id")
	if dataType == jsonparser.NotExist {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if dataType != jsonparser.String {
		return "", true, errors.New("ext.prebid.storedrequest.id must be a string")
	}

	return string(value), true, nil
}

// setIPImplicitly sets the IP address on bidReq, if it's not explicitly defined and we can figure it out.
func setIPImplicitly(httpReq *http.Request, bidReq *openrtb2.BidRequest, ipValidator iputil.IPValidator) {
	if bidReq.Device == nil || (bidReq.Device.IP == "" && bidReq.Device.IPv6 == "") {
		if ip, ver := httputil.FindIP(httpReq, ipValidator); ip != nil {
			switch ver {
			case iputil.IPv4:
				if bidReq.Device == nil {
					bidReq.Device = &openrtb2.Device{}
				}
				bidReq.Device.IP = ip.String()
			case iputil.IPv6:
				if bidReq.Device == nil {
					bidReq.Device = &openrtb2.Device{}
				}
				bidReq.Device.IPv6 = ip.String()
			}
		}
	}
}

// setUAImplicitly sets the User Agent on bidReq, if it's not explicitly defined and it's defined on the request.
func setUAImplicitly(httpReq *http.Request, bidReq *openrtb2.BidRequest) {
	if bidReq.Device == nil || bidReq.Device.UA == "" {
		if ua := httpReq.UserAgent(); ua != "" {
			if bidReq.Device == nil {
				bidReq.Device = &openrtb2.Device{}
			}
			bidReq.Device.UA = ua
		}
	}
}

func setDoNotTrackImplicitly(httpReq *http.Request, bidReq *openrtb2.BidRequest) {
	if bidReq.Device == nil || bidReq.Device.DNT == nil {
		dnt := httpReq.Header.Get(dntKey)
		if dnt == "0" || dnt == "1" {
			if bidReq.Device == nil {
				bidReq.Device = &openrtb2.Device{}
			}

			switch dnt {
			case "0":
				bidReq.Device.DNT = &dntDisabled
			case "1":
				bidReq.Device.DNT = &dntEnabled
			}
		}
	}
}

// parseUserID gets this user's ID for the host machine, if it exists.
func parseUserID(cfg *config.Configuration, httpReq *http.Request) (string, bool) {
	if hostCookie, err := httpReq.Cookie(cfg.HostCookie.CookieName); hostCookie != nil && err == nil {
		return hostCookie.Value, true
	} else {
		return "", false
	}
}

// Write(return) errors to the client, if any. Returns true if errors were found.
func writeError(errs []error, w http.ResponseWriter, labels *metrics.Labels) bool {
	var rc bool = false
	if len(errs) > 0 {
		httpStatus := http.StatusBadRequest
		metricsStatus := metrics.RequestStatusBadInput
		for _, err := range errs {
			erVal := errortypes.ReadCode(err)
			if erVal == errortypes.BlacklistedAppErrorCode || erVal == errortypes.BlacklistedAcctErrorCode {
				httpStatus = http.StatusServiceUnavailable
				metricsStatus = metrics.RequestStatusBlacklisted
				break
			}
		}
		w.WriteHeader(httpStatus)
		labels.RequestStatus = metricsStatus
		for _, err := range errs {
			w.Write([]byte(fmt.Sprintf("Invalid request: %s\n", err.Error())))
		}
		rc = true
	}
	return rc
}

// Returns the account ID for the request
func getAccountID(pub *openrtb2.Publisher) string {
	if pub != nil {
		if pub.Ext != nil {
			var pubExt openrtb_ext.ExtPublisher
			err := json.Unmarshal(pub.Ext, &pubExt)
			if err == nil && pubExt.Prebid != nil && pubExt.Prebid.ParentAccount != nil && *pubExt.Prebid.ParentAccount != "" {
				return *pubExt.Prebid.ParentAccount
			}
		}
		if pub.ID != "" {
			return pub.ID
		}
	}
	return metrics.PublisherUnknown
}
