package openrtb2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/evanphx/json-patch"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid"
	"github.com/prebid/prebid-server/stored_requests"
	"golang.org/x/net/publicsuffix"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

func NewEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsById stored_requests.Fetcher, cfg *config.Configuration) (httprouter.Handle, error) {
	if ex == nil || validator == nil || requestsById == nil || cfg == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}

	return httprouter.Handle((&endpointDeps{ex, validator, requestsById, cfg}).Auction), nil
}

type endpointDeps struct {
	ex               exchange.Exchange
	paramsValidator  openrtb_ext.BidderParamValidator
	storedReqFetcher stored_requests.Fetcher
	cfg              *config.Configuration
}

func (deps *endpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, ctx, cancel, errL := deps.parseRequest(r)
	defer cancel() // Safe because parseRequest returns a no-op even if errors are present.
	if len(errL) > 0 {
		w.WriteHeader(http.StatusBadRequest)
		for _, err := range errL {
			w.Write([]byte(fmt.Sprintf("Invalid request format: %s\n", err.Error())))
		}
		return
	}

	response, err := deps.ex.HoldAuction(ctx, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Critical error while running the auction: %v", err)
		return
	}

	// Fixes #231
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

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
func (deps *endpointDeps) parseRequest(httpRequest *http.Request) (req *openrtb.BidRequest, ctx context.Context, cancel func(), errs []error) {
	req = &openrtb.BidRequest{}
	ctx = context.Background()
	cancel = func() {}
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

	// Apply any Stored Requests inside the BidRequest or Impression objects.
	if requestJson, errs = deps.processStoredRequests(ctx, requestJson); len(errs) > 0 {
		return
	}

	if err := json.Unmarshal(requestJson, req); err != nil {
		errs = []error{err}
		return
	}

	if req.TMax > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TMax)*time.Millisecond)
	}

	setFieldsImplicitly(httpRequest, req)

	if err := deps.validateRequest(req); err != nil {
		errs = []error{err}
		return
	}
	return
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

	for index, imp := range req.Imp {
		if err := deps.validateImp(&imp, index); err != nil {
			return err
		}
	}

	if (req.Site == nil && req.App == nil) || (req.Site != nil && req.App != nil) {
		return errors.New("request.site or request.app must be defined, but not both.")
	}

	if err := deps.validateSite(req.Site); err != nil {
		return err
	}

	return nil
}

func (deps *endpointDeps) validateImp(imp *openrtb.Imp, index int) error {
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

	if imp.Native != nil {
		if imp.Native.Request == "" {
			return fmt.Errorf("request.imp[%d].native.request must be a JSON encoded string conforming to the openrtb 1.2 Native spec", index)
		}
	}

	if err := validatePmp(imp.PMP, index); err != nil {
		return err
	}

	if err := deps.validateImpExt(imp.Ext, index); err != nil {
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

func (deps *endpointDeps) validateImpExt(ext openrtb.RawJSON, impIndex int) error {
	var bidderExts map[string]openrtb.RawJSON
	if err := json.Unmarshal(ext, &bidderExts); err != nil {
		return err
	}

	if len(bidderExts) < 1 {
		return fmt.Errorf("request.imp[%d].ext must contain at least one bidder", impIndex)
	}

	for bidder, ext := range bidderExts {
		bidderName, isValid := openrtb_ext.GetBidderName(bidder)
		if isValid {
			if err := deps.paramsValidator.Validate(bidderName, ext); err != nil {
				return fmt.Errorf("request.imp[%d].ext.%s failed validation.\n%v", impIndex, bidder, err)
			}
		} else if bidder != "prebid" {
			return fmt.Errorf("request.imp[%d].ext contains unknown bidder: %s", impIndex, bidder)
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

// setFieldsImplicitly uses _implicit_ information from the httpReq to set values on bidReq.
// This function does not consume the request body, which was set explicitly, but infers certain
// OpenRTB properties from the headers and other implicit info.
//
// This function _should not_ override any fields which were defined explicitly by the caller in the request.
func setFieldsImplicitly(httpReq *http.Request, bidReq *openrtb.BidRequest) {
	setDeviceImplicitly(httpReq, bidReq)

	// Per the OpenRTB spec: A bid request must not contain both a Site and an App object.
	if bidReq.App == nil {
		setSiteImplicitly(httpReq, bidReq)
	}
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
	storedRequestIds := make([]string, 0, 10)

	// Pull all the StoredRequest IDs from the BidRequest and Imps
	storedBidRequestId, hasStoredBidRequest, err := getStoredRequestId(requestJson)
	if err != nil {
		return nil, []error{err}
	} else if hasStoredBidRequest {
		storedRequestIds = append(storedRequestIds, storedBidRequestId)
	}

	// If request.imp exists, search through those imps for stored request IDs too.
	if imps, dataType, _, err := jsonparser.Get(requestJson, "imp"); err == nil && dataType == jsonparser.Array {
		var errs []error
		jsonparser.ArrayEach(imps, func(imp []byte, dataType jsonparser.ValueType, offset int, err error) {
			if storedImpId, hasStoredImp, err := getStoredRequestId(imp); err != nil {
				errs = append(errs, err)
			} else if hasStoredImp {
				storedRequestIds = append(storedRequestIds, storedImpId)
			}
		})
		if len(errs) > 0 {
			return nil, errs
		}
	}

	// Fetch all the stored requests
	storedRequests, errs := deps.storedReqFetcher.FetchRequests(context.TODO(), storedRequestIds)
	if len(errs) > 0 {
		return nil, errs
	}

	// Apply the BidRequest patch
	resolvedRequest := requestJson
	if hasStoredBidRequest {
		resolvedRequest, err = jsonpatch.MergePatch(storedRequests[storedRequestIds[0]], requestJson)
		if err != nil {
			return nil, []error{err}
		}
	}

	// Since the BidRequest patch may have included new imp data, we need to loop through this new document's
	// imps again to make sure we apply the right patches in the right places.
	if imps, dataType, _, err := jsonparser.Get(resolvedRequest, "imp"); err == nil && dataType == jsonparser.Array {
		var errs []error
		newImps := make([]json.RawMessage, 0, 10)
		jsonparser.ArrayEach(imps, func(imp []byte, dataType jsonparser.ValueType, offset int, err error) {
			if storedImpId, hasStoredImp, err := getStoredRequestId(imp); err != nil {
				errs = append(errs, err)
			} else if hasStoredImp {
				// If this Stored Imp uses an unknown ID, then it must have been inside the Stored BidRequest
				// data we just merged. Since we don't support nested stored things, it's ok to ignore them.
				if storedImpData, ok := storedRequests[storedImpId]; ok {
					newImp, err := jsonpatch.MergePatch(storedImpData, imp)
					if err != nil {
						errs = append(errs, err)
					} else {
						newImps = append(newImps, newImp)
					}
				} else {
					newImps = append(newImps, imp)
				}
				storedRequestIds = append(storedRequestIds, storedImpId)
			} else {
				newImps = append(newImps, imp)
			}
		})
		if len(errs) > 0 {
			return nil, errs
		}

		newImpJson, err := json.Marshal(newImps)
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

// Pull the Stored Request IDs from the Imps. Return both ID indexed by Imp array index, and a simple list of existing IDs.
func (deps *endpointDeps) findStoredRequestIds(imps []openrtb.Imp) ([]string, []string, []error) {
	errList := make([]error, 0, len(imps))
	storedReqIds := make([]string, len(imps))
	shortIds := make([]string, 0, len(imps))
	for i := 0; i < len(imps); i++ {
		if imps[i].Ext != nil && len(imps[i].Ext) > 0 {
			storedReqId, _, err := getStoredRequestId(imps[i].Ext)
			if err == nil && len(storedReqId) > 0 {
				storedReqIds[i] = storedReqId
				shortIds = append(shortIds, storedReqId)
			} else if len(storedReqId) > 0 {
				errList = append(errList, err)
				storedReqIds[i] = ""
			}
		} else {
			storedReqIds[i] = ""
		}
	}
	return storedReqIds, shortIds, errList
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
