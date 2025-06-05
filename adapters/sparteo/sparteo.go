package sparteo

import (
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
	endpoint   string
	bidderName string
}

type extBidWrapper struct {
	Prebid openrtb_ext.ExtBidPrebid `json:"prebid"`
}

func Builder(bidderName openrtb_ext.BidderName, cfg config.Adapter, server config.Server) (adapters.Bidder, error) {
	return &adapter{
		endpoint:   cfg.Endpoint,
		bidderName: string(bidderName),
	}, nil
}

func parseExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpSparteo, error) {
	var bidderExt adapters.ExtImpBidder

	bidderExtErr := jsonutil.Unmarshal(imp.Ext, &bidderExt)
	if bidderExtErr != nil {
		return nil, fmt.Errorf("ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, bidderExtErr)
	}

	impExt := openrtb_ext.ExtImpSparteo{}
	sparteoExtErr := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
	if sparteoExtErr != nil {
		return nil, fmt.Errorf("ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, sparteoExtErr)
	}

	return &impExt, nil
}

func (a *adapter) MakeRequests(req *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	request := *req

	request.Imp = make([]openrtb2.Imp, len(req.Imp))
	copy(request.Imp, req.Imp)

	if req.Site != nil {
		siteCopy := *req.Site
		request.Site = &siteCopy
	}

	if req.Site != nil && req.Site.Publisher != nil {
		publisherCopy := *req.Site.Publisher
		request.Site.Publisher = &publisherCopy
	}

	var errs []error
	var siteNetworkId string

	for i, imp := range request.Imp {
		extImpSparteo, err := parseExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if siteNetworkId == "" && extImpSparteo.NetworkId != "" {
			siteNetworkId = extImpSparteo.NetworkId
		}

		var extMap map[string]interface{}
		if err := jsonutil.Unmarshal(imp.Ext, &extMap); err != nil {
			errs = append(errs, fmt.Errorf("ignoring imp id=%s, error while unmarshaling ext, err: %s", imp.ID, err))
			continue
		}

		sparteoMap, ok := extMap["sparteo"].(map[string]interface{})
		if !ok {
			sparteoMap = make(map[string]interface{})
			extMap["sparteo"] = sparteoMap
		}

		paramsMap, ok := sparteoMap["params"].(map[string]interface{})
		if !ok {
			paramsMap = make(map[string]interface{})
			sparteoMap["params"] = paramsMap
		}

		bidderObj, ok := extMap["bidder"].(map[string]interface{})
		if ok {
			delete(extMap, "bidder")

			for key, value := range bidderObj {
				paramsMap[key] = value
			}
		}

		updatedExt, err := jsonutil.Marshal(extMap)
		if err != nil {
			errs = append(errs, fmt.Errorf("ignoring imp id=%s, error while marshaling updated ext, err: %s", imp.ID, err))
			continue
		}

		request.Imp[i].Ext = updatedExt
	}

	if request.Site != nil && request.Site.Publisher != nil && siteNetworkId != "" {
		var pubExt map[string]interface{}
		if request.Site.Publisher.Ext != nil {
			if err := jsonutil.Unmarshal(request.Site.Publisher.Ext, &pubExt); err != nil {
				pubExt = make(map[string]interface{})
			}
		} else {
			pubExt = make(map[string]interface{})
		}

		var paramsMap map[string]interface{}
		if raw, ok := pubExt["params"]; ok {
			if paramsMap, ok = raw.(map[string]interface{}); !ok {
				paramsMap = make(map[string]interface{})
			}
		} else {
			paramsMap = make(map[string]interface{})
		}

		paramsMap["networkId"] = siteNetworkId
		pubExt["params"] = paramsMap

		updatedPubExt, err := jsonutil.Marshal(pubExt)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Error marshaling site.publisher.ext: %s", err),
			})
		} else {
			request.Site.Publisher.Ext = jsonutil.RawMessage(updatedPubExt)
		}
	}

	body, err := jsonutil.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	requestData := &adapters.RequestData{
		Method: http.MethodPost,
		Uri:    a.endpoint,
		Body:   body,
		ImpIDs: openrtb_ext.GetImpIDs(request.Imp),
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	return []*adapters.RequestData{requestData}, errs
}

func (a *adapter) MakeBids(req *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = bidResp.Cur

	for _, seatBid := range bidResp.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := a.getMediaType(&bid)
			if err != nil {
				continue
			}

			switch bidType {
			case openrtb_ext.BidTypeBanner:
				seatBid.Bid[i].MType = openrtb2.MarkupBanner
			case openrtb_ext.BidTypeVideo:
				seatBid.Bid[i].MType = openrtb2.MarkupVideo
			case openrtb_ext.BidTypeNative:
				seatBid.Bid[i].MType = openrtb2.MarkupNative
			default:
				continue
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidderResponse, nil
}

func (a *adapter) getMediaType(bid *openrtb2.Bid) (openrtb_ext.BidType, error) {
	var wrapper extBidWrapper
	if err := jsonutil.Unmarshal(bid.Ext, &wrapper); err != nil {
		return "", fmt.Errorf("error unmarshaling bid ext for bid id=%s: %v", bid.ID, err)
	}
	bidExt := wrapper.Prebid

	bidType, err := openrtb_ext.ParseBidType(string(bidExt.Type))
	if err != nil {
		return "", fmt.Errorf("error parsing bid type for bid id=%s: %v", bid.ID, err)
	}

	if bidType == openrtb_ext.BidTypeAudio {
		return "", fmt.Errorf("bid type %q is not supported for bid id=%s", bidExt.Type, bid.ID)
	}

	return bidType, nil
}
