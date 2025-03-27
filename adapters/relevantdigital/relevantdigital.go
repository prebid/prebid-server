package relevantdigital

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"text/template"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"
)

type adapter struct {
	endpoint *template.Template
	name     string
}

const relevant_domain = ".relevant-digital.com"
const default_timeout = 1000
const default_bufffer_ms = 250

type prebidExt struct {
	StoredRequest struct {
		Id string `json:"id"`
	} `json:"storedrequest"`
	Debug bool `json:"debug"`
}

type relevantExt struct {
	Relevant struct {
		Count       int    `json:"count"`
		AdapterType string `json:"adapterType"`
	} `json:"relevant"`
	Prebid prebidExt `json:"prebid"`
}

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
	var bidRequestExt relevantExt
	if len(prebidBidRequest.Ext) != 0 {
		if err := jsonutil.Unmarshal(prebidBidRequest.Ext, &bidRequestExt); err != nil {
			return &errortypes.FailedToRequestBids{
				Message: fmt.Sprintf("failed to unmarshal ext, %s", prebidBidRequest.Ext),
			}
		}
	}

	count := bidRequestExt.Relevant.Count
	if bidRequestExt.Relevant.Count >= 5 {
		return &errortypes.FailedToRequestBids{
			Message: "too many requests",
		}
	} else {
		count = count + 1
	}

	bidRequestExt.Relevant.Count = count
	bidRequestExt.Relevant.AdapterType = "server"
	bidRequestExt.Prebid.StoredRequest.Id = id

	ext, err := json.Marshal(bidRequestExt)
	if err != nil {
		return &errortypes.FailedToRequestBids{
			Message: "failed to marshal",
		}
	}

	if len(prebidBidRequest.Ext) == 0 {
		prebidBidRequest.Ext = ext
		return nil
	}

	patchedExt, err := jsonpatch.MergePatch(prebidBidRequest.Ext, ext)
	if err != nil {
		return &errortypes.FailedToRequestBids{
			Message: fmt.Sprintf("failed patch ext, %s", err),
		}
	}
	prebidBidRequest.Ext = patchedExt
	return nil
}

func patchBidImpExt(imp *openrtb2.Imp, id string) {
	imp.Ext = []byte(fmt.Sprintf("{\"prebid\":{\"storedrequest\":{\"id\":\"%s\"}}}", id))
}

func setTMax(prebidBidRequest *openrtb2.BidRequest, pbsBufferMs int) {
	timeout := float64(prebidBidRequest.TMax)
	if timeout <= 0 {
		timeout = default_timeout
	}
	buffer := float64(pbsBufferMs)
	prebidBidRequest.TMax = int64(math.Min(math.Max(timeout-buffer, buffer), timeout))
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params []*openrtb_ext.ExtRelevantDigital) ([]byte, error) {
	bidRequestCopy := *prebidBidRequest

	err := patchBidRequestExt(&bidRequestCopy, params[0].AccountId)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("failed to create bidRequest, error: %s", err),
		}
	}

	setTMax(&bidRequestCopy, params[0].PbsBufferMs)

	for idx := range bidRequestCopy.Imp {
		patchBidImpExt(&bidRequestCopy.Imp[idx], params[idx].PlacementId)
	}

	return createJSONRequest(&bidRequestCopy)
}

func createJSONRequest(bidRequest *openrtb2.BidRequest) ([]byte, error) {
	reqJSON, err := json.Marshal(bidRequest)
	if err != nil {
		return nil, err
	}

	// Scrub previous ext data from relevant, if any
	// imp[].ext.context.relevant
	// imp[].[banner/native/video/audio].ext.relevant
	impKeyTypes := []string{"banner", "video", "native", "audio"}
	for idx := range bidRequest.Imp {
		for _, key := range impKeyTypes {
			reqJSON = jsonparser.Delete(reqJSON, "imp", fmt.Sprintf("[%d]", idx), key, "ext", "relevant")
		}
		reqJSON = jsonparser.Delete(reqJSON, "imp", fmt.Sprintf("[%d]", idx), "ext", "context", "relevant")
	}

	// Scrub previous prebid data (to not set cache on wrong servers)
	// ext.prebid.[cache/targeting/aliases]
	prebidKeyTypes := []string{"cache", "targeting", "aliases"}
	for _, key := range prebidKeyTypes {
		reqJSON = jsonparser.Delete(reqJSON, "ext", "prebid", key)
	}
	return reqJSON, nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtRelevantDigital, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "imp.ext not provided",
		}
	}
	relevantExt := openrtb_ext.ExtRelevantDigital{PbsBufferMs: default_bufffer_ms}
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &relevantExt); err != nil {
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
	reqJSON, err := createBidRequest(prebidBidRequest, params)

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
		ImpIDs:  openrtb_ext.GetImpIDs(prebidBidRequest.Imp),
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
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
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

func isSupportedMediaType(bidType openrtb_ext.BidType) error {
	switch bidType {
	case openrtb_ext.BidTypeBanner:
		fallthrough
	case openrtb_ext.BidTypeVideo:
		fallthrough
	case openrtb_ext.BidTypeAudio:
		fallthrough
	case openrtb_ext.BidTypeNative:
		return nil
	}
	return fmt.Errorf("bid type not supported %s", bidType)
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
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
				continue
			}
			if err := isSupportedMediaType(bidType); err != nil {
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
