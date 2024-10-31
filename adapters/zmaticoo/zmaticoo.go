package zmaticoo

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the zmaticoo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	adapterRequest, errs := a.makeRequest(request)
	if errs != nil {
		return nil, errs
	}
	return []*adapters.RequestData{adapterRequest}, nil

}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	errs := validateZmaticooExt(request)
	if errs != nil {
		return nil, errs
	}
	err := transform(request)
	if err != nil {
		return nil, append(errs, err)
	}
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, append(errs, err)
	}
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    reqBody,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}, errs
}

func transform(request *openrtb2.BidRequest) error {
	for i, imp := range request.Imp {
		if imp.Native != nil {
			var nativeRequest map[string]interface{}
			nativeCopyRequest := make(map[string]interface{})
			if err := jsonutil.Unmarshal([]byte(request.Imp[i].Native.Request), &nativeRequest); err != nil {
				return err
			}
			_, exists := nativeRequest["native"]
			if exists {
				continue
			}
			nativeCopyRequest["native"] = nativeRequest
			nativeReqByte, err := json.Marshal(nativeCopyRequest)
			if err != nil {
				return err
			}
			nativeCopy := *request.Imp[i].Native
			nativeCopy.Request = string(nativeReqByte)
			request.Imp[i].Native = &nativeCopy
		}
	}
	return nil
}

func validateZmaticooExt(request *openrtb2.BidRequest) []error {
	var extImpZmaticoo openrtb_ext.ExtImpZmaticoo
	var errs []error
	for _, imp := range request.Imp {
		var extBidder adapters.ExtImpBidder
		err := jsonutil.Unmarshal(imp.Ext, &extBidder)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = jsonutil.Unmarshal(extBidder.Bidder, &extImpZmaticoo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if extImpZmaticoo.ZoneId == "" || extImpZmaticoo.PubId == "" {
			errs = append(errs, fmt.Errorf("imp.ext.pubId or imp.ext.zoneId required"))
			continue
		}
	}
	return errs

}

// MakeBids make the bids for the bid response.
func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(response) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(response); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d.", response.StatusCode),
		}}
	}
	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			mediaType, err := getMediaTypeForBid(sb.Bid[i])
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: mediaType,
			})
		}
	}
	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", fmt.Errorf("unrecognized bid type in response from zmaticoo for bid %s", bid.ImpID)
	}
}
