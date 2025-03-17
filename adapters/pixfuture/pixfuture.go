package pixfuture

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"net/http"
	"strconv"
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
	if request == nil || len(request.Imp) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No impressions in bid request"}}
	}

	var errs []error
	var adapterRequests []*adapters.RequestData

	for _, imp := range request.Imp {

		// Log raw imp.Ext for debugging

		// Define struct to match the expected nesting in the JSON
		var ext struct {
			Bidder struct {
				PixID string `json:"pix_id"`
			} `json:"bidder"`
		}

		if err := json.Unmarshal(imp.Ext, &ext); err != nil {
			errs = append(errs, &errortypes.BadInput{Message: "Invalid impression extension"})
			continue
		}

		// Extract pix_id from the structure
		pixID := ext.Bidder.PixID
		idType := "pix_id"

		if pixID == "" {
			errs = append(errs, &errortypes.BadInput{Message: "Missing " + idType})
			continue
		}
		if len(pixID) < 3 {
			errs = append(errs, &errortypes.BadInput{Message: idType + " must be at least 3 characters long"})
			continue
		}

		// Check for supported impression types (banner, native, or video)
		if imp.Banner == nil && imp.Native == nil && imp.Video == nil {
			errs = append(errs, &errortypes.BadInput{Message: "Banner, Native, or Video impression required"})
			continue
		}

		reqJSON, err := json.Marshal(request)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Set("Content-Type", "application/json")
		headers.Set("Accept", "application/json")

		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  "POST",
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
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{Message: "Invalid response format: " + err.Error()}}
	}

	bidResponse := adapters.NewBidderResponse()
	bidResponse.Currency = bidResp.Cur

	var errs []error
	for _, seatBid := range bidResp.SeatBid {
		for _, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errs = append(errs, &errortypes.BadServerResponse{Message: "Failed to parse impression \"" + bid.ImpID + "\" mediatype"})
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
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
	if err := json.Unmarshal(bid.Ext, &ext); err != nil {
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
