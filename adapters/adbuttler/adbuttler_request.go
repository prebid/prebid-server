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
	"github.com/prebid/prebid-server/adapters/koddi"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type AdButlerRequest struct {
	SearchString      string            `json:"search,omitempty"`
	SearchType        string            `json:"search_type,omitempty"`
	Params            map[string][]string `json:"params,omitempty"`
	Target            map[string]interface{} `json:"_abdk_json,omitempty"`
	Limit             int               `json:"limit,omitempty"`
	Source            int64             `json:"source,omitempty"`
	UserID            string            `json:"udb_uid,omitempty"`
	IP                string            `json:"ip,omitempty"`
	UserAgent         string            `json:"ua,omitempty"`
	Referrer          string            `json:"referrer,omitempty"`
	FloorCPC          float64           `json:"bid_floor_cpc,omitempty"`
}


func (a *AdButtlerAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var adButlerReq AdButlerRequest 
    var configValueMap = make(map[string]string)
    var configTypeMap = make(map[string]int)
	
	var extension map[string]json.RawMessage
	var preBidExt openrtb_ext.ExtRequestPrebid
	var commerceExt koddi.ExtImpCommerce
	var accountID, zoneID string

	json.Unmarshal(request.Ext, &extension)
	json.Unmarshal(extension["prebid"], &preBidExt)
	json.Unmarshal(request.Imp[0].Ext, &commerceExt)
	
	for _,obj := range commerceExt.Bidder.CustomConfig {
		configValueMap[*obj.Key] = *obj.Value
		configTypeMap[*obj.Key] = *obj.Type
	}

	//Assign Page Source if Present
	val, ok := configValueMap[PAGE_SOURCE]
	if ok {
		if pageSource, err := strconv.ParseInt(val,10, 64); err == nil {
			adButlerReq.Source = pageSource
		}
	}

	adButlerReq.Params = commerceExt.ComParams.Filtering
	adButlerReq.Target = commerceExt.ComParams.Targeting

	if adButlerReq.Target == nil {
		adButlerReq.Target = make(map[string]interface{})
	}

	if request.User != nil {
		if(request.User.Yob > 0) {
			now := time.Now()	
			age := now.Year() - request.User.Yob
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

	//Assign Search Term if present along with searchType
	if commerceExt.ComParams.SearchTerm != "" {
		adButlerReq.SearchString = commerceExt.ComParams.SearchTerm
		val, ok := configValueMap[SEARCHTYPE]
		if ok {
			adButlerReq.SearchType = val
		} else {
			adButlerReq.SearchType = SEARCHTYPE_DEFAULT
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

	/*request.TMax = 0
	customConfig := commerceExt.Bidder.CustomConfig
	//Nobid := false
	for _, eachCustomConfig := range customConfig {
		if *eachCustomConfig.Key == "bidder_timeout"{
				var timeout int
				timeout,_ = strconv.Atoi(*eachCustomConfig.Value)
				request.TMax = int64(timeout)
		}
	}*/

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
