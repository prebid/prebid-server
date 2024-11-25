package taboola

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
	gvlID    string
}

// Builder builds a new instance of Taboola adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	gvlID := ""
	if server.GvlID > 0 {
		gvlID = strconv.Itoa(server.GvlID)
	}

	bidder := &adapter{
		endpoint: endpointTemplate,
		gvlID:    gvlID,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var requests []*adapters.RequestData

	taboolaRequests, errs := createTaboolaRequests(request)
	if len(errs) > 0 {
		return nil, errs
	}

	for _, taboolaRequest := range taboolaRequests {
		if len(taboolaRequest.Imp) > 0 {
			request, err := a.buildRequest(taboolaRequest)
			if err != nil {
				return nil, []error{fmt.Errorf("unable to build request %v", err)}
			}
			requests = append(requests, request)
		}
	}

	return requests, errs
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errs []error

	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaType(seatBid.Bid[i].ImpID, request.Imp)
			resolveMacros(&seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}
	return bidResponse, errs
}

func (a *adapter) buildRequest(request *openrtb2.BidRequest) (*adapters.RequestData, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	const (
		NATIVE_ENDPOINT_PREFIX  = "native"
		DISPLAY_ENDPOINT_PREFIX = "display"
	)

	//set MediaType based on first imp
	var mediaType string
	if request.Imp[0].Banner != nil {
		mediaType = DISPLAY_ENDPOINT_PREFIX
	} else if request.Imp[0].Native != nil {
		mediaType = NATIVE_ENDPOINT_PREFIX
	} else {
		return nil, fmt.Errorf("unsupported media type for imp: %v", request.Imp[0])
	}

	var taboolaPublisherId string
	if request.Site != nil && request.Site.ID != "" {
		taboolaPublisherId = request.Site.ID
	} else if request.App != nil && request.App.ID != "" {
		taboolaPublisherId = request.App.ID
	}

	url, err := a.buildEndpointURL(taboolaPublisherId, mediaType)
	if err != nil {
		return nil, err
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    url,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}

	return requestData, nil
}

// Builds endpoint url based on adapter-specific pub settings from imp.ext
func (a *adapter) buildEndpointURL(publisherId string, mediaType string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: publisherId, MediaType: mediaType, GvlID: a.gvlID}
	resolvedUrl, err := macros.ResolveMacros(a.endpoint, endpointParams)
	if err != nil {
		return "", err
	}
	return resolvedUrl, nil
}

func createTaboolaRequests(request *openrtb2.BidRequest) (taboolaRequests []*openrtb2.BidRequest, errors []error) {
	modifiedRequest := *request
	var nativeImp []openrtb2.Imp
	var bannerImp []openrtb2.Imp
	var errs []error

	var taboolaExt openrtb_ext.ImpExtTaboola
	for i := 0; i < len(modifiedRequest.Imp); i++ {
		imp := modifiedRequest.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &taboolaExt); err != nil {
			errs = append(errs, err)
			continue
		}

		tagId := taboolaExt.TagID
		if len(taboolaExt.TagID) < 1 {
			tagId = taboolaExt.TagId
		}

		imp.TagID = tagId
		modifiedRequest.Imp[i] = imp

		if taboolaExt.BidFloor != 0 {
			imp.BidFloor = taboolaExt.BidFloor
			modifiedRequest.Imp[i] = imp
		}

		if modifiedRequest.Imp[i].Banner != nil {
			if taboolaExt.Position != nil {
				bannerCopy := *imp.Banner
				bannerCopy.Pos = adcom1.PlacementPosition(*taboolaExt.Position).Ptr()
				imp.Banner = &bannerCopy
				modifiedRequest.Imp[i] = imp
			}
			bannerImp = append(bannerImp, modifiedRequest.Imp[i])
		} else if modifiedRequest.Imp[i].Native != nil {
			nativeImp = append(nativeImp, modifiedRequest.Imp[i])
		}

	}

	publisher := &openrtb2.Publisher{
		ID: taboolaExt.PublisherId,
	}

	if modifiedRequest.Site != nil {
		modifiedSite := *modifiedRequest.Site
		modifiedSite.ID = taboolaExt.PublisherId
		modifiedSite.Name = taboolaExt.PublisherId
		modifiedSite.Domain = evaluateDomain(taboolaExt.PublisherDomain, request)
		modifiedSite.Publisher = publisher
		modifiedRequest.Site = &modifiedSite
	}
	if modifiedRequest.App != nil {
		modifiedApp := *modifiedRequest.App
		modifiedApp.ID = taboolaExt.PublisherId
		modifiedApp.Publisher = publisher
		modifiedRequest.App = &modifiedApp
	}

	if taboolaExt.BCat != nil {
		modifiedRequest.BCat = taboolaExt.BCat
	}

	if taboolaExt.BAdv != nil {
		modifiedRequest.BAdv = taboolaExt.BAdv
	}

	if taboolaExt.PageType != "" {
		requestExt, requestExtErr := makeRequestExt(taboolaExt.PageType)
		if requestExtErr == nil {
			modifiedRequest.Ext = requestExt
		} else {
			errs = append(errs, requestExtErr)
		}
	}

	taboolaRequests = append(taboolaRequests, overrideBidRequestImp(&modifiedRequest, nativeImp))
	taboolaRequests = append(taboolaRequests, overrideBidRequestImp(&modifiedRequest, bannerImp))

	return taboolaRequests, errs
}

func makeRequestExt(pageType string) (json.RawMessage, error) {
	requestExt := &RequestExt{
		PageType: pageType,
	}

	requestExtJson, err := json.Marshal(requestExt)
	if err != nil {
		return nil, fmt.Errorf("could not marshal %s, err: %s", requestExt, err)
	}
	return requestExtJson, nil

}

func getMediaType(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find banner/native impression \"%s\" ", impID),
	}
}

func evaluateDomain(publisherDomain string, request *openrtb2.BidRequest) (result string) {
	if publisherDomain != "" {
		return publisherDomain
	}
	if request.Site != nil {
		return request.Site.Domain
	}
	return ""
}

func overrideBidRequestImp(originBidRequest *openrtb2.BidRequest, imp []openrtb2.Imp) (bidRequest *openrtb2.BidRequest) {
	bidRequestResult := *originBidRequest
	bidRequestResult.Imp = imp
	return &bidRequestResult
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid != nil {
		price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
		bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
		bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
	}
}
