package sparteo

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
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding extImpBidder, err: %s", imp.ID, bidderExtErr),
		}
	}

	impExt := openrtb_ext.ExtImpSparteo{}
	sparteoExtErr := jsonutil.Unmarshal(bidderExt.Bidder, &impExt)
	if sparteoExtErr != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Ignoring imp id=%s, error while decoding impExt, err: %s", imp.ID, sparteoExtErr),
		}
	}

	return &impExt, nil
}

func (a *adapter) MakeRequests(req *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	impressions := req.Imp
	if len(impressions) == 0 {
		return nil, []error{&errortypes.BadInput{Message: "No impressions in the bid request"}}
	}
	var errs []error

	var siteNetworkId string

	for i, imp := range impressions {
		extImpSparteo, err := parseExt(&imp)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if siteNetworkId == "" && extImpSparteo.NetworkId != "" {
			siteNetworkId = extImpSparteo.NetworkId
		}

		var extMap map[string]interface{}
		if err := json.Unmarshal(imp.Ext, &extMap); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Ignoring imp id=%s, error while unmarshaling ext, err: %s", imp.ID, err),
			})
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
			delete(bidderObj, "networkId")

			for key, value := range bidderObj {
				paramsMap[key] = value
			}
		}

		updatedExt, err := json.Marshal(extMap)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Ignoring imp id=%s, error while marshaling updated ext, err: %s", imp.ID, err),
			})
			continue
		}

		req.Imp[i].Ext = updatedExt
	}

	if req.Site != nil && req.Site.Publisher != nil && siteNetworkId != "" {
		var pubExt map[string]interface{}

		if req.Site.Publisher.Ext != nil {
			if err := json.Unmarshal(req.Site.Publisher.Ext, &pubExt); err != nil {
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

		updatedPubExt, err := json.Marshal(pubExt)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Error marshaling site.publisher.ext: %s", err),
			})
		} else {
			req.Site.Publisher.Ext = json.RawMessage(updatedPubExt)
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   body,
		ImpIDs: openrtb_ext.GetImpIDs(req.Imp),
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	return []*adapters.RequestData{requestData}, errs
}

func (a *adapter) MakeBids(req *openrtb2.BidRequest, reqData *adapters.RequestData, respData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if respData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if respData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", respData.StatusCode),
		}}
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
	if err := json.Unmarshal(bid.Ext, &wrapper); err != nil {
		return "", fmt.Errorf("error unmarshaling bid ext for bid id=%s: %v", bid.ID, err)
	}
	bidExt := wrapper.Prebid

	mediaMap := map[string]openrtb_ext.BidType{
		"video":  openrtb_ext.BidTypeVideo,
		"banner": openrtb_ext.BidTypeBanner,
	}

	if mt, ok := mediaMap[string(bidExt.Type)]; ok {
		return mt, nil
	}
	return "", fmt.Errorf("unknown bid type %q for bid id=%s", bidExt.Type, bid.ID)
}
