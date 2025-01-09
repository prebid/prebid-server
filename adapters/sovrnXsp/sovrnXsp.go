package sovrnXsp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/openrtb/v20/openrtb2"
)

type adapter struct {
	Endpoint string
}

// bidExt.CreativeType values.
const (
	creativeTypeBanner int = 0
	creativeTypeVideo  int = 1
	creativeTypeNative int = 2
	creativeTypeAudio  int = 3
)

// Bid response extension from XSP.
type bidExt struct {
	CreativeType int `json:"creative_type"`
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	appCopy := *request.App
	if appCopy.Publisher == nil {
		appCopy.Publisher = &openrtb2.Publisher{}
	} else {
		publisherCopy := *appCopy.Publisher
		appCopy.Publisher = &publisherCopy
	}
	request.App = &appCopy

	var errors []error
	var imps []openrtb2.Imp

	for idx, imp := range request.Imp {
		if imp.Banner == nil && imp.Video == nil && imp.Native == nil {
			continue
		}

		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: ext.bidder not provided", idx),
			}
			errors = append(errors, err)
			continue
		}

		var xspExt openrtb_ext.ExtImpSovrnXsp
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &xspExt); err != nil {
			err = &errortypes.BadInput{
				Message: fmt.Sprintf("imp #%d: %s", idx, err.Error()),
			}
			errors = append(errors, err)
			continue
		}

		request.App.Publisher.ID = xspExt.PubID
		if xspExt.MedID != "" {
			request.App.ID = xspExt.MedID
		}
		if xspExt.ZoneID != "" {
			imp.TagID = xspExt.ZoneID
		}
		imps = append(imps, imp)
	}

	if len(imps) == 0 {
		return nil, append(errors, &errortypes.BadInput{
			Message: "no matching impression with ad format",
		})
	}

	request.Imp = imps
	requestJson, err := json.Marshal(request)
	if err != nil {
		return nil, append(errors, err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("x-openrtb-version", "2.5")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.Endpoint,
		Body:    requestJson,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}}, errors
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

	var errors []error
	result := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))

	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid
			var ext bidExt
			if err := jsonutil.Unmarshal(bid.Ext, &ext); err != nil {
				errors = append(errors, err)
				continue
			}

			var bidType openrtb_ext.BidType
			var mkupType openrtb2.MarkupType
			switch ext.CreativeType {
			case creativeTypeBanner:
				bidType = openrtb_ext.BidTypeBanner
				mkupType = openrtb2.MarkupBanner
			case creativeTypeVideo:
				bidType = openrtb_ext.BidTypeVideo
				mkupType = openrtb2.MarkupVideo
			case creativeTypeNative:
				bidType = openrtb_ext.BidTypeNative
				mkupType = openrtb2.MarkupNative
			default:
				errors = append(errors, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unsupported creative type: %d", ext.CreativeType),
				})
				continue
			}

			if bid.MType == 0 {
				bid.MType = mkupType
			}

			result.Bids = append(result.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: bidType,
			})
		}
	}

	if len(result.Bids) == 0 {
		// it's possible an empty seat array was sent as a response
		return nil, errors
	}
	return result, errors
}

// Builder builds a new instance of the SovrnXSP adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		Endpoint: config.Endpoint,
	}
	return bidder, nil
}
