package smartadserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type SmartadserverAdapter struct {
	host string
}

func NewSmartadserverBidder(host string) *SmartadserverAdapter {
	return &SmartadserverAdapter{
		host: host,
	}
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *SmartadserverAdapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	var adapterRequests []*adapters.RequestData
	var errs []error
	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: "Error parsing bidderExt object",
			})
			continue
		}

		var smartadserverExt openrtb_ext.ExtImpSmartadserver
		if err := json.Unmarshal(bidderExt.Bidder, &smartadserverExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: "Error parsing smartadserverExt parameters",
			})
			continue
		}

		var errMarshal error
		if imp.Ext, errMarshal = json.Marshal(smartadserverExt); errMarshal != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: errMarshal.Error(),
			})
			continue
		}

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: "Error parsing reqJSON object",
			})
			continue
		}

		url, err := a.BuildEndpointURL(&smartadserverExt)
		if url == "" {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")
		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  "POST",
			Uri:     url,
			Body:    reqJSON,
			Headers: headers,
		})
	}
	return adapterRequests, errs
}

// MakeBids unpacks the server's response into Bids.
func (a *SmartadserverAdapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: getMediaTypeForImp(bid.ImpID, internalRequest.Imp),
			})

		}
	}
	return bidResponse, []error{}
}

// BuildEndpointURL : Builds endpoint url
func (a *SmartadserverAdapter) BuildEndpointURL(params *openrtb_ext.ExtImpSmartadserver) (string, error) {

	host := a.host
	if params.Domain != "" {
		host = params.Domain
	}

	uri, err := url.Parse(host)

	if err != nil {
		return "", &errortypes.BadInput{
			Message: "Malformed URL: " + err.Error(),
		}
	}

	uri.Path = path.Join(uri.Path, "api/prebidserver")

	return uri.String(), nil
}

func getMediaTypeForImp(impId string, imps []openrtb.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impId {
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}
