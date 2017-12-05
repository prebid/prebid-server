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
	"github.com/golang/glog"
)

func NewEndpoint(ex exchange.Exchange, validator openrtb_ext.BidderParamValidator) (httprouter.Handle, error) {
	if ex == nil || validator == nil {
		return nil, errors.New("NewEndpoint requires non-nil arguments.")
	}
	return httprouter.Handle((&endpointDeps{ex, validator}).Auction), nil
}

type endpointDeps struct {
	ex exchange.Exchange
	paramsValidator openrtb_ext.BidderParamValidator
}

func (deps *endpointDeps) Auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	req, err := deps.parseRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Invalid request format: %s", err.Error())))
		return
	}
	ctx := context.Background()
	cancel := func() { }
	if req.TMax > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TMax) * time.Millisecond)
		defer cancel()
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

// parseRequest turns the HTTP request into an OpenRTB request.
//
// This will return an error if the request couldn't be parsed, or if the request isn't valid according
// to the OpenRTB 2.5 spec.
//
// It will also return errors for some of the "strong recommendations" in the spec, as long as
// the same request can be sent in a better way which agrees with the recommendations.
func (deps *endpointDeps) parseRequest(httpRequest *http.Request) (*openrtb.BidRequest, error) {
	var ortbRequest openrtb.BidRequest
	if err := json.NewDecoder(httpRequest.Body).Decode(&ortbRequest); err != nil {
		return nil, err
	}

	if err := deps.validateRequest(&ortbRequest); err != nil {
		return nil, err
	}

	return &ortbRequest, nil
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
		} else {
			return fmt.Errorf("request.imp[%d].ext contains unknown bidder: %s", impIndex, bidder)
		}
	}

	return nil
}
