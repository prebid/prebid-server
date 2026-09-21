package aps

import (
	"encoding/json"
	"fmt"
	"net/http"
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

const adapterVersion = "1.0.0"

const defaultRegion = "na"

var validRegions = map[string]struct{}{"na": {}, "eu": {}, "fe": {}}

type adapter struct {
	endpoint *template.Template
}

type apsSDKExt struct {
	Version string `json:"version"`
	Source  string `json:"source"`
}

// Builder builds a new instance of the Aps adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpoint, err := template.New("apsEndpoint").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse APS endpoint template: %v", err)
	}
	return &adapter{endpoint: endpoint}, nil
}

// MakeRequests builds the APS request. Sample OpenRTB request this adapter accepts (dummy values):
//
//	{
//	  "id": "req-1",
//	  "test": 1,
//	  "site": {"page": "https://example.com"},
//	  "imp": [
//	    {"id": "imp-1", "banner": {"format": [{"w": 300, "h": 250}]},
//	     "ext": {"prebid": {"bidder": {"aps": {"accountID": "1234", "region": "na"}}}}},
//	    {"id": "imp-2", "video": {"mimes": ["video/mp4"], "w": 640, "h": 480, "protocols": [2, 3, 5, 6]},
//	     "ext": {"prebid": {"bidder": {"aps": {"accountID": "1234"}}}}}
//	  ]
//	}
//
// accountID is a per-imp bidder param (or set once via ext.prebid.bidderparams.aps.accountID); all imps
// must resolve to the same account. region is an optional per-imp param (default na); it fills the
// endpoint macro and is validated against a fixed set of valid regions (na, eu, fe); others are rejected.
// test:1 appends amzn_debug_mode=1 to the endpoint.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if request.App != nil {
		return nil, []error{&errortypes.BadInput{
			Message: "the APS adapter supports web (site) inventory only; app requests are not supported",
		}}
	}

	var errors []error
	validImps := make([]openrtb2.Imp, 0, len(request.Imp))
	accountID := ""
	region := defaultRegion

	for _, imp := range request.Imp {
		sanitizedImp, err := sanitizeMediaTypes(imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		impAccountID, impRegion, err := parseImpParams(imp)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		if accountID == "" {
			accountID = impAccountID
		} else if accountID != impAccountID {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("imp %q: all imps in a request must use the same APS accountID (got %q and %q)",
					imp.ID, accountID, impAccountID),
			})
			return nil, errors
		}
		region = impRegion

		if sanitizedImp.Banner != nil {
			sanitizedImp = backfillBannerSize(sanitizedImp)
		}
		validImps = append(validImps, sanitizedImp)
	}

	if len(validImps) == 0 {
		return nil, errors
	}

	data, err := a.makeRequest(*request, validImps, accountID, region)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	return []*adapters.RequestData{data}, errors
}

func parseImpParams(imp openrtb2.Imp) (accountID string, region string, err error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp %q: invalid imp.ext: %s", imp.ID, err)}
	}

	var apsExt openrtb_ext.ExtImpAps
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &apsExt); err != nil {
		return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp %q: invalid aps bidder params: %s", imp.ID, err)}
	}

	accountID = strings.TrimSpace(apsExt.AccountID)
	if accountID == "" {
		return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp %q: the APS bidder param \"accountID\" is required", imp.ID)}
	}

	region = strings.TrimSpace(apsExt.Region)
	if region == "" {
		region = defaultRegion
	}
	if _, ok := validRegions[region]; !ok {
		return "", "", &errortypes.BadInput{Message: fmt.Sprintf("imp %q: invalid APS region %q", imp.ID, region)}
	}
	return accountID, region, nil
}

func (a *adapter) makeRequest(request openrtb2.BidRequest, imps []openrtb2.Imp, accountID string, region string) (*adapters.RequestData, error) {
	request.Imp = imps

	sanitizeUserObject(&request)
	sanitizeDeviceObject(&request)

	if len(request.Cur) == 0 {
		request.Cur = []string{"USD"}
	}

	if err := applyRequestExt(&request, accountID); err != nil {
		return nil, err
	}

	requestJSON, err := jsonutil.Marshal(&request)
	if err != nil {
		return nil, err
	}

	uri, err := macros.ResolveMacros(a.endpoint, macros.EndpointTemplateParams{Region: region})
	if err != nil {
		return nil, err
	}
	if request.Test == 1 {
		uri = appendDebugMode(uri)
	}

	return &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    uri,
		Body:   requestJSON,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
	}, nil
}

// appendDebugMode appends the amzn_debug_mode=1 query param for test requests (bidRequest.test == 1).
func appendDebugMode(endpoint string) string {
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	return endpoint + separator + "amzn_debug_mode=1"
}

func sanitizeMediaTypes(imp openrtb2.Imp) (openrtb2.Imp, error) {
	if imp.Banner == nil && imp.Video == nil {
		return imp, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %q: the APS adapter supports only banner and video media types", imp.ID),
		}
	}
	imp.Audio = nil
	imp.Native = nil
	return imp, nil
}

func backfillBannerSize(imp openrtb2.Imp) openrtb2.Imp {
	banner := imp.Banner
	if len(banner.Format) == 0 {
		return imp
	}
	if banner.W != nil && banner.H != nil {
		return imp
	}

	format := banner.Format[0]
	w := format.W
	h := format.H
	modifiedBanner := *banner
	modifiedBanner.W = &w
	modifiedBanner.H = &h
	imp.Banner = &modifiedBanner
	return imp
}

func sanitizeUserObject(request *openrtb2.BidRequest) {
	if request.User != nil {
		modifiedUser := *request.User
		modifiedUser.Gender = ""
		modifiedUser.Yob = 0
		modifiedUser.CustomData = ""
		modifiedUser.Geo = nil
		request.User = &modifiedUser
	}
}

func sanitizeDeviceObject(request *openrtb2.BidRequest) {
	if request.Device != nil && request.Device.Geo != nil {
		modifiedGeo := *request.Device.Geo
		modifiedGeo.Lat = nil
		modifiedGeo.Lon = nil
		modifiedDevice := *request.Device
		modifiedDevice.Geo = &modifiedGeo
		request.Device = &modifiedDevice
	}
}

func applyRequestExt(request *openrtb2.BidRequest, accountID string) error {
	extMap := map[string]json.RawMessage{}
	if len(request.Ext) > 0 {
		if err := jsonutil.Unmarshal(request.Ext, &extMap); err != nil {
			return &errortypes.BadInput{Message: err.Error()}
		}
	}

	account, err := jsonutil.Marshal(accountID)
	if err != nil {
		return err
	}
	extMap["account"] = account

	sdk, err := jsonutil.Marshal(apsSDKExt{Version: adapterVersion, Source: "prebid-server"})
	if err != nil {
		return err
	}
	extMap["sdk"] = sdk

	ext, err := jsonutil.Marshal(extMap)
	if err != nil {
		return err
	}
	request.Ext = ext
	return nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher.",
		}}
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errors = append(errors, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errors
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unsupported MType %d for impression %q", bid.MType, bid.ImpID),
		}
	}
}
