package nativery

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Function used to  builds a new instance of the Nativery adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	// build bidder
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// makeRequests creates HTTP requests for a given BidRequest and adapter configuration.
// It generates requests for each ad exchange targeted by the BidRequest,
// serializes the BidRequest into the request body, and sets the appropriate
// HTTP headers and other parameters.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	reqCopy := *request
	var errs []error

	// check if the request come from AMP
	var isAMP int
	if reqInfo.PbsEntryPoint == metrics.ReqTypeAMP {
		isAMP = 1
	}

	var widgetId string

	// attach body request for all the impressions
	validImps := []openrtb2.Imp{}
	for i, imp := range request.Imp {
		reqCopy.Imp = []openrtb2.Imp{imp}

		nativeryExt, err := buildNativeryExt(&reqCopy.Imp[0])
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// at the first impression set widgetId value
		if i == 0 {
			widgetId = nativeryExt.WidgetId
		}

		if err := buildRequest(reqCopy, nativeryExt); err != nil {
			errs = append(errs, err)
			continue
		}

		validImps = append(validImps, reqCopy.Imp...)

	}

	reqCopy.Imp = validImps
	// If all the requests were malformed, don't bother making a server call with no impressions.
	if len(reqCopy.Imp) == 0 {
		return nil, errs
	}

	reqExt, err := getRequestExt(reqCopy.Ext)
	if err != nil {
		return nil, append(errs, err)
	}

	reqExtNativery, err := getNativeryExt(reqExt, isAMP, widgetId)
	if err != nil {
		return nil, append(errs, err)
	}
	// TODO: optimize it, we reiterate imp there and before
	adapterRequests, errors := splitRequests(reqCopy.Imp, &reqCopy, reqExt, reqExtNativery, a.endpoint)

	return adapterRequests, append(errs, errors...)
}

func buildNativeryExt(imp *openrtb2.Imp) (openrtb_ext.ImpExtNativery, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ImpExtNativery{}, err
	}

	var nativeryExt openrtb_ext.ImpExtNativery
	if err := json.Unmarshal(bidderExt.Bidder, &nativeryExt); err != nil {
		return openrtb_ext.ImpExtNativery{}, err
	}

	return nativeryExt, nil
}

// utility function used to build the body for the http request for a single impression
func buildRequest(reqCopy openrtb2.BidRequest, reqExt openrtb_ext.ImpExtNativery) error {

	impExt := impExt{Nativery: nativeryExtReqBody{
		Id:  reqExt.WidgetId,
		Xhr: 2,
		V:   3,
		// TODO: Site is only for browser request, we have to handle if the req comes from app or dooh
		Ref:    reqCopy.Site.Page,
		RefRef: refRef{Page: reqCopy.Site.Page, Ref: reqCopy.Site.Ref},
	}}

	var err error
	reqCopy.Imp[0].Ext, err = json.Marshal(&impExt)

	return err
}

// makebids handles the entire bidding process for a single BidRequest.
// It creates and sends bid requests to multiple ad exchanges, receives
// and parses responses, extracts bids and other relevant information,
// and populates a BidderResponse object with the aggregated information.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// check if the response has no content
	if adapters.IsResponseStatusCodeNoContent(response) {
		// Extract nativery no content reason if is present
		reason := ""
		if response.Headers != nil {
			reason = response.Headers.Get("X-Nativery-Error")
		}
		if reason == "" {
			reason = "No Content"
		}
		// Add the reason to errors
		return nil, []error{errors.New(reason)}
	}

	// check if the response has errors
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	// handle response
	var nativeryResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &nativeryResponse); err != nil {
		return nil, []error{err}
	}

	var errs []error
	// create bidder with impressions length capacity
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	for _, sb := range nativeryResponse.SeatBid {
		for i := range sb.Bid {
			bid := sb.Bid[i]

			// should be data sended from nativery server to partecipate to the auction
			var bidExt bidExt
			if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
				errs = append(errs, err)
				continue
			}

			bidType, err := getMediaTypeForBid(&bidExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			// get metadata
			bidMeta := buildBidMeta(string(bidType), bidExt.Nativery.BidAdvDomains)

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
				// metadata is encouraged
				BidMeta: bidMeta,
			})
		}

	}

	// set bidder currency, EUR by default
	if nativeryResponse.Cur != "" {
		bidderResponse.Currency = nativeryResponse.Cur
	} else {
		bidderResponse.Currency = "EUR"
	}
	return bidderResponse, errs

}

// getMediaTypeForBid switch nativery type in bid type.
func getMediaTypeForBid(bid *bidExt) (openrtb_ext.BidType, error) {
	switch bid.Nativery.BidType {
	case "native":
		return openrtb_ext.BidTypeNative, nil
	case "display", "banner", "rich_media":
		return openrtb_ext.BidTypeBanner, nil
	case "video":
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unrecognized bid_ad_media_type in response from nativery: %s", bid.Nativery.BidType)
	}
}

func convertIntToBoolean(num *int) bool {
	var b bool
	// Dereferenzia num usando *
	if num != nil && *num == 1 {
		b = true
	} else {
		b = false
	}
	return b
}

func buildBidMeta(mediaType string, advDomain []string) *openrtb_ext.ExtBidPrebidMeta {

	//advertiserDomains and dchain are encouraged to implements
	return &openrtb_ext.ExtBidPrebidMeta{
		MediaType:         mediaType,
		AdvertiserDomains: advDomain,
		/*
			DChain: json.RawMessage{} ,
			Cosa include Dchain:
				nodes: Un array di oggetti che rappresentano i diversi partecipanti alla catena di domanda.
				complete: Un flag che indica se la catena di domanda è completa (1) o incompleta (0).
				ver: La versione del modulo Dchain utilizzato.
		*/
	}
}

func splitRequests(imps []openrtb2.Imp, request *openrtb2.BidRequest, requestExt map[string]json.RawMessage, requestExtNativery bidReqExtNativery, uri string) ([]*adapters.RequestData, []error) {
	var errs []error

	resArr := make([]*adapters.RequestData, 0, 1)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	nativeryExtJson, err := json.Marshal(requestExtNativery)
	if err != nil {
		errs = append(errs, err)
	}

	requestExtClone := maps.Clone(requestExt)
	requestExtClone["nativery"] = nativeryExtJson

	request.Ext, err = json.Marshal(requestExtClone)
	if err != nil {
		errs = append(errs, err)
	}

	for _, imp := range imps {
		impsForReq := []openrtb2.Imp{imp}
		request.Imp = impsForReq

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}

		resArr = append(resArr, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		})
	}
	return resArr, errs
}

func getRequestExt(ext json.RawMessage) (map[string]json.RawMessage, error) {
	extMap := make(map[string]json.RawMessage)

	if len(ext) > 0 {
		if err := json.Unmarshal(ext, &extMap); err != nil {
			return nil, err
		}
	}

	return extMap, nil
}

func getNativeryExt(extMap map[string]json.RawMessage, isAMP int, widgetId string) (bidReqExtNativery, error) {
	var nativeryExt bidReqExtNativery

	// if ext.nativery already exists return it
	if nativeryExtJson, exists := extMap["nativery"]; exists && len(nativeryExtJson) > 0 {
		if err := json.Unmarshal(nativeryExtJson, &nativeryExt); err != nil {
			return nativeryExt, err
		}
	}

	nativeryExt.IsAMP = convertIntToBoolean(&isAMP)
	nativeryExt.WidgetId = widgetId

	return nativeryExt, nil
}
