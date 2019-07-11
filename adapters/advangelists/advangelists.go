package advangelists

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdvangelistsAdapter struct {
	EndpointTemplate template.Template
}

//MakeRequests prepares request information for prebid-server core
func (adapter *AdvangelistsAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "No impression in the bid request"})
		return nil, errs
	}
	imps, impExts, err := getImpressionsInfo(request.Imp)
	if len(imps) == 0 {
		return nil, err
	}
	errs = append(errs, err...)

	pub2impressions, dispErrors := dispatchImpressions(imps, impExts)
	if len(pub2impressions) == 0 {
		return nil, errs
	}
	errs = append(errs, dispErrors...)

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
func getImpressionsInfo(imps []openrtb.Imp) ([]openrtb.Imp, []openrtb_ext.ExtImpAdvangelists, []error) {
	impsCount := len(imps)
	errors := make([]error, 0, impsCount)
	resImps := make([]openrtb.Imp, 0, impsCount)
	resImpExts := make([]openrtb_ext.ExtImpAdvangelists, 0, impsCount)

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

func validateImpression(imp *openrtb.Imp, impExt *openrtb_ext.ExtImpAdvangelists) error {
	if impExt.PubID == "" {
		return &errortypes.BadInput{Message: "No pubid value provided"}
	}
	return nil
}

//Group impressions by advangelists-specific parameters `pubid
func dispatchImpressions(imps []openrtb.Imp, impsExt []openrtb_ext.ExtImpAdvangelists) (map[openrtb_ext.ExtImpAdvangelists][]openrtb.Imp, []error) {
	res := make(map[openrtb_ext.ExtImpAdvangelists][]openrtb.Imp)
	errors := make([]error, 0)
	for idx := range imps {
		imp := imps[idx]
		err := compatImpression(&imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		impExt := impsExt[idx]
		if res[impExt] == nil {
			res[impExt] = make([]openrtb.Imp, 0)
		}
		res[impExt] = append(res[impExt], imp)

	}
	return res, errors
}

//Alter impression info to comply with advangelists platform requirements
func compatImpression(imp *openrtb.Imp) error {
	imp.Ext = nil //do not forward ext to advangelists platform
	if imp.Banner != nil {
		return compatBannerImpression(imp)
	}
	return nil
}

func compatBannerImpression(imp *openrtb.Imp) error {
	// Create a copy of the banner, since imp is a shallow copy of the original.
	bannerCopy := *imp.Banner
	banner := &bannerCopy
	//As banner.w/h are required fields for advangelistsAdn platform - take the first format entry
	if banner.W == nil && banner.H == nil {
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

func getImpressionExt(imp *openrtb.Imp) (*openrtb_ext.ExtImpAdvangelists, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var advangelistsExt openrtb_ext.ExtImpAdvangelists
	if err := json.Unmarshal(bidderExt.Bidder, &advangelistsExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	return &advangelistsExt, nil
}

func (adapter *AdvangelistsAdapter) buildAdapterRequest(prebidBidRequest *openrtb.BidRequest, params *openrtb_ext.ExtImpAdvangelists, imps []openrtb.Imp) (*adapters.RequestData, error) {
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

func createBidRequest(prebidBidRequest *openrtb.BidRequest, params *openrtb_ext.ExtImpAdvangelists, imps []openrtb.Imp) *openrtb.BidRequest {
	bidRequest := *prebidBidRequest
	bidRequest.Imp = imps
	for idx := range bidRequest.Imp {
		imp := &bidRequest.Imp[idx]
		imp.TagID = params.Placement
		fmt.Printf(imp.TagID)
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
func (adapter *AdvangelistsAdapter) buildEndpointURL(params *openrtb_ext.ExtImpAdvangelists) (string, error) {
	endpointParams := macros.EndpointTemplateParams{PubID: params.PubID}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

//MakeBids translates advangelists bid response to prebid-server specific format
func (adapter *AdvangelistsAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var msg = ""
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		msg = fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)
		return nil, []error{&errortypes.BadServerResponse{Message: msg}}
	}
	var bidResp openrtb.BidResponse
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
func getMediaTypeForImpID(impID string, imps []openrtb.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID && imp.Video != nil {
			return openrtb_ext.BidTypeVideo
		}
	}
	return openrtb_ext.BidTypeBanner
}

// NewAdvangelistsAdapter to be called in prebid-server core to create advangelists adapter instance
func NewAdvangelistsBidder(endpointTemplate string) adapters.Bidder {
	template, err := template.New("endpointTemplate").Parse(endpointTemplate)
	if err != nil {
		glog.Fatal("Unable to parse endpoint url template")
		return nil
	}
	return &AdvangelistsAdapter{EndpointTemplate: *template}
}
