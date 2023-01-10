package commerce

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type CommerceAdapter struct {
	endpoint *template.Template
	impurl *template.Template
	clickurl *template.Template
	conversionurl *template.Template
}

// ExtImpFilteringSubCategory - Impression Filtering SubCategory Extension
type ExtImpFilteringSubCategory struct {
	Name  *string   `json:"name,omitempty"`
	Value []*string `json:"value,omitempty"`
}

// ExtImpPreferred - Impression Preferred Extension
type ExtImpPreferred struct {
	ProductID *string  `json:"pid,omitempty"`
	Rating    *float64 `json:"rating,omitempty"`
}

// ExtImpFiltering - Impression Filtering Extension
type ExtImpFiltering struct {
	Category    []*string                     `json:"category,omitempty"`
	Brand       []*string                     `json:"brand,omitempty"`
	SubCategory []*ExtImpFilteringSubCategory `json:"subcategory,omitempty"`
}

// ExtImpTargeting - Impression Targeting Extension
type ExtImpTargeting struct {
	Name  *string   `json:"name,omitempty"`
	Value []*string `json:"value,omitempty"`
	Type  *int      `json:"type,omitempty"`
}

type ExtCustomConfig struct {
	Key   *string `json:"key,omitempty"`
	Value *string `json:"value,omitempty"`
	Type  *int    `json:"type,omitempty"`
}

// ImpExtensionCommerce - Impression Commerce Extension
type ExtImpCommerce struct {
	MaxRequested   *int               `json:"max_requested,omitempty"`
	SlotsAvailable *int               `json:"slots_available,omitempty"`
	Preferred      []*ExtImpPreferred `json:"preferred,omitempty"`
	Filtering      *ExtImpFiltering   `json:"filtering,omitempty"`
	Targeting      []*ExtImpTargeting `json:"targeting,omitempty"`
}

// UserExtensionCommerce - User Commerce Extension
type ExtUserCommerce struct {
	IsAuthenticated *bool   `json:"is_authenticated,omitempty"`
	Consent         *string `json:"consent,omitempty"`
}

// SiteExtensionCommerce - Site Commerce Extension
type ExtSiteCommerce struct {
	Page *string `json:"page,omitempty"`
}

// AppExtensionCommerce - App Commerce Extension
type ExtAppCommerce struct {
	Page *string `json:"page,omitempty"`
}

type ExtBidderCommerce struct {
	PrebidBidderName  *string            `json:"prebidname,omitempty"`
	BidderCode        *string            `json:"biddercode,omitempty"`
	HostName          *string            `json:"hostname,omitempty"`
	CustomConfig      []*ExtCustomConfig `json:"config,omitempty"`
}

type ExtBidCommerce struct {
	ProductId  *string              `json:"productid,omitempty"`
	ClickUrl        *string         `json:"curl,omitempty"`
	ConversionUrl        *string            `json:"purl,omitempty"`
	ClickPrice        *float64            `json:"clickprice,omitempty"`
	Rate          *float64             `json:"rate,omitempty"`

}

const MAX_COUNT = 10
const COMMERCE_DEFAULT_HOSTNAME = "pubMatic"

func (a *CommerceAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	host := "localhost"

	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt ExtBidderCommerce
	json.Unmarshal(request.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(preBidExt.BidderParams, &commerceExt)
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

func AddDefaultFields(bid *openrtb2.Bid){
	if bid != nil {
		bid.CrID = "DefaultCRID"
	}
}


func GetRequestSlotCount(internalRequest *openrtb2.BidRequest)int {
	impArray := internalRequest.Imp
	reqCount := 0
	for _, eachImp := range impArray {
		var impExt ExtImpCommerce
		json.Unmarshal(eachImp.Ext, &impExt)
		reqCount += *impExt.SlotsAvailable
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

func GetDefaultBidID(name string) string {
	prefix := "BidResponse_" + name+ "_"
	t := time.Now().UnixNano() / int64(time.Microsecond)
	return prefix + strconv.Itoa(int(t))
}

func GetRandomBidPrice() float64 {
	min := 0.0
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

func GetDummyBids(impUrl , clickUrl , conversionUrl, seatName string, requestCount int) (*adapters.BidderResponse) {
	var typedArray     []*adapters.TypedBid

	if requestCount > MAX_COUNT {
		requestCount = MAX_COUNT
	}
	for i := 1; i <= requestCount; i++ {
		productid := GetRandomProductID()
		campaignID := GetRandomCampaignID()
		bidPrice := GetRandomBidPrice()
		clikcPrice := GetRandomClickPrice()
		bidID := GetDefaultBidID(seatName) + "_" + strconv.Itoa(i)
		newIurl := impUrl + "_ImpID=" +bidID
		newCurl := clickUrl + "_ImpID=" +bidID
		newPurl := conversionUrl + "_ImpID=" +bidID
		bidExt := &ExtBidCommerce{
			ProductId:  &productid,
			ClickUrl: &newCurl,
			ConversionUrl: &newPurl,
			ClickPrice: &clikcPrice,
		}
		
		bid := &openrtb2.Bid {
			ID: bidID,
			ImpID: bidID,
			Price: bidPrice,
			CID: campaignID,
			IURL: newIurl,
		}

		AddDefaultFields(bid)

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
	for i := 1; i <= requestCount; i++ {
		productid := GetRandomProductID()
		campaignID := GetRandomCampaignID()
		bidPrice := GetRandomBidPrice()
		clickPrice := GetRandomClickPrice()
		bidID := GetDefaultBidID(seatName) + "_" + strconv.Itoa(i)
		newIurl := impUrl + "_ImpID=" +bidID
		newCurl := clickUrl + "_ImpID=" +bidID
		newPurl := conversionUrl + "_ImpID=" +bidID
		bidExt := &ExtBidCommerce{
			ProductId:  &productid,
			ClickUrl: &newCurl,
			ConversionUrl: &newPurl,
			ClickPrice: &clickPrice,
			
		}
		
		bid := &openrtb2.Bid {
			ID: bidID,
			ImpID: bidID,
			Price: bidPrice,
			CID: campaignID,
			IURL: newIurl,
			Tactic: "Dummy",
		}

		AddDefaultFields(bid)

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


func GetHostName(internalRequest *openrtb2.BidRequest) string {
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt ExtBidderCommerce
	json.Unmarshal(internalRequest.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(preBidExt.BidderParams, &commerceExt)

	return *commerceExt.HostName
}
func (a *CommerceAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	hostName := GetHostName(internalRequest)
	if len(hostName) == 0 {
		hostName = COMMERCE_DEFAULT_HOSTNAME
	}
	iurl, _ := a.buildImpressionURL(hostName) 
	curl, _ := a.buildClickURL(hostName)
	purl, _ := a.buildConversionURL(hostName)
	requestCount := GetRequestSlotCount(internalRequest)

	responseF := GetDummyBids(iurl, curl, purl, "commerce", requestCount)
	return responseF, nil
}

// Builder builds a new instance of the Commerce adapter for the given bidder with the given config.
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

	bidder := &CommerceAdapter{
		endpoint: endpointtemplate,
	    impurl: impurltemplate,
		clickurl: clickurltemplate,
		conversionurl: conversionurltemplate,
	}

	return bidder, nil
}

func (a *CommerceAdapter) buildEndpointURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func (a *CommerceAdapter) buildImpressionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.impurl, endpointParams)
}

func (a *CommerceAdapter) buildClickURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.clickurl, endpointParams)
}

func (a *CommerceAdapter) buildConversionURL(hostName string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{ Host: hostName}
	return macros.ResolveMacros(a.conversionurl, endpointParams)
}