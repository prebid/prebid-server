package koddi

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type KoddiAdapter struct {
	endpoint *template.Template
	impurl *template.Template
	clickurl *template.Template
	conversionurl *template.Template
}

const MAX_COUNT = 10
const COMMERCE_DEFAULT_HOSTNAME = "pubMatic"

func GetRequestSlotCount(internalRequest *openrtb2.BidRequest)int {
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
	randomN :=rand.Intn(200000)
	t := strconv.Itoa(randomN)
	return t
}

func  GetRandomCampaignID() string {
	randomN :=rand.Intn(9000000)
	t := strconv.Itoa(randomN)
	return t
}


func GetRandomBidPrice() float64 {
	min := 0.1
	max := 1.0
	untruncated := min + rand.Float64() * (max - min)
	truncated := float64(int(untruncated * 100)) / 100
	return truncated
}

func GetRandomClickPrice() float64 {
	min := 1.0
	max := 5.0
	untruncated := min + rand.Float64() * (max - min)
	truncated := float64(int(untruncated * 100)) / 100
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


func GetDummyBids(impUrl , clickUrl , conversionUrl, seatName string, requestCount int, ImpID string) (*adapters.BidderResponse) {
	var typedArray     []*adapters.TypedBid

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
		
		bid := &openrtb2.Bid {
			ID: bidID,
			ImpID: impID,
			Price: bidPrice,
			CID: campaignID,
		}

		adapters.AddDefaultFieldsComm(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		typedbid := &adapters.TypedBid {
			Bid:  bid,
			Seat: openrtb_ext.BidderName(seatName),
		}
		typedArray = append(typedArray, typedbid)
	}

	responseF := &adapters.BidderResponse{
		Bids: typedArray,
	}
	return responseF
}

func GetDummyBids_NoBid(impUrl , clickUrl , conversionUrl, seatName string, requestCount int) (*adapters.BidderResponse) {
	var typedArray     []*adapters.TypedBid

	if requestCount > MAX_COUNT {
		requestCount = MAX_COUNT
	}
	for i := 0; i < requestCount; i++ {
		productid := GetRandomProductID()
		campaignID := GetRandomCampaignID()
		bidPrice := GetRandomBidPrice()
		clickPrice := GetRandomClickPrice()
		bidID := adapters.GenerateUniqueBidIDComm()
		newIurl := impUrl + "_ImpID=" +bidID
		newCurl := clickUrl + "_ImpID=" +bidID
		newPurl := conversionUrl + "_ImpID=" +bidID

		bidExt := &openrtb_ext.ExtBidCommerce{
			ProductId:  productid,
			ClickUrl: newCurl,
			ConversionUrl: newPurl,
			ClickPrice: clickPrice,
			
		}
		
		bid := &openrtb2.Bid {
			ID: bidID,
			ImpID: bidID,
			Price: bidPrice,
			CID: campaignID,
			IURL: newIurl,
			Tactic: "Dummy",
		}

		adapters.AddDefaultFieldsComm(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		typedbid := &adapters.TypedBid {
			Bid:  bid,
			Seat: openrtb_ext.BidderName(seatName),
		}
		typedArray = append(typedArray, typedbid)
	}

	responseF := &adapters.BidderResponse{
		Bids: typedArray,
	}
	return responseF
}


func (a *KoddiAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	host := "localhost"
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt openrtb_ext.ExtImpCommerce
	json.Unmarshal(request.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(request.Imp[0].Ext, &commerceExt)
	endPoint,_ := a.buildEndpointURL(host)
	errs := make([]error, 0, len(request.Imp))

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
	
}
func (a *KoddiAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
    var errors []error 

	hostName := GetHostName(internalRequest)
	if len(hostName) == 0 {
		hostName = COMMERCE_DEFAULT_HOSTNAME
	}
	iurl, _ := a.buildImpressionURL(hostName) 
	curl, _ := a.buildClickURL(hostName)
	purl, _ := a.buildConversionURL(hostName)
	requestCount := GetRequestSlotCount(internalRequest)
	impiD := internalRequest.Imp[0].ID
	
	responseF := GetDummyBids(iurl, curl, purl, "koddi", requestCount, impiD)
	//responseF := commerce.GetDummyBids_NoBid(iurl, curl, purl, "koddi", 1)
    //err := fmt.Errorf("No Bids available for the given request from Koddi")
	//errors = append(errors,err )
	return responseF, errors

}

// Builder builds a new instance of the Koddi adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {

	endpointtemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	impurltemplate, err := template.New("impurlTemplate").Parse(config.ComParams.ImpTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse imp url template: %v", err)
	}

	clickurltemplate, err := template.New("clickurlTemplate").Parse(config.ComParams.ClickTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse click url template: %v", err)
	}

	conversionurltemplate, err := template.New("endpointTemplate").Parse(config.ComParams.ConversionTracker)
	if err != nil {
		return nil, fmt.Errorf("unable to parse conversion url template: %v", err)
	}

	bidder := &KoddiAdapter{
		endpoint: endpointtemplate,
	    impurl: impurltemplate,
		clickurl: clickurltemplate,
		conversionurl: conversionurltemplate,
	}

	return bidder, nil
}

func (a *KoddiAdapter) buildEndpointURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *KoddiAdapter) buildImpressionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.impurl, endpointParams)
}

func (a *KoddiAdapter) buildClickURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.clickurl, endpointParams)
}

func (a *KoddiAdapter) buildConversionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.conversionurl, endpointParams)
}

