package zmaticoo

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"net/http"
	"net/url"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the zmaticoo adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	endpointURL, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %v", err)
	}
	bidder := &adapter{
		endpoint: endpointURL.String(),
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adapterRequests []*adapters.RequestData
	adapterRequest, errs := a.makeRequest(request)
	if errs == nil {
		adapterRequests = append(adapterRequests, adapterRequest)
	}
	return adapterRequests, errs
}

func (a *adapter) makeRequest(request *openrtb2.BidRequest) (*adapters.RequestData, []error) {
	var errs []error
	zmaticooExt, errs := getZmaticooExt(request)
	if zmaticooExt == nil {
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
	}, errs
}

func transform(request *openrtb2.BidRequest) error {
	for i, imp := range request.Imp {
		if imp.Native != nil {
			var nativeRequest map[string]interface{}
			nativeCopyRequest := make(map[string]interface{})
			err := json.Unmarshal([]byte(request.Imp[i].Native.Request), &nativeRequest)
			//just ignore the bad native request
			if err == nil {
				_, exists := nativeRequest["native"]
				if exists {
					continue
				}
				nativeCopyRequest["native"] = nativeRequest
				nativeReqByte, err := json.Marshal(nativeCopyRequest)
				//just ignore the bad native request
				if err != nil {
					return err
				}
				nativeCopy := *request.Imp[i].Native
				nativeCopy.Request = string(nativeReqByte)
				request.Imp[i].Native = &nativeCopy
			} else {
				return err
			}
		}
	}
	return nil
}

func getZmaticooExt(request *openrtb2.BidRequest) (*openrtb_ext.ExtImpZmaticoo, []error) {
	var extImpZmaticoo openrtb_ext.ExtImpZmaticoo
	var errs []error
	for _, imp := range request.Imp {
		var extBidder adapters.ExtImpBidder
		err := json.Unmarshal(imp.Ext, &extBidder)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		err = json.Unmarshal(extBidder.Bidder, &extImpZmaticoo)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		break
	}
	return &extImpZmaticoo, errs

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
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}
	var errs []error
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
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
	default:
		return "", fmt.Errorf("unrecognized bid type in response from rtbhouse for bid %s", bid.ImpID)
	}
}
