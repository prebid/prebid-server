package adkernel

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	mf_suffix        = "__mf"
	mf_suffix_banner = "b" + mf_suffix
	mf_suffix_video  = "v" + mf_suffix
	mf_suffix_audio  = "a" + mf_suffix
	mf_suffix_native = "n" + mf_suffix
)

type adkernelAdapter struct {
	EndpointTemplate *template.Template
}

// MakeRequests prepares request information for prebid-server core
func (adapter *adkernelAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, newBadInputError("No impression in the bid request"))
		return nil, errs
	}
	imps, impExts, err := getImpressionsInfo(request.Imp)
	if len(imps) == 0 {
		return nil, err
	}
	errs = append(errs, err...)

	pub2impressions, dispErrors := dispatchImpressions(imps, impExts)
	if len(dispErrors) > 0 {
		errs = append(errs, dispErrors...)
	}
	if len(pub2impressions) == 0 {
		return nil, errs
	}
	result := make([]*adapters.RequestData, 0, len(pub2impressions))
	for k, imps := range pub2impressions {
		bidRequest, err := adapter.buildAdapterRequest(request, &k, imps)
		if err != nil {
			errs = append(errs, err)
		} else {
			result = append(result, bidRequest)
		}
	}
	return result, errs
}

// getImpressionsInfo checks each impression for validity and returns impressions copy with corresponding exts
func getImpressionsInfo(imps []openrtb2.Imp) ([]openrtb2.Imp, []openrtb_ext.ExtImpAdkernel, []error) {
	impsCount := len(imps)
	errors := make([]error, 0, impsCount)
	resImps := make([]openrtb2.Imp, 0, impsCount)
	resImpExts := make([]openrtb_ext.ExtImpAdkernel, 0, impsCount)

	for _, imp := range imps {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if err := validateImpression(&imp, impExt); err != nil {
			errors = append(errors, err)
			continue
		}
		resImps = append(resImps, imp)
		resImpExts = append(resImpExts, *impExt)
	}
	return resImps, resImpExts, errors
}

func validateImpression(imp *openrtb2.Imp, impExt *openrtb_ext.ExtImpAdkernel) error {
	if impExt.ZoneId < 1 {
		return newBadInputError(fmt.Sprintf("Invalid zoneId value: %d. Ignoring imp id=%s", impExt.ZoneId, imp.ID))
	}
	return nil
}

// Group impressions by AdKernel-specific parameter `zoneId`
func dispatchImpressions(imps []openrtb2.Imp, impsExt []openrtb_ext.ExtImpAdkernel) (map[openrtb_ext.ExtImpAdkernel][]openrtb2.Imp, []error) {
	res := make(map[openrtb_ext.ExtImpAdkernel][]openrtb2.Imp)
	errors := make([]error, 0)
	for idx := range imps {
		imp := imps[idx]
		imp.Ext = nil
		impExt := impsExt[idx]
		if res[impExt] == nil {
			res[impExt] = make([]openrtb2.Imp, 0)
		}
		if isMultiFormatImp(&imp) {
			splImps := splitMultiFormatImp(&imp)
			res[impExt] = append(res[impExt], splImps...)
		} else {
			res[impExt] = append(res[impExt], imp)
		}
	}
	return res, errors
}

func isMultiFormatImp(imp *openrtb2.Imp) bool {
	count := 0
	if imp.Video != nil {
		count++
	}
	if imp.Audio != nil {
		count++
	}
	if imp.Banner != nil {
		count++
	}
	if imp.Native != nil {
		count++
	}
	return count > 1
}

func splitMultiFormatImp(imp *openrtb2.Imp) []openrtb2.Imp {
	splitImps := make([]openrtb2.Imp, 0, 4)
	if imp.Banner != nil {
		impCopy := *imp
		impCopy.Video = nil
		impCopy.Native = nil
		impCopy.Audio = nil
		impCopy.ID += mf_suffix_banner
		splitImps = append(splitImps, impCopy)
	}
	if imp.Video != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Native = nil
		impCopy.Audio = nil
		impCopy.ID += mf_suffix_video
		splitImps = append(splitImps, impCopy)
	}

	if imp.Native != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Video = nil
		impCopy.Audio = nil
		impCopy.ID += mf_suffix_native
		splitImps = append(splitImps, impCopy)
	}

	if imp.Audio != nil {
		impCopy := *imp
		impCopy.Banner = nil
		impCopy.Video = nil
		impCopy.Native = nil
		impCopy.ID += mf_suffix_audio
		splitImps = append(splitImps, impCopy)
	}
	return splitImps
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpAdkernel, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var adkernelExt openrtb_ext.ExtImpAdkernel
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adkernelExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	return &adkernelExt, nil
}

func (adapter *adkernelAdapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpAdkernel, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	newBidRequest := createBidRequest(prebidBidRequest, params, imps)
	reqJSON, err := json.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	url, err := adapter.buildEndpointURL(params)
	if err != nil {
		return nil, err
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps)}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpAdkernel, imps []openrtb2.Imp) *openrtb2.BidRequest {
	bidRequest := *prebidBidRequest
	bidRequest.Imp = imps
	if bidRequest.Site != nil {
		// Need to copy Site as Request is a shallow copy
		siteCopy := *bidRequest.Site
		bidRequest.Site = &siteCopy
		bidRequest.Site.Publisher = nil
	}
	if bidRequest.App != nil {
		// Need to copy App as Request is a shallow copy
		appCopy := *bidRequest.App
		bidRequest.App = &appCopy
		bidRequest.App.Publisher = nil
	}
	return &bidRequest
}

// Builds endpoint url based on adapter-specific pub settings from imp.ext
func (adapter *adkernelAdapter) buildEndpointURL(params *openrtb_ext.ExtImpAdkernel) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ZoneID: strconv.Itoa(params.ZoneId)}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

// MakeBids translates adkernel bid response to prebid-server specific format
func (adapter *adkernelAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)),
		}
	}
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Bad server response: %d", err)),
		}
	}

	if len(bidResp.SeatBid) != 1 {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Invalid SeatBids count: %d", len(bidResp.SeatBid))),
		}
	}

	seatBid := bidResp.SeatBid[0]
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	bidResponse.Currency = bidResp.Cur
	for i := 0; i < len(seatBid.Bid); i++ {
		bid := seatBid.Bid[i]
		if strings.HasSuffix(bid.ImpID, mf_suffix) {
			sfxStart := len(bid.ImpID) - len(mf_suffix) - 1
			bid.ImpID = bid.ImpID[:sfxStart]
		}
		bidType, err := getMediaTypeForBid(&bid)
		if err != nil {
			return nil, []error{err}
		}
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: bidType,
		})
	}
	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for
func getMediaTypeForBid(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d", bid.MType),
		}
	}
}

func newBadInputError(message string) error {
	return &errortypes.BadInput{
		Message: message,
	}
}

func newBadServerResponseError(message string) error {
	return &errortypes.BadServerResponse{
		Message: message,
	}
}

// Builder builds a new instance of the Adkernel adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	urlTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adkernelAdapter{
		EndpointTemplate: urlTemplate,
	}
	return bidder, nil
}
