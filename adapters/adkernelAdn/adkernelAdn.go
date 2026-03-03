package adkernelAdn

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adkernelAdnAdapter struct {
	EndpointTemplate *template.Template
}

// MakeRequests prepares request information for prebid-server core
func (adapter *adkernelAdnAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))
	if len(request.Imp) == 0 {
		errs = append(errs, newBadInputError("No impression in the bid request"))
		return nil, errs
	}
	imps, impExts, err := getImpressionsInfo(request.Imp)
	if len(imps) == 0 {
		return nil, err
	}
	errs = append(errs, err...)

	pub2impressions, dispErrors := dispatchImpressions(imps, impExts)
	if len(dispErrors) > 0 {
		errs = append(errs, dispErrors...)
	}
	if len(pub2impressions) == 0 {
		return nil, errs
	}
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
func getImpressionsInfo(imps []openrtb2.Imp) ([]openrtb2.Imp, []openrtb_ext.ExtImpAdkernelAdn, []error) {
	impsCount := len(imps)
	errors := make([]error, 0, impsCount)
	resImps := make([]openrtb2.Imp, 0, impsCount)
	resImpExts := make([]openrtb_ext.ExtImpAdkernelAdn, 0, impsCount)

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

func validateImpression(imp *openrtb2.Imp, impExt *openrtb_ext.ExtImpAdkernelAdn) error {
	if impExt.PublisherID < 1 {
		return newBadInputError(fmt.Sprintf("Invalid pubId value. Ignoring imp id=%s", imp.ID))
	}
	if imp.Video == nil && imp.Banner == nil {
		return newBadInputError(fmt.Sprintf("Invalid imp with id=%s. Expected imp.banner or imp.video", imp.ID))
	}
	return nil
}

// Group impressions by AdKernel-specific parameters `pubId` & `host`
func dispatchImpressions(imps []openrtb2.Imp, impsExt []openrtb_ext.ExtImpAdkernelAdn) (map[openrtb_ext.ExtImpAdkernelAdn][]openrtb2.Imp, []error) {
	res := make(map[openrtb_ext.ExtImpAdkernelAdn][]openrtb2.Imp)
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
			res[impExt] = make([]openrtb2.Imp, 0)
		}
		res[impExt] = append(res[impExt], imp)
	}
	return res, errors
}

// Alter impression info to comply with adkernel platform requirements
func compatImpression(imp *openrtb2.Imp) error {
	imp.Ext = nil //do not forward ext to adkernel platform
	if imp.Banner != nil {
		return compatBannerImpression(imp)
	}
	if imp.Video != nil {
		return compatVideoImpression(imp)
	}
	return newBadInputError("Unsupported impression has been received")
}

func compatBannerImpression(imp *openrtb2.Imp) error {
	// Create a copy of the banner, since imp is a shallow copy of the original.
	bannerCopy := *imp.Banner
	banner := &bannerCopy
	//As banner.w/h are required fields for adkernelAdn platform - take the first format entry
	if banner.W == nil && banner.H == nil {
		if len(banner.Format) == 0 {
			return newBadInputError("Expected at least one banner.format entry or explicit w/h")
		}
		format := banner.Format[0]
		banner.Format = banner.Format[1:]
		banner.W = &format.W
		banner.H = &format.H
		imp.Banner = banner
	}

	imp.Video = nil
	imp.Native = nil
	imp.Audio = nil

	return nil
}

func compatVideoImpression(imp *openrtb2.Imp) error {
	imp.Banner = nil
	imp.Audio = nil
	imp.Native = nil
	return nil
}

func getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpAdkernelAdn, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	var adkernelAdnExt openrtb_ext.ExtImpAdkernelAdn
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &adkernelAdnExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: err.Error(),
		}
	}
	return &adkernelAdnExt, nil
}

func (adapter *adkernelAdnAdapter) buildAdapterRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpAdkernelAdn, imps []openrtb2.Imp) (*adapters.RequestData, error) {
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
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(imps)}, nil
}

func createBidRequest(prebidBidRequest *openrtb2.BidRequest, params *openrtb_ext.ExtImpAdkernelAdn, imps []openrtb2.Imp) *openrtb2.BidRequest {
	bidRequest := *prebidBidRequest
	bidRequest.Imp = imps

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
func (adapter *adkernelAdnAdapter) buildEndpointURL(params *openrtb_ext.ExtImpAdkernelAdn) (string, error) {
	pubIDStr := strconv.Itoa(params.PublisherID)
	endpointParams := macros.EndpointTemplateParams{PublisherID: pubIDStr}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)
}

// MakeBids translates adkernel bid response to prebid-server specific format
func (adapter *adkernelAdnAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if response.StatusCode != http.StatusOK {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Unexpected http status code: %d", response.StatusCode)),
		}
	}
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Bad server response: %d", err)),
		}
	}
	if len(bidResp.SeatBid) != 1 {
		return nil, []error{
			newBadServerResponseError(fmt.Sprintf("Invalid SeatBids count: %d", len(bidResp.SeatBid))),
		}
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
		if imp.ID == impID && imp.Banner != nil {
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeVideo
}

func newBadInputError(message string) error {
	return &errortypes.BadInput{
		Message: message,
	}
}

func newBadServerResponseError(message string) error {
	return &errortypes.BadServerResponse{
		Message: message,
	}
}

// Builder builds a new instance of the AdkernelAdn adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	template, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adkernelAdnAdapter{
		EndpointTemplate: template,
	}
	return bidder, nil
}
