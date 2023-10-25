package criteostaples

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/PubMatic-OpenWrap/prebid-server/macros"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	MAX_COUNT                 = 10
	COMMERCE_DEFAULT_HOSTNAME = "pubMatic"
)

type Placement struct {
	Format       string                   `json:"format"`
	Products     []map[string]interface{} `json:"products"`
	OnLoadBeacon string                   `json:"OnLoadBeacon,omitempty"`
	OnViewBeacon string                   `json:"OnViewBeacon,omitempty"`
}

type CriteoResponse struct {
	Status               string                   `json:"status"`
	OnAvailabilityUpdate interface{}              `json:"OnAvailabilityUpdate"`
	Placements           []map[string][]Placement `json:"placements"`
}

func (a *CriteoStaplesAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	var errors []error

	commerceExt, err := adapters.GetImpressionExtComm(&(internalRequest.Imp[0]))
	if err != nil {
		errors := append(errors, err)
		return nil, errors
	}

	if commerceExt.ComParams.TestRequest {

		dummyResponse := a.GetDummyResponse(internalRequest)
		return dummyResponse, nil
	}

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", response.StatusCode),
		}}
	}

	criteoResponse, err := newCriteoStaplesResponseFromBytes(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	if criteoResponse.Status != RESPONSE_OK {
		return nil, []error{&errortypes.BidderFailedSchemaValidation{
			Message: "Error Occured at Criteo for the given request ",
		}}
	}

	if criteoResponse.Placements == nil || len(criteoResponse.Placements) <= 0 {
		return nil, []error{&errortypes.NoBidPrice{
			Message: "No Bid For the given Request",
		}}
	}

	impID := internalRequest.Imp[0].ID
	bidderResponse := a.getBidderResponse(internalRequest, &criteoResponse, impID)
	return bidderResponse, nil
}

func (a *CriteoStaplesAdapter) GetDummyResponse(internalRequest *openrtb2.BidRequest) *adapters.BidderResponse {

	hostName := GetHostName(internalRequest)
	if len(hostName) == 0 {
		hostName = COMMERCE_DEFAULT_HOSTNAME
	}

	iurl, _ := a.buildImpressionURL(hostName)
	curl, _ := a.buildClickURL(hostName)
	purl, _ := a.buildConversionURL(hostName)
	requestCount := GetRequestSlotCount(internalRequest)
	impiD := internalRequest.Imp[0].ID

	responseF := GetDummyBids(iurl, curl, purl, requestCount, impiD)
	return responseF
}

func GetRequestSlotCount(internalRequest *openrtb2.BidRequest) int {
	impArray := internalRequest.Imp
	reqCount := 0
	for _, eachImp := range impArray {
		var commerceExt openrtb_ext.ExtImpCommerce
		json.Unmarshal(eachImp.Ext, &commerceExt)
		reqCount += commerceExt.ComParams.SlotsRequested
	}
	return reqCount
}

func GetRandomProductID() string {
	randomN := rand.Intn(200000)
	t := strconv.Itoa(randomN)
	return t
}

func GetRandomCampaignID() string {
	randomN := rand.Intn(9000000)
	t := strconv.Itoa(randomN)
	return t
}

func GetRandomBidPrice() float64 {
	min := 0.1
	max := 1.0
	untruncated := min + rand.Float64()*(max-min)
	truncated := float64(int(untruncated*100)) / 100
	return truncated
}

func GetRandomClickPrice() float64 {
	min := 1.0
	max := 5.0
	untruncated := min + rand.Float64()*(max-min)
	truncated := float64(int(untruncated*100)) / 100
	return truncated
}

func GetHostName(internalRequest *openrtb2.BidRequest) string {
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt openrtb_ext.ExtImpCommerce

	json.Unmarshal(internalRequest.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(internalRequest.Imp[0].Ext, &commerceExt)
	return commerceExt.Bidder.BidderCode
}

func GetDummyBids(impUrl, clickUrl, conversionUrl string, requestCount int, ImpID string) *adapters.BidderResponse {
	var typedArray []*adapters.TypedBid

	if requestCount > MAX_COUNT {
		requestCount = MAX_COUNT
	}

	for i := 1; i <= requestCount; i++ {
		productid := GetRandomProductID()
		campaignID := GetRandomCampaignID()
		bidPrice := GetRandomBidPrice()
		clickPrice := GetRandomClickPrice()
		bidID := adapters.GenerateUniqueBidIDComm()
		impID := ImpID + "_" + strconv.Itoa(i)

		bidExt := &openrtb_ext.ExtBidCommerce{
			ProductId:  productid,
			ClickPrice: clickPrice,
		}

		bid := &openrtb2.Bid{
			ID:    bidID,
			ImpID: impID,
			Price: bidPrice,
			CID:   campaignID,
		}

		adapters.AddDefaultFieldsComm(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		typedbid := &adapters.TypedBid{
			Bid:  bid,
			Seat: openrtb_ext.BidderName(SEAT_CRITEOSTAPLES),
		}
		typedArray = append(typedArray, typedbid)
	}

	responseF := &adapters.BidderResponse{
		Bids: typedArray,
	}
	return responseF
}

func (a *CriteoStaplesAdapter) getBidderResponse(request *openrtb2.BidRequest, criteoResponse *CriteoResponse, requestImpID string) *adapters.BidderResponse {

	noOfBids := countSponsoredProducts(criteoResponse)
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(noOfBids)
	index := 1
	for _, placementMap := range criteoResponse.Placements {
		for _, placements := range placementMap {
			for _, placement := range placements {
				if placement.Format == FORMAT_SPONSORED {
					for _, productMap := range placement.Products {
						bidID := adapters.GenerateUniqueBidIDComm()
						impID := requestImpID + "_" + strconv.Itoa(index)
						bidPrice, _ := strconv.ParseFloat(strings.TrimSpace(productMap[BID_PRICE].(string)), 64)
						clickPrice, _ := strconv.ParseFloat(strings.TrimSpace(productMap[CLICK_PRICE].(string)), 64)
						productID := productMap[PRODUCT_ID].(string)

						impressionURL := IMP_KEY + adapters.EncodeURL(productMap[VIEW_BEACON].(string))
						clickURL := CLICK_KEY + adapters.EncodeURL(productMap[CLICK_BEACON].(string))
						index++

						// Add ProductDetails to bidExtension
						productDetails := make(map[string]interface{})
						for key, value := range productMap {
							productDetails[key] = value
						}

						delete(productDetails, PRODUCT_ID)
						delete(productDetails, BID_PRICE)
						delete(productDetails, CLICK_PRICE)
						delete(productDetails, VIEW_BEACON)
						delete(productDetails, CLICK_BEACON)

						bidExt := &openrtb_ext.ExtBidCommerce{
							ProductId:      productID,
							ClickUrl:       clickURL,
							ClickPrice:     clickPrice,
							ProductDetails: productDetails,
						}

						bid := &openrtb2.Bid{
							ID:    bidID,
							ImpID: impID,
							Price: bidPrice,
							IURL:  impressionURL,
						}

						adapters.AddDefaultFieldsComm(bid)
						bidExtJSON, err1 := json.Marshal(bidExt)
						if nil == err1 {
							bid.Ext = json.RawMessage(bidExtJSON)
						}

						seat := openrtb_ext.BidderName(SEAT_CRITEOSTAPLES)

						typedbid := &adapters.TypedBid{
							Bid:  bid,
							Seat: seat,
						}
						bidResponse.Bids = append(bidResponse.Bids, typedbid)
					}
				}
			}
		}
	}
	return bidResponse
}

func newCriteoStaplesResponseFromBytes(bytes []byte) (CriteoResponse, error) {
	var err error
	var bidResponse CriteoResponse

	if err = json.Unmarshal(bytes, &bidResponse); err != nil {
		return bidResponse, err
	}

	return bidResponse, nil
}

func countSponsoredProducts(adResponse *CriteoResponse) int {
	count := 0

	// Iterate through placements
	for _, placementMap := range adResponse.Placements {
		for _, placements := range placementMap {
			for _, placement := range placements {
				if placement.Format == FORMAT_SPONSORED {
					count += len(placement.Products)
				}
			}
		}
	}

	return count
}

func (a *CriteoStaplesAdapter) buildImpressionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: hostName}
	return macros.ResolveMacros(a.impurl, endpointParams)
}

func (a *CriteoStaplesAdapter) buildClickURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: hostName}
	return macros.ResolveMacros(a.clickurl, endpointParams)
}

func (a *CriteoStaplesAdapter) buildConversionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{Host: hostName}
	return macros.ResolveMacros(a.conversionurl, endpointParams)
}
