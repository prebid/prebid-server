package criteostaples

import (
	"encoding/json"
	"math/rand"
	"strconv"

	"github.com/PubMatic-OpenWrap/prebid-server/macros"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	MAX_COUNT                 = 10
	COMMERCE_DEFAULT_HOSTNAME = "pubMatic"
)
func (a *CriteoStaplesAdapter) GetMockResponse(internalRequest *openrtb2.BidRequest) *adapters.BidderResponse {

	hostName := GetHostName(internalRequest)
	if len(hostName) == 0 {
		hostName = COMMERCE_DEFAULT_HOSTNAME
	}

	iurl, _ := a.buildImpressionURL(hostName)
	curl, _ := a.buildClickURL(hostName)
	purl, _ := a.buildConversionURL(hostName)
	requestCount := GetRequestSlotCount(internalRequest)
	impiD := internalRequest.Imp[0].ID

	responseF := GetMockBids(iurl, curl, purl, requestCount, impiD)
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

func GetMockBids(impUrl, clickUrl, conversionUrl string, requestCount int, ImpID string) *adapters.BidderResponse {
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


