package adbuttler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdButlerRequest struct { 
	SearchString      string                  `json:"search,omitempty"`
	SearchType        string                  `json:"search_type,omitempty"`
	Params            map[string][]string     `json:"params,omitempty"`
	Identifiers       []string                `json:"identifiers,omitempty"`
	Target            map[string]interface{}  `json:"_abdk_json,omitempty"`
	Limit             int                     `json:"limit,omitempty"`
	Source            string                  `json:"source,omitempty"`
	UserID            string                  `json:"adb_uid,omitempty"`
	IP                string                  `json:"ip,omitempty"`
	UserAgent         string                  `json:"ua,omitempty"`
	Referrer          string                  `json:"referrer,omitempty"`
	FloorCPC          float64                 `json:"bid_floor_cpc,omitempty"`
	IsTestRequest     bool                    `json:"test_request,omitempty"`
	
}

func (a *AdButtlerAdapter) getImpressionExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpCommerce, error) {
	var commerceExt openrtb_ext.ExtImpCommerce
	if err := json.Unmarshal(imp.Ext, &commerceExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Impression extension not provided or can't be unmarshalled",
		}
	}

	return &commerceExt, nil

}


func (a *AdButtlerAdapter) getSiteExt(request *openrtb2.BidRequest) (*openrtb_ext.ExtSiteCommerce, error) {
	var siteExt openrtb_ext.ExtSiteCommerce

	if request.Site.Ext != nil {
		if err := json.Unmarshal(request.Site.Ext, &siteExt); err != nil {
			return nil, &errortypes.BadInput{
				Message: "Impression extension not provided or can't be unmarshalled",
			}
		}
	}

	return &siteExt, nil

}

func (a *AdButtlerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var commerceExt *openrtb_ext.ExtImpCommerce
	var siteExt *openrtb_ext.ExtSiteCommerce
	var err error
	var errors []error

	if len(request.Imp) > 0 {
		commerceExt, err = a.getImpressionExt(&(request.Imp[0]))
		if err != nil {
			errors = append(errors, err)
		}
	} else {
		errors = append(errors, &errortypes.BadInput{
			Message: "Missing Imp Object",
		})
	}

	siteExt, err = a.getSiteExt(request)
	if err != nil {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, errors
	}

	var adButlerReq AdButlerRequest 
    var configValueMap = make(map[string]string)
    var configTypeMap = make(map[string]int)
	for _,obj := range commerceExt.Bidder.CustomConfig {
		configValueMap[obj.Key] = obj.Value
		configTypeMap[obj.Key] = obj.Type
	}

	//Assign Page Source if Present
	if siteExt != nil {
		adButlerReq.Source = siteExt.Page
	}

    //Retrieve AccountID and ZoneID from Request and Build endpoint Url
	var accountID, zoneID string
	val, ok := configValueMap[BIDDERDETAILS_PREFIX + BD_ACCOUNT_ID]
	if ok {
		accountID = val
	}
	
	val, ok = configValueMap[BIDDERDETAILS_PREFIX + BD_ZONE_ID]
	if ok {
		zoneID = val
	} 
		
	endPoint, err := a.buildEndpointURL(accountID, zoneID)
	if err != nil {
		return nil, []error{err}
	}
		
	adButlerReq.Target = make(map[string]interface{})
	//Add User Targeting
	if request.User != nil {
		if(request.User.Yob > 0) {
			now := time.Now()	
			age := int64(now.Year()) - request.User.Yob
			adButlerReq.Target[USER_AGE] = age
		}

		if request.User.Gender != "" {
			if strings.EqualFold(request.User.Gender, "M") {
				adButlerReq.Target[USER_GENDER] = GENDER_MALE
			} else if strings.EqualFold(request.User.Gender, "F") {
				adButlerReq.Target[USER_GENDER] = GENDER_FEMALE
			} else if strings.EqualFold(request.User.Gender, "O") {
				adButlerReq.Target[USER_GENDER] = GENDER_OTHER
			}
		}	
	}

	//Add Geo Targeting
	if request.Device != nil && request.Device.Geo != nil {
		if request.Device.Geo.Country != "" {
			adButlerReq.Target[COUNTRY] = request.Device.Geo.Country
		}
		if request.Device.Geo.Region != "" {
			adButlerReq.Target[REGION] = request.Device.Geo.Region
		}
		if request.Device.Geo.City != "" {
			adButlerReq.Target[CITY] = request.Device.Geo.City
		}
	}


	//Add Page Source Targeting
	if adButlerReq.Source != ""  {
		adButlerReq.Target[PAGE_SOURCE] = adButlerReq.Source
	}

	//Add Dynamic Targeting from AdRequest
	for _,targetObj := range commerceExt.ComParams.Targeting {
		key := targetObj.Name
		datatype := targetObj.Type
        if len(targetObj.Value) > 0 {
			switch datatype {
				case DATATYE_NUMBER, DATATYE_STRING, DATATYE_DATETIME, DATATYE_DATE, DATATYE_TIME :
					adButlerReq.Target[key] = targetObj.Value[0]
		    	case DATATYE_ARRAY:
					if len(targetObj.Value) == 1 {
						adButlerReq.Target[key] = targetObj.Value[0]
					} else {
						adButlerReq.Target[key] = targetObj.Value
					}
			}
		}
	}
	//Add Identifiers from AdRequest
	for _,prefObj := range commerceExt.ComParams.Preferred {
		adButlerReq.Identifiers = append(adButlerReq.Identifiers, prefObj.ProductID)
	}

	//Add Category Params from AdRequest
	if len(adButlerReq.Identifiers) <= 0 && commerceExt.ComParams.Filtering != nil {
		adButlerReq.Params = make(map[string][]string)
		if commerceExt.ComParams.Filtering.Category != nil && len(commerceExt.ComParams.Filtering.Category) > 0 {
			//Retailer Specific Category  Name is present from Product Feed Template
			val, ok = configValueMap[PRODUCTTEMPLATE_PREFIX + PD_TEMPLATE_CATEGORY]
			if ok {
				adButlerReq.Params[val] = commerceExt.ComParams.Filtering.Category
			} else {
				adButlerReq.Params[DEFAULT_CATEGORY] = commerceExt.ComParams.Filtering.Category
			}
		}

		if commerceExt.ComParams.Filtering.Brand != nil && len(commerceExt.ComParams.Filtering.Brand) > 0 {
		    //Retailer Specific Brand Name is present from Product Feed Template
			val, ok = configValueMap[PRODUCTTEMPLATE_PREFIX + PD_TEMPLATE_BRAND]
			if ok {
				adButlerReq.Params[val] = commerceExt.ComParams.Filtering.Brand
			} else {
				adButlerReq.Params[DEFAULT_BRAND] = commerceExt.ComParams.Filtering.Brand
			}
		}

		if commerceExt.ComParams.Filtering.SubCategory != nil {
			for _,subCategory := range commerceExt.ComParams.Filtering.SubCategory {
				key := subCategory.Name
				value := subCategory.Value
				adButlerReq.Params[key] = value
			}
		}
	}
	

	//Assign Search Term if present along with searchType
	if len(adButlerReq.Identifiers) <= 0 && commerceExt.ComParams.Filtering == nil && commerceExt.ComParams.SearchTerm != "" {
		adButlerReq.SearchString = commerceExt.ComParams.SearchTerm
		if commerceExt.ComParams.SearchType == SEARCHTYPE_EXACT ||
		    commerceExt.ComParams.SearchType == SEARCHTYPE_BROAD {
				adButlerReq.SearchType = commerceExt.ComParams.SearchType
		} else {
			val, ok := configValueMap[SEARCHTYPE]
			if ok {
				adButlerReq.SearchType = val
			} else {
				adButlerReq.SearchType = SEARCHTYPE_DEFAULT
			}
		}
	}

	adButlerReq.IP = request.Device.IP
	// Domain Name from Site Object if Prsent or App Obj
	if request.Site != nil {
		adButlerReq.Referrer = request.Site.Domain
	} else {
		adButlerReq.Referrer = request.App.Domain
	}

	// Take BidFloor from BidRequest - High Priority, Otherwise from Auction Config
	if request.Imp[0].BidFloor > 0 {
		adButlerReq.FloorCPC = request.Imp[0].BidFloor
	} else {
		val, ok := configValueMap[AUCTIONDETAILS_PREFIX + AD_FLOOR_PRICE]
		if ok {
			if floorPrice, err := strconv.ParseFloat(val, 64); err == nil {
				adButlerReq.FloorCPC = floorPrice
			}
		}
	}

	//Test Request
	if commerceExt.ComParams.TestRequest {
		adButlerReq.IsTestRequest = true
	}
	adButlerReq.UserID = request.User.ID
	adButlerReq.UserAgent = request.Device.UA
	adButlerReq.Limit = commerceExt.ComParams.SlotsRequested

	//Temporarily for Debugging
	//u, _ := json.Marshal(adButlerReq)
	//fmt.Println(string(u))

	reqJSON, err := json.Marshal(adButlerReq)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json")

	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     endPoint,
		Body:    reqJSON,
		Headers: headers,
	}}, nil
	
}




