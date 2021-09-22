package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
}
type adnAdunit struct {
	AuId     string `json:"auId"`
	TargetId string `json:"targetId"`
}

type AdnResponse struct {
	AdUnits []struct {
		AuId       string
		TargetId   string
		Html       string
		ResponseId string
		Ads        []struct {
			Bid struct {
				Amount   float64
				Currency string
			}
			AdId            string
			CreativeWidth   string
			CreativeHeight  string
			CreativeId      string
			LineItemId      string
			Html            string
			DestinationUrls map[string]string
		}
	}
}
type adnMetaData struct {
	Usi string `json:"usi"`
}
type adnRequest struct {
	AdUnits  []adnAdunit `json:"adUnits"`
	MetaData adnMetaData `json:"metaData,omitempty"`
	Context  string      `json:"context,omitempty"`
}

// Builder builds a new instance of the BrightMountainMedia adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var extRequests []*adapters.RequestData
	var errs []error

	for _, imp := range request.Imp {
		extRequest, err := a.generateRequests(*request, imp)
		if err != nil {
			errs = append(errs, err)
		} else {
			extRequests = append(extRequests, extRequest)
		}
	}
	return extRequests, errs
}

func (a *adapter) generateRequests(ortbRequest openrtb2.BidRequest, ortbImp openrtb2.Imp) (*adapters.RequestData, error) {

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")


	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(ortbImp.Ext, &bidderExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBidder: %s", err.Error()),
		}
	}

	var adnuntiusExt openrtb_ext.ImpExtAdnunitus
	if err := json.Unmarshal(bidderExt.Bidder, &adnuntiusExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("Error unmarshalling ExtImpBmtm: %s", err.Error()),
		}
	}

	adnReq, reqErr := generateAdUnitRequest(adnuntiusExt, ortbRequest)
	if reqErr != nil {
		return nil, reqErr
	}

	adnJson, err := json.Marshal(adnReq)
	if err != nil {
		return nil, err
	}

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    adnJson,
		Headers: headers,
	}

	return requestData, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unknown status code: %d.", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unknown status code: %d.", response.StatusCode),
		}}
	}

	var adnResponse AdnResponse
	if err := json.Unmarshal(response.Body, &adnResponse); err != nil {
		return nil, []error{err}
	}

	bidResponse, bidErr := generateBidResponse(&adnResponse, request)
	if bidErr != nil {
		return nil, bidErr
	}

	return bidResponse, nil
}

func generateBidResponse(adnResponse *AdnResponse, request *openrtb2.BidRequest) (*adapters.BidderResponse, []error) {
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(adnResponse.AdUnits))
	currency := bidResponse.Currency

	for i, adunit := range adnResponse.AdUnits {
		var bid openrtb2.Bid
		ad := adunit.Ads[0]

		currency = ad.Bid.Currency

		creativeWidth, Werr := strconv.ParseInt(ad.CreativeWidth, 10, 64)
		if Werr != nil {
			return nil, []error{Werr}
		}

		creativeHeight, Herr := strconv.ParseInt(ad.CreativeHeight, 10, 64)
		if Herr != nil {
			return nil, []error{Herr}
		}

		adDomain := []string{}
		for _, url := range ad.DestinationUrls {
			domainArray := strings.Split(url, "/")
			domain := strings.Replace(domainArray[2], "www.", "", -1)
			adDomain = append(adDomain, domain)
		}

		bid = openrtb2.Bid{
			ID:      ad.AdId,
			ImpID:   request.Imp[i].ID,
			W:       creativeWidth,
			H:       creativeHeight,
			AdID:    ad.AdId,
			CID:     ad.LineItemId,
			CrID:    ad.CreativeId,
			Price:   ad.Bid.Amount * 1000,
			AdM:     adunit.Html,
			ADomain: adDomain,
		}

		bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
			Bid:     &bid,
			BidType: "banner",
		})

	}
	bidResponse.Currency = currency
	return bidResponse, nil
}

func generateAdUnitRequest(adnuntiusExt openrtb_ext.ImpExtAdnunitus, ortbRequest openrtb2.BidRequest) (*adnRequest, error) {
	adunits := []adnAdunit{}
	rawUuid, uidErr := uuid.NewV4()
	if uidErr != nil {
		return nil, uidErr
	}
	
	adnuntiusAdunits := adnAdunit{
		AuId:     adnuntiusExt.Auid,
		TargetId: rawUuid.String(),
	}

	adunits = append(adunits, adnuntiusAdunits)

	var userId string
	ortbUser := ortbRequest.User
	if ortbUser != nil {
		ortbUserId := ortbRequest.User.ID
		if ortbUserId != "" {
			userId = ortbRequest.User.ID
		}
	}

	var site string
	if ortbRequest.Site != nil && ortbRequest.Site.Page != "" {
		site = ortbRequest.Site.Page
	}

	adnuntiusRequest := adnRequest{
		AdUnits:  adunits,
		MetaData: adnMetaData{Usi: userId},
		Context:  site,
	}

	return &adnuntiusRequest, nil
}
