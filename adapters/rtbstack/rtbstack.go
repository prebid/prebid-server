package rtbstack

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/macros"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

// impCtx represents the context containing an OpenRTB impression and its corresponding RTBStack extension configuration.
type impCtx struct {
	imp         openrtb2.Imp
	rtbStackExt *openrtb_ext.ExtImpRTBStack
}

// extImpRTBStack is used for imp->ext when sending to rtb-stack backend.
type extImpRTBStack struct {
	CustomParams map[string]interface{} `json:"customParams,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	tpl, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: tpl,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impressions in request",
		}}
	}

	var errs []error
	impsByRoute := make(map[string][]*impCtx)
	var routeOrder []string

	for i := range request.Imp {
		imp, ext, err := preprocessImp(request.Imp[i])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		ctx := &impCtx{
			imp:         imp,
			rtbStackExt: ext,
		}
		if _, ok := impsByRoute[ext.Route]; !ok {
			routeOrder = append(routeOrder, ext.Route)
		}
		impsByRoute[ext.Route] = append(impsByRoute[ext.Route], ctx)
	}

	if len(routeOrder) == 0 {
		return nil, errs
	}

	var newSite *openrtb2.Site
	if request.Site != nil && request.Site.Domain == "" {
		siteCopy := *request.Site
		pageURL, parseErr := url.Parse(request.Site.Page)
		if parseErr == nil && pageURL.Hostname() != "" {
			siteCopy.Domain = pageURL.Hostname()
		} else {
			siteCopy.Domain = request.Site.Page
		}
		newSite = &siteCopy
	}

	requests := make([]*adapters.RequestData, 0, len(routeOrder))
	for _, route := range routeOrder {
		group := impsByRoute[route]

		endpoint, err := a.buildEndpointURL(group[0].rtbStackExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		processedImps := make([]openrtb2.Imp, 0, len(group))
		for _, v := range group {
			processedImps = append(processedImps, v.imp)
		}

		newRequest := *request
		newRequest.Imp = processedImps
		if newSite != nil {
			newRequest.Site = newSite
		}

		reqJSON, err := jsonutil.Marshal(newRequest)
		if err != nil {
			errs = append(errs, &errortypes.FailedToRequestBids{
				Message: "Error parsing reqJSON object",
			})
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(newRequest.Imp),
		})
	}

	if len(requests) == 0 {
		return nil, errs
	}

	return requests, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))

	if bidResp.Cur != "" {
		bidResponse.Currency = bidResp.Cur
	}

	var bidErrs []error
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			bidType, err := getMediaTypeForBid(sb.Bid[i])
			if err != nil {
				bidErrs = append(bidErrs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: bidType,
			})
		}
	}
	return bidResponse, bidErrs
}

var validRegions = map[string]bool{"us": true, "eu": true, "sg": true}

func (a *adapter) buildEndpointURL(ext *openrtb_ext.ExtImpRTBStack) (string, error) {
	routeURL, err := url.Parse(ext.Route)
	if err != nil {
		return "", &errortypes.BadInput{Message: fmt.Sprintf("invalid route URL: %v", err)}
	}

	region, err := extractRegion(routeURL.Hostname())
	if err != nil {
		return "", err
	}

	queryParams := routeURL.Query()
	client := queryParams.Get("client")
	endpoint := queryParams.Get("endpoint")
	ssp := queryParams.Get("ssp")

	if client == "" || endpoint == "" || ssp == "" {
		return "", &errortypes.BadInput{Message: "route URL must contain client, endpoint, and ssp query parameters"}
	}

	params := macros.EndpointTemplateParams{
		Region:    region,
		SspID:     ssp,
		ZoneID:    endpoint,
		PartnerId: client,
	}

	return macros.ResolveMacros(a.endpoint, params)
}

func extractRegion(hostname string) (string, error) {
	parts := strings.Split(hostname, ".")
	for _, part := range parts {
		if strings.HasSuffix(part, "-adx-admixer") {
			region := strings.ToLower(strings.TrimSuffix(part, "-adx-admixer"))
			if validRegions[region] {
				return region, nil
			}
		}
	}
	return "", &errortypes.BadInput{Message: "unable to extract valid region from route URL hostname"}
}

func preprocessImp(
	imp openrtb2.Imp,
) (openrtb2.Imp, *openrtb_ext.ExtImpRTBStack, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return imp, nil, &errortypes.BadInput{Message: err.Error()}
	}

	var impExt openrtb_ext.ExtImpRTBStack
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &impExt); err != nil {
		return imp, nil, &errortypes.BadInput{
			Message: "Wrong RTBStack bidder ext",
		}
	}

	imp.TagID = impExt.TagId

	newExt := extImpRTBStack{
		CustomParams: impExt.CustomParams,
	}

	newImpExtForRTBStack, err := jsonutil.Marshal(newExt)
	if err != nil {
		return imp, nil, &errortypes.BadInput{Message: err.Error()}
	}
	imp.Ext = newImpExtForRTBStack

	return imp, &impExt, nil
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("unsupported MType %d for bid %s", bid.MType, bid.ImpID),
		}
	}
}
