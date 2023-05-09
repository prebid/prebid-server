package adbuttler

import (
	"fmt"
	"text/template"

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
