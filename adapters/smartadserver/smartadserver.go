package smartadserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type SmartAdserverAdapter struct {
	host string
}

// Builder builds a new instance of the SmartAdserver adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &SmartAdserverAdapter{
		host: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests makes the HTTP requests which should be made to fetch bids.
func (a *SmartAdserverAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	if len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{
			Message: "No impression in the bid request",
		}}
	}

	var adapterRequests []*adapters.RequestData
	var errs []error

	// We copy the original request.
	smartRequest := *request

	// We create or copy the Site object.
	if smartRequest.Site == nil {
		smartRequest.Site = &openrtb2.Site{}
	} else {
		site := *smartRequest.Site
		smartRequest.Site = &site
	}

	// We create or copy the Publisher object.
	if smartRequest.Site.Publisher == nil {
		smartRequest.Site.Publisher = &openrtb2.Publisher{}
	} else {
		publisher := *smartRequest.Site.Publisher
		smartRequest.Site.Publisher = &publisher
	}

	// We send one serialized "smartRequest" per impression of the original request.
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

		// Adding publisher id.
		smartRequest.Site.Publisher.ID = strconv.Itoa(smartadserverExt.NetworkID)

		// We send one request for each impression.
		smartRequest.Imp = []openrtb2.Imp{imp}

		var errMarshal error
		if imp.Ext, errMarshal = json.Marshal(smartadserverExt); errMarshal != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: errMarshal.Error(),
			})
			continue
		}

		reqJSON, err := json.Marshal(smartRequest)
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
func (a *SmartAdserverAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: "Unexpected status code: " + strconv.Itoa(response.StatusCode) + ". Run with request.debug = 1 for more info",
		}}
	}

	var bidResp openrtb2.BidResponse
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
func (a *SmartAdserverAdapter) BuildEndpointURL(params *openrtb_ext.ExtImpSmartadserver) (string, error) {
	uri, err := url.Parse(a.host)
	if err != nil || uri.Scheme == "" || uri.Host == "" {
		return "", &errortypes.BadInput{
			Message: "Malformed URL: " + a.host + ".",
		}
	}

	uri.Path = path.Join(uri.Path, "api/bid")
	uri.RawQuery = "callerId=5"

	return uri.String(), nil
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) openrtb_ext.BidType {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Video != nil {
				return openrtb_ext.BidTypeVideo
			}
			return openrtb_ext.BidTypeBanner
		}
	}
	return openrtb_ext.BidTypeBanner
}
