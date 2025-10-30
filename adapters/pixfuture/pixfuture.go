package pixfuture

import (
	"net/http"
	"strconv"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint: config.Endpoint,
	}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errs []error
	var adapterRequests []*adapters.RequestData
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")
	headers.Set("Accept", "application/json")

	for imp := range iterutil.SlicePointerValues(request.Imp) {
		var bidderExt struct {
			Bidder struct {
				PixID string `json:"pix_id"`
			} `json:"bidder"`
		}

		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: "Invalid pix_id in impression ext: " + err.Error()})
			continue
		}

		requestCopy := *request
		requestCopy.Imp = []openrtb2.Imp{*imp} // slice notation with dereferencing

		reqJSON, err := jsonutil.Marshal(requestCopy)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		})
	}

	if len(adapterRequests) == 0 && len(errs) > 0 {
		return nil, errs
	}
	return adapterRequests, errs
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{Message: "Unexpected status code: " + strconv.Itoa(response.StatusCode)}}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "Invalid response format: " + err.Error()}}
	}

	bidResponse := adapters.NewBidderResponse()
	bidResponse.Currency = bidResp.Cur

	var errs []error
	for i := range bidResp.SeatBid {
		seatBid := &bidResp.SeatBid[i]
		for j := range seatBid.Bid {
			bid := &seatBid.Bid[j]
			bidType, err := getMediaTypeForBid(*bid)
			if err != nil {
				errs = append(errs, &errortypes.BadServerResponse{Message: "Failed to parse impression \"" + bid.ImpID + "\" mediatype"})
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	if len(bidResponse.Bids) == 0 {
		if len(errs) > 0 {
			return nil, errs
		}
		return nil, nil
	}
	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	var ext struct {
		Prebid struct {
			Type string `json:"type"`
		} `json:"prebid"`
	}
	if err := jsonutil.Unmarshal(bid.Ext, &ext); err != nil {
		return "", err
	}

	switch ext.Prebid.Type {
	case "banner":
		return openrtb_ext.BidTypeBanner, nil
	case "video":
		return openrtb_ext.BidTypeVideo, nil
	case "native":
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", &errortypes.BadServerResponse{Message: "Unknown bid type: " + ext.Prebid.Type}
	}
}
