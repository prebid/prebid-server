package risemediatech

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: config.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	for _, imp := range request.Imp {
		impExt, err := parseImpExt(imp.Ext)
		if err != nil {
			errs = append(errs, fmt.Errorf("impID %s: %v", imp.ID, err))
			continue
		}

		//Validate banner fields if this is a banner impression
		if imp.Banner != nil {
			if imp.Banner.W == nil || imp.Banner.H == nil {
				errs = append(errs, fmt.Errorf("impID %s: missing banner w or h", imp.ID))
				continue
			}
			if *imp.Banner.W == 0 || *imp.Banner.H == 0 {
				errs = append(errs, fmt.Errorf("impID %s: banner w or h cannot be zero", imp.ID))
				continue
			}
		}

		// Validate additional video fields if this is a video impression
		if imp.Video != nil {
			if impExt.MinDuration == 0 {
				errs = append(errs, fmt.Errorf("impID %s: missing minDuration", imp.ID))
				continue
			}
			if impExt.MaxDuration == 0 {
				errs = append(errs, fmt.Errorf("impID %s: missing maxDuration", imp.ID))
				continue
			}
			if impExt.StartDelay == 0 {
				errs = append(errs, fmt.Errorf("impID %s: missing startDelay", imp.ID))
				continue
			}
			if len(impExt.Protocols) == 0 {
				errs = append(errs, fmt.Errorf("impID %s: missing protocols", imp.ID))
				continue
			}
		}

		// Prepare sanitized request for each impression
		newImp := imp
		newImp.Ext = nil // Remove bidder extension

		// Create individual bid request with single imp
		modifiedRequest := *request
		modifiedRequest.Imp = []openrtb2.Imp{newImp}

		reqJSON, err := json.Marshal(modifiedRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		headers := http.Header{}
		headers.Add("Content-Type", "application/json;charset=utf-8")
		headers.Add("Accept", "application/json")

		adapterRequests = append(adapterRequests, &adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  []string{imp.ID},
		})
	}

	if len(adapterRequests) == 0 {
		return nil, append(errs, errors.New("no valid impressions"))
	}
	return adapterRequests, errs
}

func parseImpExt(ext json.RawMessage) (*openrtb_ext.ExtImpRiseMediaTech, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(ext, &bidderExt); err != nil {
		return nil, err
	}
	var riseExt openrtb_ext.ExtImpRiseMediaTech
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &riseExt); err != nil {
		return nil, err
	}
	return &riseExt, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(respData) {
		return nil, nil
	}
	if err := adapters.CheckResponseStatusCodeForErrors(respData); err != nil {
		return nil, []error{err}
	}

	var bidResp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(respData.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	br := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid))
	if bidResp.Cur != "" {
		br.Currency = bidResp.Cur
	}

	for _, seatBid := range bidResp.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getBidType(bid)
			if err != nil {
				return nil, []error{err}
			}

			br.Bids = append(br.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}
	return br, nil
}

func getBidType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unknown bid type mtype=%d", bid.MType)
	}
}
