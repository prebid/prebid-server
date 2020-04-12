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
	"strings"
)

type Adapter struct {
	url string
}

type spotxRequest struct {
	ID         string          `json:"id"`
	Imp        *openrtb.Imp    `json:"imp"`
	Site       *openrtb.Site   `json:"site"`
	Device     *openrtb.Device `json:"device"`
	Ext        json.RawMessage `json:"ext"`
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

func makeRequest(a *Adapter, request *openrtb.BidRequest, imp openrtb.Imp) (*adapters.RequestData, []error) {
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

	if spotxExt.Secure || strings.HasPrefix(request.Site.Page, "https") {
		*imp.Secure = int8(1)
	}

	impVideoExt := map[string]interface{}{}
	_ = json.Unmarshal(imp.Video.Ext, &impVideoExt)
	impVideoExt["ad_volume"] = fmt.Sprintf("%g", spotxExt.AdVolume)
	impVideoExt["ad_unit"] = spotxExt.AdUnit
	if spotxExt.HideSkin {
		impVideoExt["hide_skin"] = 1
	}
	imp.Video.Ext, _ = json.Marshal(impVideoExt)

	spotReq := spotxRequest{
		ID:     spotxExt.ChannelID,
		Imp:    &imp, //TODO: Other adapters are sending this as an array
		Site:   request.Site,
		Device: request.Device,
	}

	imp.BidFloor = float64(spotxExt.PriceFloor)

	reqJSON, err := json.Marshal(spotReq)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	return &adapters.RequestData{
		Method:  "POST",
		Uri:     fmt.Sprintf("%s/%s", a.url, spotxExt.ChannelID),
		Body:    reqJSON, //TODO: This is a custom request, other adapters are sending this openrtb.BidRequest
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
