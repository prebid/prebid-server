package relevantdigital

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/macros"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
	name     string
}

const relevant_domain = ".relevant-digital.com"
const default_timeout = 1000
const default_bufffer_ms = 250
const stored_request_ext = "{\"prebid\":{\"debug\":%t,\"storedrequest\":{\"id\":\"%s\"}},\"relevant\":{\"count\":%d,\"adapterType\":\"server\"}}"
const stored_imp_ext = "{\"prebid\":{\"storedrequest\":{\"id\":\"%s\"}}}"

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	return &adapter{
		endpoint: template,
		name:     bidderName.String(),
	}, nil
}

func patchBidRequestExt(prebidBidRequest *openrtb2.BidRequest, id string) error {
	count, cerr := jsonparser.GetInt(prebidBidRequest.Ext, "relevant", "count")
	if cerr != nil {
		count = 0
	}

	if count >= 5 {
		return &errortypes.FailedToRequestBids{
			Message: "too many requests",
		}
	} else {
		count = count + 1
	}

	debug, derr := jsonparser.GetBoolean(prebidBidRequest.Ext, "prebid", "debug")
	if derr != nil {
		debug = false
	}

	prebidBidRequest.Ext = []byte(fmt.Sprintf(stored_request_ext, debug, id, count))
	return nil
}

func patchBidImpExt(imp *openrtb2.Imp, id string) {
	imp.Ext = []byte(fmt.Sprintf(stored_imp_ext, id))
	if imp.Banner != nil {
		imp.Banner.Ext = nil
	}
	if imp.Video != nil {
		imp.Video.Ext = nil
	}
	if imp.Native != nil {
		imp.Native.Ext = nil
	}
	if imp.Audio != nil {
		imp.Audio.Ext = nil
	}
}

func setTMax(prebidBidRequest *openrtb2.BidRequest, pbsBufferMs int) {
	timeout := float64(prebidBidRequest.TMax)
	if timeout <= 0 {
		timeout = default_timeout
	}
	buffer := float64(pbsBufferMs)
	prebidBidRequest.TMax = int64(math.Min(math.Max(timeout-buffer, buffer), timeout))
}

func cloneBidRequest(prebidBidRequest *openrtb2.BidRequest) (*openrtb2.BidRequest, error) {
	jsonRes, err := json.Marshal(prebidBidRequest)
	if err != nil {
		return nil, err
	}
	var copy openrtb2.BidRequest
	err = json.Unmarshal(jsonRes, &copy)
	return &copy, err
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params []*openrtb_ext.ExtRelevantDigital) (*openrtb2.BidRequest, error) {
	bidRequestCopy, err := cloneBidRequest(prebidBidRequest)
	if err != nil {
		return nil, err
	}

	err = patchBidRequestExt(bidRequestCopy, params[0].AccountId)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("failed to create bidRequest, error: %s", err),
		}
	}

	setTMax(bidRequestCopy, params[0].PbsBufferMs)

	for idx := range bidRequestCopy.Imp {
		patchBidImpExt(&bidRequestCopy.Imp[idx], params[idx].PlacementId)
	}
	return bidRequestCopy, err
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtRelevantDigital, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "imp.ext not provided",
		}
	}
	relevantExt := openrtb_ext.ExtRelevantDigital{PbsBufferMs: default_bufffer_ms}
	if err := json.Unmarshal(bidderExt.Bidder, &relevantExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "ext.bidder not provided",
		}
	}
	return &relevantExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtRelevantDigital) (string, error) {
	params.Host = strings.ReplaceAll(params.Host, "http://", "")
	params.Host = strings.ReplaceAll(params.Host, "https://", "")
	params.Host = strings.ReplaceAll(params.Host, relevant_domain, "")

	endpointParams := macros.EndpointTemplateParams{Host: params.Host}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *adapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params []*openrtb_ext.ExtRelevantDigital) (*adapters.RequestData, error) {
	newBidRequest, err := createBidRequest(prebidBidRequest, params)

	if err != nil {
		return nil, err
	}

	reqJSON, err := json.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	url, err := a.buildEndpointURL(params[0])
	if err != nil {
		return nil, err
	}

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     url,
		Body:    reqJSON,
		Headers: getHeaders(prebidBidRequest),
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	impParams, errs := getImpressionsInfo(request.Imp)
	if len(errs) > 0 {
		return nil, errs
	}

	bidRequest, err := a.buildAdapterRequest(request, impParams)
	if err != nil {
		errs = []error{err}
	}

	if bidRequest != nil {
		return []*adapters.RequestData{bidRequest}, errs
	}
	return nil, errs
}

func getImpressionsInfo(imps []openrtb2.Imp) (resImps []*openrtb_ext.ExtRelevantDigital, errors []error) {
	for _, imp := range imps {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		resImps = append(resImps, impExt)
	}
	return
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.UA) > 0 {
			headers.Add("User-Agent", request.Device.UA)
		}
		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}
	return headers
}

func getMediaTypeForBidFromExt(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}
	return "", fmt.Errorf("failed to parse bid type, missing ext: %s", bid.ImpID)
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return getMediaTypeForBidFromExt(bid)
	}
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(response.SeatBid))
	bidResponse.Currency = response.Cur
	var errs []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, err)
			} else {
				b := &adapters.TypedBid{
					Bid:     &seatBid.Bid[i],
					BidType: bidType,
				}
				bidResponse.Bids = append(bidResponse.Bids, b)
			}
		}
	}
	return bidResponse, errs
}
