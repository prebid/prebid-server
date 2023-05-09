package adbuttler

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"text/template"
	"time"

	"github.com/PubMatic-OpenWrap/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	FLOOR_PRICE                   = "floor_price"
	ZONE_ID                       = "zone_id"
	ACCOUNT_ID                    = "account_id"
	SEARCHTYPE_DEFAULT            = "exact"
	SEARCHTYPE                    = "search_type"
	PAGE_SOURCE                   = "page_source"
	USER_AGE					  = "user_age"
	GENDER_MALE					  = "male"
	GENDER_FEMALE                 = "female"
	GENDER_OTHER                  = "other"
	USER_GENDER                   = "user_gender"
	COUNTRY                       = "country"
	REGION                        = "region"
	CITY                          = "city"
)

type AdButtlerAdapter struct {
	endpoint *template.Template
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
type CommerceParams struct {
	SlotsRequested *int               `json:"slots_requested,omitempty"`
	SearchTerm     string            `json:"search_term,omitempty"`
	SearchType     string            `json:"search_type,omitempty"`
	Preferred      []*ExtImpPreferred `json:"preferred,omitempty"`
	Filtering      *ExtImpFiltering   `json:"filtering,omitempty"`
	Targeting      []*ExtImpTargeting `json:"targeting,omitempty"`
 }

// ImpExtensionCommerce - Impression Commerce Extension
type ExtImpCommerce struct {
	ComParams   *CommerceParams        `json:"commerce,omitempty"`
	Bidder *ExtBidderCommerce          `json:"bidder,omitempty"`
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

func AddDefaultFields(bid *openrtb2.Bid){
	if bid != nil {
		bid.CrID = "DefaultCRID"
	}
}

func GetRequestSlotCount(internalRequest *openrtb2.BidRequest)int {
	impArray := internalRequest.Imp
	reqCount := 0
	for _, eachImp := range impArray {
		var commerceExt adbuttler.ExtImpCommerce
		json.Unmarshal(eachImp.Ext, &commerceExt)
		reqCount += *commerceExt.ComParams.SlotsRequested
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

func GetDummyBids(impUrl , clickUrl , conversionUrl, seatName string, requestCount int, ImpID string) (*adapters.BidderResponse) {
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
		impID := ImpID + "_" + strconv.Itoa(i)
		bidExt := &adbuttler.ExtBidCommerce{
			ProductId:  &productid,
			ClickPrice: &clikcPrice,
		}
		
		bid := &openrtb2.Bid {
			ID: bidID,
			ImpID: impID,
			Price: bidPrice,
			CID: campaignID,
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
		bidExt := &adbuttler.ExtBidCommerce{
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
	var commerceExt adbuttler.ExtImpCommerce
	json.Unmarshal(internalRequest.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(internalRequest.Imp[0].Ext, &commerceExt)
	return *commerceExt.Bidder.BidderCode
}

// Builder builds a new instance of the AdButtler adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	
	endpointtemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &AdButtlerAdapter{
		endpoint: endpointtemplate,
	}
	return bidder, nil
}

func (a *AdButtlerAdapter) buildEndpointURL(accountID, zoneID string) (string, error) {
	endpointParams := macros.EndpointTemplateParams{
		AccountID: accountID,
		ZoneID:    zoneID,
	}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}
