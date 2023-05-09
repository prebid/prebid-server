package adbuttler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdButlerRequest struct { 
	SearchString      string                  `json:"search,omitempty"`
	SearchType        string                  `json:"search_type,omitempty"`
	Params            map[string][]string     `json:"params,omitempty"`
	Identifiers       []string                `json:"identifiers,omitempty"`
	Target            map[string]interface{}  `json:"_abdk_json,omitempty"`
	Limit             int                     `json:"limit,omitempty"`
	Source            int64                   `json:"source,omitempty"`
	UserID            string                  `json:"udb_uid,omitempty"`
	IP                string                  `json:"ip,omitempty"`
	UserAgent         string                  `json:"ua,omitempty"`
	Referrer          string                  `json:"referrer,omitempty"`
	FloorCPC          float64                 `json:"bid_floor_cpc,omitempty"`
}


func (a *AdButtlerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adButlerReq AdButlerRequest 
    var configValueMap = make(map[string]string)
    var configTypeMap = make(map[string]int)
	
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt ExtImpCommerce
	var accountID, zoneID string

	adButlerReq.Target = make(map[string]interface{})

	json.Unmarshal(request.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(request.Imp[0].Ext, &commerceExt)
	
	for _,obj := range commerceExt.Bidder.CustomConfig {
		configValueMap[obj.Key] = obj.Value
		configTypeMap[obj.Key] = obj.Type
	}

	//Assign Page Source if Present
	val, ok := configValueMap[PAGE_SOURCE]
	if ok {
		if pageSource, err := strconv.ParseInt(val,10, 64); err == nil {
			adButlerReq.Source = pageSource
		}
	}

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

	//Add Dynamic Targeting from AdRequest
	for _,targetObj := range commerceExt.ComParams.Targeting {
		key := targetObj.Name
		datatype := targetObj.Type

		switch datatype {
			case DATATYE_INT:
				value, err := strconv.ParseInt(targetObj.Value, 10, 64)
				if err == nil {
					adButlerReq.Target[key] = value
				}
			
			case DATATYE_FLOAT:
				value, err := strconv.ParseFloat(targetObj.Value, 64)
				if err == nil {
					adButlerReq.Target[key] = value
				}
		  
		    case DATATYE_STRING:
				adButlerReq.Target[key] = targetObj.Value

			case DATATYE_BOOL:
				if targetObj.Value == "true" {
					adButlerReq.Target[key] = true
				} else if targetObj.Value == "false" {
					adButlerReq.Target[key] = false
				}
		}
	}

	//Add Identifiers from AdRequest
	for _,prefObj := range commerceExt.ComParams.Preferred {
		adButlerReq.Identifiers = append(adButlerReq.Identifiers, prefObj.ProductID)
	}

	//Add Category Params from AdRequest
	if commerceExt.ComParams.Filtering != nil {
		adButlerReq.Params = make(map[string][]string)
		if commerceExt.ComParams.Filtering.Category != nil && len(commerceExt.ComParams.Filtering.Category) > 0 {
			adButlerReq.Params[CATEGORY] = commerceExt.ComParams.Filtering.Category
		}

		if commerceExt.ComParams.Filtering.Brand != nil && len(commerceExt.ComParams.Filtering.Brand) > 0 {
			adButlerReq.Params[BRAND] = commerceExt.ComParams.Filtering.Brand
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
	if commerceExt.ComParams.SearchTerm != "" {
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
		val, ok := configValueMap[FLOOR_PRICE]
		if ok {
			if floorPrice, err := strconv.ParseFloat(val, 64); err == nil {
				adButlerReq.FloorCPC = floorPrice
			}
		}
	}
	adButlerReq.UserID = request.User.ID
	adButlerReq.UserAgent = request.Device.UA
	adButlerReq.Limit = *commerceExt.ComParams.SlotsRequested

	u, _ := json.Marshal(adButlerReq)
	fmt.Println(string(u))

	//Assign Page Source if Present
	val, ok = configValueMap[ACCOUNT_ID]
	if ok {
		accountID = val
	}

	val, ok = configValueMap[ZONE_ID]
	if ok {
		zoneID = val
	} 
	
	endPoint,_ := a.buildEndpointURL(accountID, zoneID)
	errs := make([]error, 0, len(request.Imp))

	reqJSON, err := json.Marshal(adButlerReq)
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
