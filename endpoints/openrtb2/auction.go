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
	"strconv"
	"time"

	"github.com/buger/jsonparser"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"github.com/mxmCherry/openrtb"
	nativeRequests "github.com/mxmCherry/openrtb/native/request"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/prebid"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/rcrowley/go-metrics"
	"golang.org/x/net/publicsuffix"
)

const defaultRequestTimeoutMillis = 5000
const storedRequestTimeoutMillis = 50

func NewEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsById stored_requests.Fetcher, cfg *config.Configuration, met *pbsmetrics.Metrics) (httprouter.Handle, error) {
	if ex == nil || validator == nil || requestsById == nil || cfg == nil || met == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}

	return httprouter.Handle((&endpointDeps{ex, validator, requestsById, cfg, met}).Auction), nil
}

type endpointDeps struct {
	ex               exchange.Exchange
	paramsValidator  openrtb_ext.BidderParamValidator
	storedReqFetcher stored_requests.Fetcher
	cfg              *config.Configuration
	metrics          *pbsmetrics.Metrics
}

func (deps *endpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// Prebid Server interprets request.tmax to be the maximum amount of time that a caller is willing
	// to wait for bids. However, tmax may be defined in the Stored Request data.
	//
	// If so, then the trip to the backend might use a significant amount of this time.
	// We can respect timeouts more accurately if we note the *real* start time, and use it
	// to compute the auction timeout.
	start := time.Now()
	deps.metrics.RequestMeter.Mark(1)
	deps.metrics.ORTBRequestMeter.Mark(1)

	isSafari := checkSafari(r, deps.metrics.SafariRequestMeter)

	req, errL := deps.parseRequest(r)

	if writeError(errL, deps.metrics.ErrorMeter, w) {
		return
	}

	ctx := context.Background()
	cancel := func() {}
	if req.TMax > 0 {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(req.TMax)*time.Millisecond))
	} else {
		ctx, cancel = context.WithDeadline(ctx, start.Add(time.Duration(defaultRequestTimeoutMillis)*time.Millisecond))
	}
	defer cancel()

	usersyncs := pbs.ParsePBSCookieFromRequest(r, &(deps.cfg.HostCookie.OptOutCookie))
	if req.App != nil {
		deps.metrics.AppRequestMeter.Mark(1)
	} else if usersyncs.LiveSyncCount() == 0 {
		deps.metrics.NoCookieMeter.Mark(1)
		if isSafari {
			deps.metrics.SafariNoCookieMeter.Mark(1)
		}
	}

	response, err := deps.ex.HoldAuction(ctx, req, usersyncs)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		glog.Errorf("/openrtb2/auction Critical error: %v", err)
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
		glog.Errorf("/openrtb2/auction Error encoding response: %v", err)
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
func (deps *endpointDeps) parseRequest(httpRequest *http.Request) (req *openrtb.BidRequest, errs []error) {
	req = &openrtb.BidRequest{}
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
	if requestJson, errs = deps.processStoredRequests(ctx, requestJson); len(errs) > 0 {
		return
	}

	if err := json.Unmarshal(requestJson, req); err != nil {
		errs = []error{err}
		return
	}

	// Populate any "missing" OpenRTB fields with info from other sources, (e.g. HTTP request headers).
	deps.setFieldsImplicitly(httpRequest, req)

	if err := deps.validateRequest(req); err != nil {
		errs = []error{err}
		return
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

func (deps *endpointDeps) validateRequest(req *openrtb.BidRequest) error {
	if req.ID == "" {
		return errors.New("request missing required field: \"id\"")
	}

	if req.TMax < 0 {
		return fmt.Errorf("request.tmax must be nonnegative. Got %d", req.TMax)
	}

	if len(req.Imp) < 1 {
		return errors.New("request.imp must contain at least one element.")
	}

	var aliases map[string]string
	if bidExt, err := deps.parseBidExt(req.Ext); err != nil {
		return err
	} else if bidExt != nil {
		aliases = bidExt.Prebid.Aliases
	}

	if err := deps.validateAliases(aliases); err != nil {
		return err
	}

	for index, imp := range req.Imp {
		if err := deps.validateImp(&imp, aliases, index); err != nil {
			return err
		}
	}

	if (req.Site == nil && req.App == nil) || (req.Site != nil && req.App != nil) {
		return errors.New("request.site or request.app must be defined, but not both.")
	}

	if err := deps.validateSite(req.Site); err != nil {
		return err
	}

	if err := validateUser(req.User, aliases); err != nil {
		return err
	}

	if err := validateRegs(req.Regs); err != nil {
		return err
	}

	return nil
}

func (deps *endpointDeps) validateImp(imp *openrtb.Imp, aliases map[string]string, index int) error {
	if imp.ID == "" {
		return fmt.Errorf("request.imp[%d] missing required field: \"id\"", index)
	}

	if len(imp.Metric) != 0 {
		return errors.New("request.imp[%d].metric is not yet supported by prebid-server. Support may be added in the future.")
	}

	if imp.Banner == nil && imp.Video == nil && imp.Audio == nil && imp.Native == nil {
		return errors.New("request.imp[%d] must contain at least one of \"banner\", \"video\", \"audio\", or \"native\"")
	}

	if err := validateBanner(imp.Banner, index); err != nil {
		return err
	}

	if imp.Video != nil {
		if len(imp.Video.MIMEs) < 1 {
			return fmt.Errorf("request.imp[%d].video.mimes must contain at least one supported MIME type", index)
		}
	}

	if imp.Audio != nil {
		if len(imp.Audio.MIMEs) < 1 {
			return fmt.Errorf("request.imp[%d].audio.mimes must contain at least one supported MIME type", index)
		}
	}

	if err := fillAndValidateNative(imp.Native, index); err != nil {
		return err
	}

	if err := validatePmp(imp.PMP, index); err != nil {
		return err
	}

	if err := deps.validateImpExt(imp.Ext, aliases, index); err != nil {
		return err
	}

	return nil
}

func validateBanner(banner *openrtb.Banner, impIndex int) error {
	if banner == nil {
		return nil
	}

	// Although these are only deprecated in the spec... since this is a new endpoint, we know nobody uses them yet.
	// Let's start things off by pointing callers in the right direction.
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

	for fmtIndex, format := range banner.Format {
		if err := validateFormat(&format, impIndex, fmtIndex); err != nil {
			return err
		}
	}
	return nil
}

// fillAndValidateNative validates the request, and assigns the Asset IDs as recommended by the Native v1.2 spec.
func fillAndValidateNative(n *openrtb.Native, impIndex int) error {
	if n == nil {
		return nil
	}

	var nativePayload nativeRequests.Request
	if err := json.Unmarshal(json.RawMessage(n.Request), &nativePayload); err != nil {
		return err
	}

	if err := validateNativeContext(nativePayload.Context, impIndex); err != nil {
		return err
	}
	if err := validateNativePlacementType(nativePayload.PlcmtType, impIndex); err != nil {
		return err
	}
	if err := fillAndValidateNativeAssets(nativePayload.Assets, impIndex); err != nil {
		return err
	}

	// TODO #218: Validate eventtrackers once mxmcherry/openrtb has been updated to support Native v1.2

	serialized, err := json.Marshal(nativePayload)
	if err != nil {
		return err
	}
	n.Request = string(serialized)
	return nil
}

func validateNativeContext(c nativeRequests.ContextType, impIndex int) error {
	if c < 1 || c > 3 {
		return fmt.Errorf("request.imp[%d].native.request.context must be in the range [1, 3]. Got %d", impIndex, c)
	}
	return nil
}

func validateNativePlacementType(pt nativeRequests.PlacementType, impIndex int) error {
	if pt < 1 || pt > 4 {
		return fmt.Errorf("request.imp[%d].native.request.plcmttype must be in the range [1, 4]. Got %d", impIndex, pt)
	}
	return nil
}

func fillAndValidateNativeAssets(assets []nativeRequests.Asset, impIndex int) error {
	if len(assets) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets must be an array containing at least one object.", impIndex)
	}

	for i := 0; i < len(assets); i++ {
		// Per the OpenRTB spec docs, this is a "unique asset ID, assigned by exchange. Typically a counter for the array"
		// To avoid conflict with the Request, we'll return a 400 if the Request _did_ define this ID,
		// and then populate it as the spec suggests.
		if err := validateNativeAsset(assets[i], impIndex, i); err != nil {
			return err
		}
		assets[i].ID = int64(i)
	}
	return nil
}

func validateNativeAsset(asset nativeRequests.Asset, impIndex int, assetIndex int) error {
	if asset.ID != 0 {
		return fmt.Errorf(`request.imp[%d].native.request.assets[%d].id must not be defined. Prebid Server will set this automatically, using the index of the asset in the array as the ID.`, impIndex, assetIndex)
	}

	foundType := false

	if asset.Title != nil {
		foundType = true
		if err := validateNativeAssetTitle(asset.Title, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Img != nil {
		if foundType {
			return fmt.Errorf("request.imp[%d].native.request.assets[%d] must define at most one of {title, img, video, data}", impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetImg(asset.Img, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Video != nil {
		if foundType {
			return fmt.Errorf("request.imp[%d].native.request.assets[%d] must define at most one of {title, img, video, data}", impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetVideo(asset.Video, impIndex, assetIndex); err != nil {
			return err
		}
	}

	if asset.Data != nil {
		if foundType {
			return fmt.Errorf("request.imp[%d].native.request.assets[%d] must define at most one of {title, img, video, data}", impIndex, assetIndex)
		}
		foundType = true
		if err := validateNativeAssetData(asset.Data, impIndex, assetIndex); err != nil {
			return err
		}
	}

	return nil
}

func validateNativeAssetTitle(title *nativeRequests.Title, impIndex int, assetIndex int) error {
	if title.Len < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].title.len must be a positive integer", impIndex, assetIndex)
	}
	return nil
}

func validateNativeAssetImg(image *nativeRequests.Image, impIndex int, assetIndex int) error {
	// Note that w, wmin, h, and hmin cannot be negative because these variables use unsigned ints.
	// Those fail during the standard json.Unmarshal() call.
	if image.W == 0 && image.WMin == 0 {
		return fmt.Errorf(`request.imp[%d].native.request.assets[%d].img must contain at least one of "w" or "wmin"`, impIndex, assetIndex)
	}
	if image.H == 0 && image.HMin == 0 {
		return fmt.Errorf(`request.imp[%d].native.request.assets[%d].img must contain at least one of "h" or "hmin"`, impIndex, assetIndex)
	}

	return nil
}

func validateNativeAssetVideo(video *nativeRequests.Video, impIndex int, assetIndex int) error {
	if len(video.MIMEs) < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.mimes must be an array with at least one MIME type.", impIndex, assetIndex)
	}
	if video.MinDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.minduration must be a positive integer.", impIndex, assetIndex)
	}
	if video.MaxDuration < 1 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.maxduration must be a positive integer.", impIndex, assetIndex)
	}
	if err := validateNativeVideoProtocols(video.Protocols, impIndex, assetIndex); err != nil {
		return err
	}

	return nil
}

func validateNativeAssetData(data *nativeRequests.Data, impIndex int, assetIndex int) error {
	if data.Type < 1 || data.Type > 12 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].data.type must in the range [1, 12]. Got %d.", impIndex, assetIndex, data.Type)
	}

	return nil
}

func validateNativeVideoProtocols(protocols []nativeRequests.Protocol, impIndex int, assetIndex int) error {
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

func validateNativeVideoProtocol(protocol nativeRequests.Protocol, impIndex int, assetIndex int, protocolIndex int) error {
	if protocol < 0 || protocol > 10 {
		return fmt.Errorf("request.imp[%d].native.request.assets[%d].video.protocols[%d] must be in the range [1, 10]. Got %d", impIndex, assetIndex, protocolIndex, protocol)
	}
	return nil
}

func validateFormat(format *openrtb.Format, impIndex int, formatIndex int) error {
	usesHW := format.W != 0 || format.H != 0
	usesRatios := format.WMin != 0 || format.WRatio != 0 || format.HRatio != 0
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

func validatePmp(pmp *openrtb.PMP, impIndex int) error {
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

func (deps *endpointDeps) validateImpExt(ext openrtb.RawJSON, aliases map[string]string, impIndex int) error {
	var bidderExts map[string]openrtb.RawJSON
	if err := json.Unmarshal(ext, &bidderExts); err != nil {
		return err
	}

	if len(bidderExts) < 1 {
		return fmt.Errorf("request.imp[%d].ext must contain at least one bidder", impIndex)
	}

	for bidder, ext := range bidderExts {
		if bidder != "prebid" {
			coreBidder := bidder
			if tmp, isAlias := aliases[bidder]; isAlias {
				coreBidder = tmp
			}
			if bidderName, isValid := openrtb_ext.BidderMap[coreBidder]; isValid {
				if err := deps.paramsValidator.Validate(bidderName, ext); err != nil {
					return fmt.Errorf("request.imp[%d].ext.%s failed validation.\n%v", impIndex, coreBidder, err)
				}
			} else {
				return fmt.Errorf("request.imp[%d].ext contains unknown bidder: %s. Did you forget an alias in request.ext.prebid.aliases?", impIndex, bidder)
			}
		}
	}

	return nil
}

func (deps *endpointDeps) parseBidExt(ext openrtb.RawJSON) (*openrtb_ext.ExtRequest, error) {
	if len(ext) < 1 {
		return nil, nil
	}
	var tmpExt openrtb_ext.ExtRequest
	if err := json.Unmarshal(ext, &tmpExt); err != nil {
		return nil, fmt.Errorf("request.ext is invalid: %v", err)
	}
	return &tmpExt, nil
}

func (deps *endpointDeps) validateAliases(aliases map[string]string) error {
	for thisAlias, coreBidder := range aliases {
		if _, isCoreBidder := openrtb_ext.BidderMap[coreBidder]; !isCoreBidder {
			return fmt.Errorf("request.ext.prebid.aliases.%s refers to unknown bidder: %s", thisAlias, coreBidder)
		}
		if thisAlias == coreBidder {
			return fmt.Errorf("request.ext.prebid.aliases.%s defines a no-op alias. Choose a different alias, or remove this entry.", thisAlias)
		}
	}
	return nil
}

func (deps *endpointDeps) validateSite(site *openrtb.Site) error {
	if site != nil && site.ID == "" && site.Page == "" {
		return errors.New("request.site should include at least one of request.site.id or request.site.page.")
	}

	return nil
}

func validateUser(user *openrtb.User, aliases map[string]string) error {
	// DigiTrust support
	if user != nil && user.Ext != nil {
		// Creating ExtUser object to check if DigiTrust is valid
		var userExt openrtb_ext.ExtUser
		if err := json.Unmarshal(user.Ext, &userExt); err == nil {
			if userExt.DigiTrust != nil && userExt.DigiTrust.Pref != 0 {
				// DigiTrust is not valid. Return error.
				return errors.New("request.user contains a digitrust object that is not valid.")
			}
			// Check if the buyeruids are valid
			if userExt.Prebid != nil {
				if len(userExt.Prebid.BuyerUIDs) < 1 {
					return errors.New(`request.user.ext.prebid requires a "buyeruids" property with at least one ID defined. If none exist, then request.user.ext.prebid should not be defined.`)
				}
				for bidderName, _ := range userExt.Prebid.BuyerUIDs {
					if _, ok := openrtb_ext.BidderMap[bidderName]; !ok {
						if _, ok := aliases[bidderName]; !ok {
							return fmt.Errorf("request.user.ext.%s is neither a known bidder name nor an alias in request.ext.prebid.aliases.", bidderName)
						}
					}
				}
			}
		} else {
			// Return error.
			return fmt.Errorf("request.user.ext object is not valid: %v", err)
		}
	}

	return nil
}

func validateRegs(regs *openrtb.Regs) error {
	if regs != nil && len(regs.Ext) > 0 {
		var regsExt openrtb_ext.ExtRegs
		if err := json.Unmarshal(regs.Ext, &regsExt); err != nil {
			return fmt.Errorf("request.regs.ext is invalid: %v", err)
		}
		if regsExt.GDPR != nil && (*regsExt.GDPR < 0 || *regsExt.GDPR > 1) {
			return errors.New("request.regs.ext.gdpr must be either 0 or 1.")
		}
	}
	return nil
}

// setFieldsImplicitly uses _implicit_ information from the httpReq to set values on bidReq.
// This function does not consume the request body, which was set explicitly, but infers certain
// OpenRTB properties from the headers and other implicit info.
//
// This function _should not_ override any fields which were defined explicitly by the caller in the request.
func (deps *endpointDeps) setFieldsImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	setDeviceImplicitly(httpReq, bidReq)

	// Per the OpenRTB spec: A bid request must not contain both a Site and an App object.
	if bidReq.App == nil {
		setSiteImplicitly(httpReq, bidReq)
	}

	deps.setUserImplicitly(httpReq, bidReq)
}

// setDeviceImplicitly uses implicit info from httpReq to populate bidReq.Device
func setDeviceImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	setIPImplicitly(httpReq, bidReq) // Fixes #230
	setUAImplicitly(httpReq, bidReq)
}

// setSiteImplicitly uses implicit info from httpReq to populate bidReq.Site
func setSiteImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	if bidReq.Site == nil || bidReq.Site.Page == "" || bidReq.Site.Domain == "" {
		referrerCandidate := httpReq.Referer()
		if parsedUrl, err := url.Parse(referrerCandidate); err == nil {
			if domain, err := publicsuffix.EffectiveTLDPlusOne(parsedUrl.Host); err == nil {
				if bidReq.Site == nil {
					bidReq.Site = &openrtb.Site{}
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
}

func (deps *endpointDeps) processStoredRequests(ctx context.Context, requestJson []byte) ([]byte, []error) {
	// Parse the Stored Request IDs from the BidRequest and Imps.
	storedBidRequestId, hasStoredBidRequest, err := getStoredRequestId(requestJson)
	if err != nil {
		return nil, []error{err}
	}
	imps, impIds, idIndices, errs := parseImpInfo(requestJson)
	if len(errs) > 0 {
		return nil, errs
	}

	// Fetch all of the Stored Request data
	var allIds = make([]string, len(impIds), len(impIds)+1)
	copy(allIds, impIds)
	if hasStoredBidRequest {
		allIds = append(allIds, storedBidRequestId)
	}
	storedRequests, errs := deps.storedReqFetcher.FetchRequests(ctx, allIds)
	if len(errs) > 0 {
		return nil, errs
	}

	// Apply the Stored BidRequest, if it exists
	resolvedRequest := requestJson
	if hasStoredBidRequest {
		resolvedRequest, err = jsonpatch.MergePatch(storedRequests[storedBidRequestId], requestJson)
		if err != nil {
			return nil, []error{err}
		}
	}

	// Apply any Stored Imps, if they exist. Since the JSON Merge Patch overrides arrays,
	// and Prebid Server defers to the HTTP Request to resolve conflicts, it's safe to
	// assume that the request.imp data did not change when applying the Stored BidRequest.
	for i := 0; i < len(impIds); i++ {
		resolvedImp, err := jsonpatch.MergePatch(storedRequests[impIds[i]], imps[idIndices[i]])
		if err != nil {
			return nil, []error{err}
		}
		imps[idIndices[i]] = resolvedImp
	}
	if len(impIds) > 0 {
		newImpJson, err := json.Marshal(imps)
		if err != nil {
			return nil, []error{err}
		}
		resolvedRequest, err = jsonparser.Set(resolvedRequest, newImpJson, "imp")
		if err != nil {
			return nil, []error{err}
		}
	}

	return resolvedRequest, nil
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
		jsonparser.ArrayEach(impArray, func(imp []byte, dataType jsonparser.ValueType, offset int, err error) {
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
	value, dataType, _, err := jsonparser.Get(data, "ext", "prebid", "storedrequest", "id")
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

// setUserImplicitly uses implicit info from httpReq to populate bidReq.User
func (deps *endpointDeps) setUserImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	if bidReq.User == nil || bidReq.User.ID == "" {
		if id, ok := parseUserID(deps.cfg, httpReq); ok {
			if bidReq.User == nil {
				bidReq.User = &openrtb.User{}
			}
			if bidReq.User.ID == "" {
				bidReq.User.ID = id
			}
		}
	}
}

// setIPImplicitly sets the IP address on bidReq, if it's not explicitly defined and we can figure it out.
func setIPImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	if bidReq.Device == nil || bidReq.Device.IP == "" {
		if ip := prebid.GetIP(httpReq); ip != "" {
			if bidReq.Device == nil {
				bidReq.Device = &openrtb.Device{}
			}
			bidReq.Device.IP = ip
		}
	}
}

// setUAImplicitly sets the User Agent on bidReq, if it's not explicitly defined and it's defined on the request.
func setUAImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	if bidReq.Device == nil || bidReq.Device.UA == "" {
		if ua := httpReq.UserAgent(); ua != "" {
			if bidReq.Device == nil {
				bidReq.Device = &openrtb.Device{}
			}
			bidReq.Device.UA = ua
		}
	}
}

// parseUserId gets this user's ID  for the host machine, if it exists.
func parseUserID(cfg *config.Configuration, httpReq *http.Request) (string, bool) {
	if hostCookie, err := httpReq.Cookie(cfg.HostCookie.CookieName); hostCookie != nil && err == nil {
		return hostCookie.Value, true
	} else {
		return "", false
	}
}

// Check if a request comes from a Safari browser
func checkSafari(r *http.Request, safariRequestsMeter metrics.Meter) (isSafari bool) {
	isSafari = false
	if ua := user_agent.New(r.Header.Get("User-Agent")); ua != nil {
		name, _ := ua.Browser()
		if name == "Safari" {
			isSafari = true
			safariRequestsMeter.Mark(1)
		}
	}
	return
}

// Write(return) errors to the client, if any. Returns true if errors were found.
func writeError(errs []error, errMeter metrics.Meter, w http.ResponseWriter) bool {
	if len(errs) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errs {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		errMeter.Mark(1)
		return true
	}
	return false
}
