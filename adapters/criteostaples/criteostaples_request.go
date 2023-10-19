package criteostaples

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func getProductList(commerceExt *openrtb_ext.ExtImpCommerce) string {
	// Check if there are preferred products
	if commerceExt != nil && commerceExt.ComParams != nil && len(commerceExt.ComParams.Preferred) > 0 {
		// Initialize a slice to hold the product IDs
		productIDs := make([]string, 0)

		// Iterate through the preferred products and collect their IDs
		for _, preferredProduct := range commerceExt.ComParams.Preferred {
			productIDs = append(productIDs, preferredProduct.ProductID)
		}

		// Join the product IDs with a pipe separator
		return strings.Join(productIDs, "|")
	}

	// Return an empty string if no preferred products are found
	return ""
}

func (a *CriteoStaplesAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
	
	uri, err := url.Parse(a.endpoint)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to parse yieldlab endpoint: %v", err)}
	}

	var criteoPartnerID string
	val, ok := configValueMap[BIDDERDETAILS_PREFIX + BD_ACCOUNT_ID]
	if ok {
		criteoPartnerID = val
	}
	
	values := url.Values{}

	// Add the fields to the query string
	values.Add("criteo-partner-id", criteoPartnerID)
	values.Add("environment", "d")
	values.Add("retailer-visitor-id", request.User.ID)
	values.Add("event-type", "viewHome")
	values.Add("page-id", siteExt.Page)
	values.Add("item-whitelist", getProductList(commerceExt))
	// Add other fields as needed

	criteoQueryString := values.Encode()
	requestURL := uri + "?" + criteoQueryString

	return []*adapters.RequestData{{
		Method:  "GET",
		Uri:     requestURL,
		Headers: http.Header{},
	}}, nil
}


