package nexverse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Nexverse adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

// MakeRequests builds one request per impression. The Nexverse endpoint identifies the
// publisher/inventory through query parameters (uid, pub_id, pub_epid) which are supplied
// per-impression, so impressions are not batched together.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requests []*adapters.RequestData
	var errs []error

	requestCopy := *request
	for _, imp := range request.Imp {
		nexverseExt, err := parseImpExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// Nexverse expects bid floors in USD. Convert if a foreign currency is provided.
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				errs = append(errs, err)
				continue
			}
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		endpointURL, err := a.buildEndpointURL(nexverseExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requestCopy.Imp = []openrtb2.Imp{imp}
		body, err := json.Marshal(&requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		requests = append(requests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     endpointURL,
			Body:    body,
			Headers: getHeaders(&requestCopy),
			ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
		})
	}

	return requests, errs
}

func parseImpExt(imp *openrtb2.Imp) (*openrtb_ext.ImpExtNexverse, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to parse imp.ext for imp %s: %v", imp.ID, err),
		}
	}

	var nexverseExt openrtb_ext.ImpExtNexverse
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &nexverseExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to parse bidder params for imp %s: %v", imp.ID, err),
		}
	}

	if nexverseExt.UID == "" || nexverseExt.PubID == "" || nexverseExt.PubEpid == "" {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Missing required Nexverse params (uid, pubId, pubEpid) for imp %s", imp.ID),
		}
	}

	return &nexverseExt, nil
}

func (a *adapter) buildEndpointURL(params *openrtb_ext.ImpExtNexverse) (string, error) {
	endpointURL, err := url.Parse(a.endpoint)
	if err != nil {
		return "", &errortypes.BadInput{
			Message: fmt.Sprintf("Failed to parse Nexverse endpoint: %v", err),
		}
	}

	query := endpointURL.Query()
	query.Set("uid", params.UID)
	query.Set("pub_id", params.PubID)
	query.Set("pub_epid", params.PubEpid)
	if params.IsDebug {
		query.Set("test", "1")
	}
	endpointURL.RawQuery = query.Encode()

	return endpointURL.String(), nil
}

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("X-Openrtb-Version", "2.6")

	if request.Device != nil {
		if request.Device.UA != "" {
			headers.Add("User-Agent", request.Device.UA)
		}
		if request.Device.IP != "" {
			headers.Add("X-Forwarded-For", request.Device.IP)
		} else if request.Device.IPv6 != "" {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}
	}

	return headers
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
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Failed to parse Nexverse response: %v", err),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	if response.Cur != "" {
		bidResponse.Currency = response.Cur
	} else {
		bidResponse.Currency = "USD"
	}

	var errs []error
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(seatBid.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

// getMediaTypeForBid resolves the creative type. It relies on the authoritative OpenRTB 2.6
// bid.mtype field and, only when that is not set, falls back to bid.ext.mediaType, which is
// how Nexverse signals the type on responses that omit mtype. The type is never assumed.
func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	if len(bid.Ext) > 0 {
		var bidExt struct {
			MediaType string `json:"mediaType"`
		}
		if err := jsonutil.Unmarshal(bid.Ext, &bidExt); err == nil {
			switch bidExt.MediaType {
			case "banner":
				return openrtb_ext.BidTypeBanner, nil
			case "video":
				return openrtb_ext.BidTypeVideo, nil
			case "native":
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Unable to determine media type for bid %s in imp %s", bid.ID, bid.ImpID),
	}
}
