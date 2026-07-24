package hypelab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
	"github.com/prebid/prebid-server/v4/version"
)

const (
	displayManager = "HypeLab Prebid Server"
	source         = "prebid-server"
)

type adapter struct {
	endpoint string
}

type bidExt struct {
	HypeLab *hypeLabBidExt `json:"hypelab,omitempty"`
}

type hypeLabBidExt struct {
	CreativeType string `json:"creative_type,omitempty"`
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	outgoingRequest, errs := makeOutgoingRequest(request)
	if len(outgoingRequest.Imp) == 0 {
		return nil, errs
	}

	body, err := jsonutil.Marshal(outgoingRequest)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Add("Accept", "application/json")
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("X-OpenRTB-Version", "2.6")

	return []*adapters.RequestData{{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    body,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(outgoingRequest.Imp),
	}}, errs
}

func makeOutgoingRequest(request *openrtb2.BidRequest) (openrtb2.BidRequest, []error) {
	requestCopy := *request
	requestCopy.Imp = make([]openrtb2.Imp, 0, len(request.Imp))

	var errs []error
	for _, imp := range request.Imp {
		updatedImp, err := makeOutgoingImp(imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		requestCopy.Imp = append(requestCopy.Imp, updatedImp)
	}

	if err := setRequestExt(&requestCopy); err != nil {
		errs = append(errs, err)
	}

	return requestCopy, errs
}

func makeOutgoingImp(imp openrtb2.Imp) (openrtb2.Imp, error) {
	params, err := getImpParams(imp)
	if err != nil {
		return imp, err
	}

	imp.TagID = params.PlacementSlug
	imp.DisplayManager = displayManager
	imp.DisplayManagerVer = prebidServerVersion()

	imp.Ext, err = jsonutil.Marshal(map[string]openrtb_ext.ExtImpHypeLab{
		"bidder": params,
	})
	if err != nil {
		return imp, err
	}

	return imp, nil
}

func getImpParams(imp openrtb2.Imp) (openrtb_ext.ExtImpHypeLab, error) {
	var ext adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &ext); err != nil {
		return openrtb_ext.ExtImpHypeLab{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext", imp.ID),
		}
	}

	var params openrtb_ext.ExtImpHypeLab
	if err := jsonutil.Unmarshal(ext.Bidder, &params); err != nil {
		return openrtb_ext.ExtImpHypeLab{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: unable to unmarshal ext.bidder", imp.ID),
		}
	}

	if params.PropertySlug == "" || params.PlacementSlug == "" {
		return openrtb_ext.ExtImpHypeLab{}, &errortypes.BadInput{
			Message: fmt.Sprintf("imp %s: property_slug and placement_slug are required", imp.ID),
		}
	}

	return params, nil
}

func setRequestExt(request *openrtb2.BidRequest) error {
	var ext map[string]json.RawMessage
	if len(request.Ext) > 0 {
		if err := jsonutil.Unmarshal(request.Ext, &ext); err != nil {
			return err
		}
	}
	if ext == nil {
		ext = map[string]json.RawMessage{}
	}

	sourceJSON, err := jsonutil.Marshal(source)
	if err != nil {
		return err
	}
	providerVersionJSON, err := jsonutil.Marshal("prebid-server@" + prebidServerVersion())
	if err != nil {
		return err
	}

	ext["source"] = sourceJSON
	ext["provider_version"] = providerVersionJSON

	request.Ext, err = jsonutil.Marshal(ext)
	return err
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	impLookup := makeImpLookup(request.Imp)
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getBidMediaType(&seatBid.Bid[i], impLookup)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			typedBid := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			}
			if seatBid.Seat != "" {
				typedBid.Seat = openrtb_ext.BidderName(seatBid.Seat)
			}
			bidResponse.Bids = append(bidResponse.Bids, typedBid)
		}
	}

	return bidResponse, errs
}

func makeImpLookup(imps []openrtb2.Imp) map[string]openrtb2.Imp {
	lookup := make(map[string]openrtb2.Imp, len(imps))
	for _, imp := range imps {
		lookup[imp.ID] = imp
	}
	return lookup
}

func getBidMediaType(bid *openrtb2.Bid, impLookup map[string]openrtb2.Imp) (openrtb_ext.BidType, error) {
	imp, ok := impLookup[bid.ImpID]
	if !ok {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s references unknown imp %s", bid.ID, bid.ImpID),
		}
	}

	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}
	if bid.MType != 0 {
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s uses unsupported mtype %d", bid.ID, bid.MType),
		}
	}

	bidType, found, err := getBidMediaTypeFromExt(bid)
	if err != nil {
		return "", err
	}
	if found {
		return bidType, nil
	}

	if strings.HasPrefix(strings.TrimSpace(bid.AdM), "<VAST") {
		return openrtb_ext.BidTypeVideo, nil
	}

	if bidType, ok := getBidMediaTypeFromImp(imp); ok {
		return bidType, nil
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("unable to determine media type for bid %s on imp %s", bid.ID, bid.ImpID),
	}
}

func getBidMediaTypeFromImp(imp openrtb2.Imp) (openrtb_ext.BidType, bool) {
	var bidType openrtb_ext.BidType
	var mediaTypeCount int
	if imp.Banner != nil {
		bidType = openrtb_ext.BidTypeBanner
		mediaTypeCount++
	}
	if imp.Video != nil {
		bidType = openrtb_ext.BidTypeVideo
		mediaTypeCount++
	}
	if imp.Native != nil {
		bidType = openrtb_ext.BidTypeNative
		mediaTypeCount++
	}

	return bidType, mediaTypeCount == 1
}

func getBidMediaTypeFromExt(bid *openrtb2.Bid) (openrtb_ext.BidType, bool, error) {
	if len(bid.Ext) == 0 {
		return "", false, nil
	}

	var ext bidExt
	if err := jsonutil.Unmarshal(bid.Ext, &ext); err != nil {
		return "", false, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s has invalid ext", bid.ID),
		}
	}
	if ext.HypeLab == nil {
		return "", false, nil
	}

	switch ext.HypeLab.CreativeType {
	case "display":
		return openrtb_ext.BidTypeBanner, true, nil
	case "video":
		return openrtb_ext.BidTypeVideo, true, nil
	case "native":
		return openrtb_ext.BidTypeNative, true, nil
	case "":
		return "", false, nil
	default:
		return "", false, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s has unsupported creative_type %s", bid.ID, ext.HypeLab.CreativeType),
		}
	}
}

func prebidServerVersion() string {
	if version.Ver == "" {
		return version.VerUnknown
	}
	return version.Ver
}
