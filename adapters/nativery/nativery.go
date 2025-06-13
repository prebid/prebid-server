package nativery

import (
	"fmt"
	"maps"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

type bidReqExtNativery struct {
	IsAMP    bool   `json:"isAmp"`
	WidgetId string `json:"widgetId"`
}

type bidExtNativery struct {
	BidType       string   `json:"bid_ad_media_type"`
	BidAdvDomains []string `json:"bid_adv_domains"`

	AdvertiserId  string `json:"adv_id,omitempty"`
	BrandCategory int    `json:"brand_category_id,omitempty"`
}

type bidExt struct {
	Nativery bidExtNativery `json:"nativery"`
}

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
	validImps := make([]openrtb2.Imp, 0, len(request.Imp))
	for i, imp := range request.Imp {
		nativeryExt, err := buildNativeryExt(&imp)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		// at the first impression set widgetId value
		if i == 0 {
			widgetId = nativeryExt.WidgetId
		}

		validImps = append(validImps, imp)
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
	adapterRequests, splitErrors := splitRequests(reqCopy.Imp, &reqCopy, reqExt, reqExtNativery, a.endpoint)

	return adapterRequests, append(errs, splitErrors...)
}

func buildNativeryExt(imp *openrtb2.Imp) (openrtb_ext.ImpExtNativery, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ImpExtNativery{}, err
	}

	var nativeryExt openrtb_ext.ImpExtNativery
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &nativeryExt); err != nil {
		return openrtb_ext.ImpExtNativery{}, err
	}

	return nativeryExt, nil
}

// makebids handles the entire bidding process for a single BidRequest.
// It creates and sends bid requests to multiple ad exchanges, receives
// and parses responses, extracts bids and other relevant information,
// and populates a BidderResponse object with the aggregated information.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// check if the response has no content
	if adapters.IsResponseStatusCodeNoContent(response) {
		// Extract nativery no content reason if is present
		nativeryError := response.Headers.Get("X-Nativery-Error")
		if nativeryError != "" {
			return nil, []error{&errortypes.BadInput{
				Message: fmt.Sprintf("Nativery Error: %s.", nativeryError),
			}}
		}

		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("No Content"),
		}}
	}

	// check if the response has errors
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{err}
	}

	// handle response
	var nativeryResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &nativeryResponse); err != nil {
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
			if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err != nil {
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

func buildBidMeta(mediaType string, advDomain []string) *openrtb_ext.ExtBidPrebidMeta {

	//advertiserDomains and dchain are encouraged to implements
	return &openrtb_ext.ExtBidPrebidMeta{
		MediaType:         mediaType,
		AdvertiserDomains: advDomain,
	}
}

// splitRequests creates one HTTP request per Imp by deep-copying the original BidRequest
func splitRequests(
	imps []openrtb2.Imp,
	request *openrtb2.BidRequest,
	requestExt map[string]jsonutil.RawMessage,
	requestExtNativery bidReqExtNativery,
	uri string,
) ([]*adapters.RequestData, []error) {
	var errs []error

	// Pre-allocate slice to hold one RequestData per imp
	resArr := make([]*adapters.RequestData, 0, len(imps))

	// Prepare standard headers for all requests
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	// Marshal the nativery-specific extension once
	nativeryExtJson, err := jsonutil.Marshal(requestExtNativery)
	if err != nil {
		errs = append(errs, err)
	}

	// Make a shallow copy of the original request struct to use as a template
	baseReq := *request

	for _, imp := range imps {
		// Clone the bidder-level ext map and inject the nativery JSON
		extClone := maps.Clone(requestExt)
		extClone["nativery"] = nativeryExtJson

		// Start from the base request copy for this imp
		reqCopy := baseReq

		// Marshal the cloned ext back into JSON bytes
		reqCopy.Ext, err = jsonutil.Marshal(extClone)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Replace the Imp array with a single-element slice for this imp
		reqCopy.Imp = []openrtb2.Imp{imp}

		// Serialize this per-imp request to JSON
		reqJSON, err := jsonutil.Marshal(&reqCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Build the RequestData for this imp
		resArr = append(resArr, &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
		})
	}

	return resArr, errs
}

func getRequestExt(ext jsonutil.RawMessage) (map[string]jsonutil.RawMessage, error) {
	extMap := make(map[string]jsonutil.RawMessage)

	if len(ext) > 0 {
		if err := jsonutil.Unmarshal(ext, &extMap); err != nil {
			return nil, err
		}
	}

	return extMap, nil
}

func getNativeryExt(extMap map[string]jsonutil.RawMessage, isAMP int, widgetId string) (bidReqExtNativery, error) {
	var nativeryExt bidReqExtNativery

	// if ext.nativery already exists return it
	if nativeryExtJson, exists := extMap["nativery"]; exists && len(nativeryExtJson) > 0 {
		if err := jsonutil.Unmarshal(nativeryExtJson, &nativeryExt); err != nil {
			return nativeryExt, err
		}
	}

	nativeryExt.IsAMP = isAMP == 1
	nativeryExt.WidgetId = widgetId

	return nativeryExt, nil
}
