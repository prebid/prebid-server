package spotx

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/errortypes"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"net/http"
)

type Adapter struct {
	url string
}

func (a *Adapter) MakeRequests(request *openrtb.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errs []error
	var adapterRequests []*adapters.RequestData

	if len(request.Imp) == 0 {
		errs = append(errs, &errortypes.BadInput{Message: "No impression in the bid request"})
		return nil, errs
	}

	for i, imp := range request.Imp {
		if imp.Video == nil {
			errs = append(errs, errors.New(fmt.Sprintf("non video impression at index %d", i)))
			continue
		}

		adapterReq, err := makeRequest(a, request, imp)
		if adapterReq != nil {
			adapterRequests = append(adapterRequests, adapterReq)
		}
		errs = append(errs, err...)
	}

	return adapterRequests, errs
}

func makeRequest(a *Adapter, originalReq *openrtb.BidRequest, imp openrtb.Imp) (*adapters.RequestData, []error) {
	var errs []error

	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		errs = append(errs, &errortypes.BadInput{
			Message: err.Error(),
		})
		return &adapters.RequestData{}, errs
	}

	var spotxExt openrtb_ext.ExtImpSpotX
	if err := json.Unmarshal(bidderExt.Bidder, &spotxExt); err != nil {
		errs = append(errs, &errortypes.BadInput{
			Message: err.Error(),
		})
		return &adapters.RequestData{}, errs
	}

	reqCopy := *originalReq
	reqCopy.ID = spotxExt.ChannelID

	intermediateReq, _ := json.Marshal(reqCopy)
	reqMap := make(map[string]interface{})
	_ = json.Unmarshal(intermediateReq, &reqMap)

	intermediateImp, _ := json.Marshal(imp)
	impMap := make(map[string]interface{})
	_ = json.Unmarshal(intermediateImp, &impMap)

	if spotxExt.Secure {
		impMap["secure"] = 1
	} else {
		impMap["secure"] = 0
	}

	impVideoExt := map[string]interface{}{}
	if impMap["video"].(map[string]interface{})["ext"] != nil {
		_ = json.Unmarshal(impMap["video"].(map[string]interface{})["ext"].([]byte), &impVideoExt)
	}
	impVideoExt["ad_volume"] = spotxExt.AdVolume
	impVideoExt["ad_unit"] = spotxExt.AdUnit
	if spotxExt.HideSkin {
		impVideoExt["hide_skin"] = 1
	} else {
		impVideoExt["hide_skin"] = 0
	}
	impMap["video"].(map[string]interface{})["ext"] = impVideoExt
	impMap["bidfloor"] = float64(spotxExt.PriceFloor)

	// remove bidder from imp.Ext
	if bidderExt.Prebid != nil {
		byteExt, _ := json.Marshal(bidderExt)
		impMap["ext"] = byteExt
	} else {
		delete(impMap, "ext")
	}
	reqMap["imp"] = impMap

	reqJSON, err := json.Marshal(reqMap)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     fmt.Sprintf("%s/%s", a.url, spotxExt.ChannelID),
		Body:    reqJSON, //TODO: This is a custom request struct, other adapters are sending this openrtb.BidRequest
		Headers: headers,
	}, errs
}

func (a *Adapter) MakeBids(internalRequest *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))
	for _, sb := range bidResp.SeatBid {
		for i := range sb.Bid {
			if mediaType, err := getMediaTypeForImp(bidResp.ID, internalRequest.Imp); err != nil {
				bid := sb.Bid[i]
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: mediaType,
				})
			}
		}
	}
	return bidResponse, nil
}

func getMediaTypeForImp(impID string, imps []openrtb.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID && imp.Video != nil {
			return openrtb_ext.BidTypeVideo, nil
		}
	}
	return "", errors.New("only videos supported")
}

func NewSpotxBidder(url string) *Adapter {
	return &Adapter{
		url: url,
	}
}
