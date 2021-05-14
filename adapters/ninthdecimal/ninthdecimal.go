package ninthdecimal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type NinthDecimalAdapter struct {
	EndpointTemplate template.Template
}

//MakeRequests prepares request information for prebid-server core
func (adapter *NinthDecimalAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "No impression in the bid request"})
		return nil, errs
	}
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
			return nil, errs
		} else {
			result = append(result, bidRequest)
		}
	}
	return result, errs
}

// getImpressionsInfo checks each impression for validity and returns impressions copy with corresponding exts
func getImpressionsInfo(imps []openrtb2.Imp) (map[openrtb_ext.ExtImpNinthDecimal][]openrtb2.Imp, []openrtb2.Imp, []error) {
	errors := make([]error, 0, len(imps))
	resImps := make([]openrtb2.Imp, 0, len(imps))
	res := make(map[openrtb_ext.ExtImpNinthDecimal][]openrtb2.Imp)

	for _, imp := range imps {
		impExt, err := getImpressionExt(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if err := validateImpression(impExt); err != nil {
			errors = append(errors, err)
			continue
		}
		//dispatchImpressions
		//Group impressions by NinthDecimal-specific parameters `pubid
		if err := compatImpression(&imp); err != nil {
			errors = append(errors, err)
			continue
		}
		if res[*impExt] == nil {
			res[*impExt] = make([]openrtb2.Imp, 0)
		}
		res[*impExt] = append(res[*impExt], imp)
		resImps = append(resImps, imp)
	}
	return res, resImps, errors
}

func validateImpression(impExt *openrtb_ext.ExtImpNinthDecimal) error {
	if impExt.PublisherID == "" {
		return &errortypes.BadInput{Message: "No pubid value provided"}
	}
	return nil
}

//Alter impression info to comply with NinthDecimal platform requirements
func compatImpression(imp *openrtb2.Imp) error {
	imp.Ext = nil //do not forward ext to NinthDecimal platform
	if imp.Banner != nil {
		return compatBannerImpression(imp)
	}
	return nil
}

func compatBannerImpression(imp *openrtb2.Imp) error {
	// Create a copy of the banner, since imp is a shallow copy of the original.

	bannerCopy := *imp.Banner
	banner := &bannerCopy
	//As banner.w/h are required fields for NinthDecimal platform - take the first format entry
	if banner.W == nil || banner.H == nil {
		if len(banner.Format) == 0 {
			return &errortypes.BadInput{Message: "Expected at least one banner.format entry or explicit w/h"}
		}
		format := banner.Format[0]
		banner.Format = banner.Format[1:]
		banner.W = &format.W
		banner.H = &format.H
		imp.Banner = banner
	}
	return nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpNinthDecimal, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var NinthDecimalExt openrtb_ext.ExtImpNinthDecimal
	if err := json.Unmarshal(bidderExt.Bidder, &NinthDecimalExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	return &NinthDecimalExt, nil
}

func (adapter *NinthDecimalAdapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpNinthDecimal, imps []openrtb2.Imp) (*adapters.RequestData, error) {
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
		Headers: headers}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpNinthDecimal, imps []openrtb2.Imp) *openrtb2.BidRequest {
	bidRequest := *prebidBidRequest
	bidRequest.Imp = imps
	for idx := range bidRequest.Imp {
		imp := &bidRequest.Imp[idx]
		imp.TagID = params.Placement
	}
	if bidRequest.Site != nil {
		// Need to copy Site as Request is a shallow copy
		siteCopy := *bidRequest.Site
		bidRequest.Site = &siteCopy
		bidRequest.Site.Publisher = nil
		bidRequest.Site.Domain = ""
	}
	if bidRequest.App != nil {
		// Need to copy App as Request is a shallow copy
		appCopy := *bidRequest.App
		bidRequest.App = &appCopy
		bidRequest.App.Publisher = nil
	}
	return &bidRequest
}

// Builds enpoint url based on adapter-specific pub settings from imp.ext
func (adapter *NinthDecimalAdapter) buildEndpointURL(params *openrtb_ext.ExtImpNinthDecimal) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PublisherID: params.PublisherID}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

//MakeBids translates NinthDecimal bid response to prebid-server specific format
func (adapter *NinthDecimalAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var msg = ""
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		msg = fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}

	}
	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		msg = fmt.Sprintf("Bad server response: %d", err)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}
	if len(bidResp.SeatBid) != 1 {
		var msg = fmt.Sprintf("Invalid SeatBids count: %d", len(bidResp.SeatBid))
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}

	seatBid := bidResp.SeatBid[0]
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	for i := 0; i < len(seatBid.Bid); i++ {
		bid := seatBid.Bid[i]
		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: getMediaTypeForImpID(bid.ImpID, internalRequest.Imp),
		})
	}
	return bidResponse, nil
}

// getMediaTypeForImp figures out which media type this bid is for
func getMediaTypeForImpID(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID && imp.Video != nil {
			return openrtb_ext.BidTypeVideo
		}
	}
	return openrtb_ext.BidTypeBanner
}

// Builder builds a new instance of the NinthDecimal adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &NinthDecimalAdapter{
		EndpointTemplate: *template,
	}
	return bidder, nil
}
