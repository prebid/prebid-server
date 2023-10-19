package adbuttler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
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

func (a *AdButtlerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

    commerceExt, siteExt, errors := adapters.ValidateCommRequest(request)
	if len(errors) > 0 {
		return nil, errors
	}

    var configValueMap = make(map[string]string)
    var configTypeMap = make(map[string]int)
	for _,obj := range commerceExt.Bidder.CustomConfig {
		configValueMap[obj.Key] = obj.Value
		configTypeMap[obj.Key] = obj.Type
	}

	var adButlerReq AdButlerRequest 
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
	//Add Geo Targeting
	if request.Device != nil {
		switch request.Device.DeviceType {
		case 1:
			adButlerReq.Target[DEVICE] = DEVICE_COMPUTER
		case 2:
			adButlerReq.Target[DEVICE] = DEVICE_PHONE
		case 3:
			adButlerReq.Target[DEVICE] = DEVICE_TABLET
		case 4:
			adButlerReq.Target[DEVICE] = DEVICE_CONNECTEDDEVICE
		}
	}

	//Add Page Source Targeting
	if adButlerReq.Source != ""  {
		adButlerReq.Target[PAGE_SOURCE] = adButlerReq.Source
	}

	//Add Dynamic Targeting from AdRequest
	for _,targetObj := range commerceExt.ComParams.Targeting {
		key := targetObj.Name
		adButlerReq.Target[key] = targetObj.Value
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


