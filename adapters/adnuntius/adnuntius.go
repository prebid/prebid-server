package adnuntius

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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

// Builder builds a new instance of the Adnuntius adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var extRequests []*adapters.RequestData
	var errs []error
	extRequests, err := a.generateRequests(*request)
	if err != nil {
		errs = append(errs, err)
	}
	return extRequests, errs
}

/*
	Generate the requests to Adnuntius to reduce the amount of requests going out.
*/
func (a *adapter) generateRequests(ortbRequest openrtb2.BidRequest) ([]*adapters.RequestData, error) {

	var requestData []*adapters.RequestData
	networkAdunitMap := make(map[string][]adnAdunit)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	for _, imp := range ortbRequest.Imp {

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
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

		var network string
		if adnuntiusExt.Network != "" {
			network = adnuntiusExt.Network
		} else {
			network = "default"
		}

		networkAdunitMap[network] = append(
			networkAdunitMap[network],
			adnAdunit{
				AuId:     adnuntiusExt.Auid,
				TargetId: fmt.Sprintf("%s-%s", adnuntiusExt.Auid, imp.ID),
			})
	}

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

	/*
		Divide requests that go to different networks.
	*/

	for _, networkAdunits := range networkAdunitMap {

		adnuntiusRequest := adnRequest{
			AdUnits:  networkAdunits,
			MetaData: adnMetaData{Usi: userId},
			Context:  site,
		}

		adnJson, err := json.Marshal(adnuntiusRequest)
		if err != nil {
			return nil, err
		}

		_, offset := time.Now().UTC().Local().Zone()
		tzo := - offset / 3600 *60
		
		requestData = append(requestData, &adapters.RequestData{
			Method:  http.MethodPost,
			Uri:     a.endpoint + fmt.Sprintf("&tzo=%s", fmt.Sprint(tzo)),
			Body:    adnJson,
			Headers: headers,
		})

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
		if len(adunit.Ads) > 0 {
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

	}
	bidResponse.Currency = currency
	return bidResponse, nil
}
