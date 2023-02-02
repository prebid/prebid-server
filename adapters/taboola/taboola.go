package taboola

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"text/template"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the Foo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}
	bidder := &adapter{
		endpoint: endpointTemplate,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var requests []*adapters.RequestData

	taboolaRequest, errs := createTaboolaRequest(request)
	if len(errs) > 0 {
		return nil, errs
	}

	filterNative := func(imp openrtb2.Imp) bool { return imp.Native != nil }
	nativeRequest := createRequestByMediaType(*taboolaRequest, filterNative)
	if len(nativeRequest.Imp) > 0 {
		request, err := a.buildRequest(&nativeRequest, "native")
		if err != nil {
			return nil, []error{fmt.Errorf("unable to build native request %v", err)}
		}
		requests = append(requests, request)
	}

	filterDisplay := func(imp openrtb2.Imp) bool { return imp.Banner != nil }
	displayRequest := createRequestByMediaType(*taboolaRequest, filterDisplay)
	if len(displayRequest.Imp) > 0 {
		request, err := a.buildRequest(&displayRequest, "display")
		if err != nil {
			return nil, []error{fmt.Errorf("unable to build display request %v", err)}
		}
		requests = append(requests, request)
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
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaType(seatBid.Bid[i].ImpID, request.Imp)
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

func (a *adapter) buildRequest(request *openrtb2.BidRequest, mediaType string) (*adapters.RequestData, error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	url, err := a.buildEndpointURL(request.Site.ID, mediaType)
	if err != nil {
		return nil, err
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    url,
		Body:   requestJSON,
	}

	return requestData, nil
}

// Builds endpoint url based on adapter-specific pub settings from imp.ext
func (a *adapter) buildEndpointURL(publisherId string, mediaType string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: publisherId, MediaType: mediaType}
	resolvedUrl, err := macros.ResolveMacros(a.endpoint, endpointParams)
	if err != nil {
		return "", err
	}
	return resolvedUrl, nil
}

func createTaboolaRequest(request *openrtb2.BidRequest) (taboolaRequest *openrtb2.BidRequest, errors []error) {
	modifiedRequest := *request
	var errs []error

	var taboolaExt openrtb_ext.ImpExtTaboola
	for i := 0; i < len(modifiedRequest.Imp); i++ {
		imp := modifiedRequest.Imp[i]

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := json.Unmarshal(bidderExt.Bidder, &taboolaExt); err != nil {
			errs = append(errs, err)
			continue
		}
		if taboolaExt.TagId != "" {
			imp.TagID = taboolaExt.TagId
			modifiedRequest.Imp[i] = imp
		}
		if taboolaExt.BidFloor != 0 {
			imp.BidFloor = taboolaExt.BidFloor
			modifiedRequest.Imp[i] = imp
		}
	}

	publisher := &openrtb2.Publisher{
		ID: taboolaExt.PublisherId,
	}

	if modifiedRequest.Site == nil {
		newSite := &openrtb2.Site{
			ID:        taboolaExt.PublisherId,
			Name:      taboolaExt.PublisherId,
			Domain:    evaluateDomain(taboolaExt.PublisherDomain, request),
			Publisher: publisher,
		}
		modifiedRequest.Site = newSite
	} else {
		modifiedSite := *modifiedRequest.Site
		modifiedSite.Publisher = publisher
		modifiedSite.ID = taboolaExt.PublisherId
		modifiedSite.Name = taboolaExt.PublisherId
		modifiedSite.Domain = evaluateDomain(taboolaExt.PublisherDomain, request)
		modifiedRequest.Site = &modifiedSite
	}

	if taboolaExt.BCat != nil {
		modifiedRequest.BCat = taboolaExt.BCat
	}

	if taboolaExt.BAdv != nil {
		modifiedRequest.BAdv = taboolaExt.BAdv
	}

	return &modifiedRequest, errs
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

func filter[T any](ss []T, f func(T) bool) (ret []T) {
	for _, s := range ss {
		if f(s) {
			ret = append(ret, s)
		}
	}
	return
}

func createRequestByMediaType(originBidRequest openrtb2.BidRequest, f func(imp openrtb2.Imp) bool) (bidRequest openrtb2.BidRequest) {
	filteredImps := filter(originBidRequest.Imp, f)
	bidRequest = originBidRequest
	bidRequest.Imp = filteredImps
	return bidRequest
}
