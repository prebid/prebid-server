package matterfull

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	EndpointTemplate *template.Template
}

// MakeRequests prepares request information for prebid-server core
func (adapter *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error

	pub2impressions, imps, err := getImpressionsInfo(request.Imp)
	if len(imps) == 0 {
		return nil, err
	}
	errs = append(errs, err...)

	if len(pub2impressions) == 0 {
		return nil, errs
	}

	result := make([]*adapters.RequestData, 0, len(pub2impressions))
	for k, imps := range pub2impressions {
		bidRequest, err := adapter.buildAdapterRequest(request, &k, imps)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		result = append(result, bidRequest)
	}
	return result, errs
}

// getImpressionsInfo checks each impression for validity and returns impressions copy with corresponding exts
func getImpressionsInfo(imps []openrtb2.Imp) (map[openrtb_ext.ExtImpMatterfull][]openrtb2.Imp, []openrtb2.Imp, []error) {
	var errors []error
	resImps := make([]openrtb2.Imp, 0, len(imps))
	res := make(map[openrtb_ext.ExtImpMatterfull][]openrtb2.Imp)

	for imp := range iterutil.SlicePointerValues(imps) {
		impExt, err := getImpressionExt(imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		// Dispatch impressions.
		// Group impressions by the Matterfull-specific parameter `pid`.
		if err := compatImpression(imp); err != nil {
			errors = append(errors, err)
			continue
		}

		res[impExt] = append(res[impExt], *imp)
		resImps = append(resImps, *imp)
	}
	return res, resImps, errors
}

// Alter impression info to comply with Matterfull platform requirements
func compatImpression(imp *openrtb2.Imp) error {
	imp.Ext = nil //do not forward ext to Matterfull platform
	if imp.Banner != nil {
		return compatBannerImpression(imp)
	}
	return nil
}

func compatBannerImpression(imp *openrtb2.Imp) error {
	if imp.Banner.W == nil || imp.Banner.H == nil {
		bannerCopy := *imp.Banner
		banner := &bannerCopy

		if len(banner.Format) == 0 {
			return &errortypes.BadInput{
				Message: "Expected at least one banner.format entry or explicit w/h",
			}
		}

		format := banner.Format[0]

		// Deep copy the remaining formats
		if len(banner.Format) > 1 {
			newFormats := make([]openrtb2.Format, len(banner.Format)-1)
			copy(newFormats, banner.Format[1:])
			banner.Format = newFormats
		} else {
			banner.Format = nil
		}

		banner.W = &format.W
		banner.H = &format.H

		imp.Banner = banner
	}
	return nil
}

func getImpressionExt(imp *openrtb2.Imp) (openrtb_ext.ExtImpMatterfull, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return openrtb_ext.ExtImpMatterfull{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var matterfullExt openrtb_ext.ExtImpMatterfull
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &matterfullExt); err != nil {
		return openrtb_ext.ExtImpMatterfull{}, &errortypes.BadInput{
			Message: err.Error(),
		}
	}

	// Runtime validation: pid (PublisherID) must not be empty
	if matterfullExt.PublisherID == "" {
		return openrtb_ext.ExtImpMatterfull{}, &errortypes.BadInput{
			Message: "matterfull bidder requires non-empty publisher_id",
		}
	}

	return matterfullExt, nil
}

func (adapter *adapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpMatterfull, imps []openrtb2.Imp) (*adapters.RequestData, error) {
	newBidRequest := createBidRequest(prebidBidRequest, imps)
	reqJSON, err := jsonutil.Marshal(newBidRequest)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.6")

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

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, imps []openrtb2.Imp) *openrtb2.BidRequest {

	reqCopy := *prebidBidRequest
	newBidRequest := &reqCopy
	newBidRequest.Imp = imps

	if newBidRequest.Site != nil {
		siteCopy := *newBidRequest.Site
		newBidRequest.Site = &siteCopy
		newBidRequest.Site.Publisher = nil
		newBidRequest.Site.Domain = ""
	}

	if newBidRequest.App != nil {
		appCopy := *newBidRequest.App
		newBidRequest.App = &appCopy
		newBidRequest.App.Publisher = nil
	}

	return newBidRequest
}

// Builds endpoint url based on adapter-specific pub settings from imp.ext
func (adapter *adapter) buildEndpointURL(params *openrtb_ext.ExtImpMatterfull) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: params.PublisherID}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

// MakeBids translates Matterfull bid response to prebid-server specific format
func (adapter *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		msg := fmt.Sprintf("Bad server response: %v", err)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponse()
	for _, seatBid := range bidResp.SeatBid {
		for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: getMediaTypeForImpID(bid.ImpID, internalRequest.Imp),
			})
		}
	}

	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for
func getMediaTypeForImpID(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for imp := range iterutil.SlicePointerValues(imps) {
		if imp.ID == impID && imp.Video != nil {
			return openrtb_ext.BidTypeVideo
		}
	}
	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of the Matterfull adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	urlTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		EndpointTemplate: urlTemplate,
	}
	return bidder, nil
}
