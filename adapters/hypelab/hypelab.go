package hypelab

import (
	"encoding/json"
	"fmt"
	"net/http"

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

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
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

	// The HypeLab exchange resolves the property and placement from
	// imp.ext.bidder.property_slug / placement_slug, so the params must be
	// forwarded in the outgoing request.
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

// setRequestExt sets ext.source and ext.provider_version, which the HypeLab
// exchange requires to identify the integration type and version of the
// caller (its bidding logic differs between prebid and SDK traffic).
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

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getBidMediaType(&seatBid.Bid[i])
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

// The HypeLab exchange sets mtype on every bid, so no markup or imp-based
// fallback is needed to resolve the media type.
func getBidMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{
			Message: fmt.Sprintf("bid %s uses unsupported mtype %d", bid.ID, bid.MType),
		}
	}
}

func prebidServerVersion() string {
	if version.Ver == "" {
		return version.VerUnknown
	}
	return version.Ver
}
