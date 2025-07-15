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

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{endpoint: cfg.Endpoint}, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var validImps []openrtb2.Imp
	var setTestMode bool

	for _, imp := range request.Imp {
		impExt, err := parseImpExt(imp.Ext)
		if err != nil {
			errs = append(errs, fmt.Errorf("impID %s: %v", imp.ID, err))
			continue
		}

		// Validate banner
		if imp.Banner != nil {
			if imp.Banner.W == nil || imp.Banner.H == nil || *imp.Banner.W == 0 || *imp.Banner.H == 0 {
				errs = append(errs, fmt.Errorf("impID %s: invalid banner dimensions", imp.ID))
				continue
			}
		}

		// Validate video
		if imp.Video != nil {
			if len(imp.Video.MIMEs) == 0 {
				errs = append(errs, fmt.Errorf("impID %s: missing or empty video.mimes", imp.ID))
				continue
			}
			if imp.Video.W == nil || imp.Video.H == nil || *imp.Video.W == 0 || *imp.Video.H == 0 {
				errs = append(errs, fmt.Errorf("impID %s: missing or invalid video width/height", imp.ID))
				continue
			}
		}

		// Setting bid floor if present
		if impExt.BidFloor != nil && *impExt.BidFloor > 0 {
			imp.BidFloor = *impExt.BidFloor
		}

		// Check test mode
		if impExt.TestMode != nil && *impExt.TestMode == 1 {
			setTestMode = true
		}

		validImps = append(validImps, imp)
	}

	if len(validImps) == 0 {
		return nil, append(errs, errors.New("no valid impressions"))
	}

	modifiedRequest := *request
	modifiedRequest.Imp = validImps
	if setTestMode {
		modifiedRequest.Test = 1
	}

	reqJSON, err := json.Marshal(modifiedRequest)
	if err != nil {
		return nil, append(errs, err)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json;charset=utf-8")
	headers.Set("Accept", "application/json")
	headers.Set("User-Agent", "prebid-server")
	headers.Set("X-Prebid", "true")

	return []*adapters.RequestData{
		{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    reqJSON,
			Headers: headers,
			ImpIDs:  extractImpIDs(validImps),
		},
	}, errs
}

func extractImpIDs(imps []openrtb2.Imp) []string {
	ids := make([]string, 0, len(imps))
	for _, imp := range imps {
		ids = append(ids, imp.ID)
	}
	return ids
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

			typedBid := &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			}

			br.Bids = append(br.Bids, typedBid)
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
	default:
		return "", fmt.Errorf("unknown bid type mtype=%d", bid.MType)
	}
}
