package openrtb2

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"github.com/mxmCherry/openrtb"
	"encoding/json"
	"github.com/prebid/prebid-server/exchange"
	"fmt"
	"context"
	"errors"
	"github.com/prebid/prebid-server/openrtb_ext"
	"time"
	"github.com/evanphx/json-patch"
	"github.com/prebid/prebid-server/openrtb2_config"
)

func NewEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator, requestsByAccount openrtb2_config.ConfigFetcher, requestsById openrtb2_config.ConfigFetcher) (httprouter.Handle, error) {
	if ex == nil || validator == nil || requestsByAccount == nil || requestsById == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}

	return httprouter.Handle((&endpointDeps{ex, validator, requestsByAccount, requestsById}).Auction), nil
}

type endpointDeps struct {
	ex exchange.Exchange
	paramsValidator openrtb_ext.BidderParamValidator
	accountFetcher openrtb2_config.ConfigFetcher
	configFetcher openrtb2_config.ConfigFetcher
}

// Slimmed down Imp.Ext object to just pull the config ID
type impExtConfig struct {
	Prebid struct { Managedconfig string `json:"managedconfig"`} `json:"prebid"`
}
// Slimmed down to extract just the ID
type impId struct {
	ID string `json:"id"`
}

func (deps *endpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, ctx, cancel, errL := deps.parseRequest(r)
	defer cancel() // Safe because parseRequest returns a no-op if there's nothing to cancel
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

	responseBytes, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to marshal auction response: %v", err)
	}

	w.WriteHeader(200)
	w.Write(responseBytes)
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
	cancel = func() { }
	errs = nil

	if err := json.NewDecoder(httpRequest.Body).Decode(req); err != nil {
		errs = []error{err}
		return
	}

	if req.TMax > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TMax) * time.Millisecond)
	}

	// Process any config directives in the impression objects.
	if errL := deps.processConfigs(ctx, req); len(errL)>0 {
		errs = errL
		return
	}

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

// Check the request for configs, patch in the stored config if found.
func (deps *endpointDeps) processConfigs(ctx context.Context, request *openrtb.BidRequest) []error {
	// Potentially handle Request level configs.

	// Pull the Imp configs.
	configIds, shortIds, errList := deps.findImpConfigIds(request.Imp)

	configs, errL := deps.configFetcher.GetConfigs(ctx, shortIds)
	if len(errL) > 0 {
		errList = append(errList, errL...)
	}

	// Process Imp level configs.
	for i := 0; i < len(request.Imp); i++ {
		// Check if a config was requested
		if len(configIds[i]) > 0 {
			conf, ok := configs[configIds[i]]
			if ok && len(conf) > 0 {
				err := deps.processImpConfig(&request.Imp[i], conf)
				if err != nil {
					errList = append(errList, err)
				}
			}
		}
	}
	if len(errList) == 0 {
		return nil
	}
	return errList
}

// Pull the Imp configs. Return both ID indexed by Imp array index, and a simple list of existing IDs.
func (deps *endpointDeps) findImpConfigIds(imps []openrtb.Imp) ([]string, []string, []error) {
	errList := make([]error, 0, len(imps))
	configIds := make([]string, len(imps))
	shortIds := make([]string, 0, len(imps))
	for i := 0; i < len(imps); i++ {
		if imps[i].Ext != nil && len(imps[i].Ext) > 0 {
			eConf := &impExtConfig{}
			err := json.Unmarshal(imps[i].Ext, eConf)
			if err == nil && len(eConf.Prebid.Managedconfig) > 0 {
				configIds[i] = eConf.Prebid.Managedconfig
				shortIds = append(shortIds, eConf.Prebid.Managedconfig)
			} else if len(eConf.Prebid.Managedconfig) > 0 {
				errList = append(errList, err)
				configIds[i] = ""
			}
		} else{
			configIds[i] = ""
		}
	}
	return configIds, shortIds, errList
}


// Process the configs for an Imp. Need to modify the Imp object in place as we cannot simply assign one Imp
// to another (deep copy)
func (deps *endpointDeps) processImpConfig(imp *openrtb.Imp, conf json.RawMessage) error {
	impJson, err := json.Marshal(imp)
	if err != nil {
		return err
	}

	newImp, err := jsonpatch.MergePatch(conf, impJson)
	if err != nil {
		return err
	}
	err = json.Unmarshal(newImp, imp)
	// Due to the definition of openrtb.Imp, ID will always have a value, even if empty. So we need to restore the ID
	// from the config if overriden by a blank ID.
	if imp.ID == "" {
		confId := &impId{}
		err = json.Unmarshal(conf, confId)
		imp.ID = confId.ID
	}
	return err
}