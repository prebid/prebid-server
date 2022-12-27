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
	CustomConfig      []*ExtCustomConfig `json:"config,omitempty"`
}

type ExtBidCommerce struct {
	ProductId  *string              `json:"productid,omitempty"`
	ImpUrl        *string           `json:"iurl,omitempty"`
	ClickUrl        *string         `json:"curl,omitempty"`
	ConversionUrl        *string            `json:"purl,omitempty"`
	BidPrice        *float64            `json:"bidprice,omitempty"`
	ClickPrice        *float64            `json:"clickprice,omitempty"`
	Rate          *float64             `json:"rate,omitempty"`
}

const MAX_COUNT = 10

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
		bid.Price = 2.50
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

func GetDefaultBidID(name string) string {
	prefix := "BidResponse_" + name+ "_"
	t := time.Now().UnixNano() / int64(time.Millisecond)
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

func (a *CommerceAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var typedArray     []*adapters.TypedBid
	iurl, _ := a.buildImpressionURL("commerce") 
	curl, _ := a.buildClickURL("commerce")
	purl, _ := a.buildConversionURL("commerce")
	requestCount := GetRequestSlotCount(internalRequest)


	if requestCount > MAX_COUNT {
		requestCount = MAX_COUNT
	}
	for i := 1; i <= requestCount; i++ {
		productid := GetRandomProductID()
		bidPrice := GetRandomBidPrice()
		clikcPrice := GetRandomClickPrice()
		bidID := GetDefaultBidID("commerce") + "_" + strconv.Itoa(i)
		newIurl := iurl + "_ImpID=" +bidID
		newCurl := curl + "_ImpID=" +bidID
		newPurl := purl + "_ImpID=" +bidID
		bidExt := &ExtBidCommerce{
			ProductId:  &productid,
			ImpUrl:        &newIurl,
			ClickUrl: &newCurl,
			ConversionUrl: &newPurl,
			BidPrice: &bidPrice,
			ClickPrice: &clikcPrice,
		}
		
		bid := &openrtb2.Bid {
			ID:bidID,
			ImpID: bidID,
		}

		AddDefaultFields(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		typedbid := &adapters.TypedBid {
			Bid:  bid,
			Seat: "commerce",
		}
		typedArray = append(typedArray, typedbid)
	}

	responseF := &adapters.BidderResponse{
		Bids: typedArray,
	}
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