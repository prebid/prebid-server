package criteostaples

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/PubMatic-OpenWrap/prebid-server/errortypes"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type product struct {
	ProductName   string `json:"ProductName,omitempty"`
	Image         string `json:"Image,omitempty"`
	ProductPage   string `json:"ProductPage,omitempty"`
	ComparePrice  string `json:"ComparePrice,omitempty"`
	Price         string `json:"Price,omitempty"`
	Rating        string `json:"Rating,omitempty"`
	RendAttr      string `json:"RenderingAttributes,omitempty"`
	AdId          string `json:"adid,omitempty"`
	ShortDscrp    string `json:"shortDescription,omitempty"`
	MatchType     string `json:"MatchType,omitempty"`
	ParentSKU     string `json:"ParentSKU,omitempty"`
	OnViewBeacon  string `json:"OnViewBeacon,omitempty"`
	OnClickBeacon string `json:"OnClickBeacon,omitempty"`
	ProductID     string `json:"ProductID,omitempty"`
}

type item struct {
	Format       string     `json:"format,omitempty"`
	Products     []*product `json:"products,omitempty"`
	OnLoadBeacon string     `json:"OnLoadBeacon,omitempty"`
	OnViewBeacon string     `json:"OnViewBeacon,omitempty"`
}

type placement struct {
	Items []*item `json:"viewItem_API_Rec_desktop-Carousel,omitempty"`
}

type criteoStaplesResponse struct {
	Status       string       `json:"status,omitempty"`
	OnAvalUpdate string       `json:"OnAvailabilityUpdate,omitempty"`
	Placements   []*placement `json:"placements,omitempty"`
}

func (a *CriteoStaplesAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d", response.StatusCode),
		}}
	}

	criteoStaplesResponse, err := newCriteoStaplesResponseFromBytes(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	if criteoStaplesResponse.Status != RESPONSE_OK {
		return nil, []error{&errortypes.BidderFailedSchemaValidation{
			Message: "Error Occured at Criteo for the given request with ErrorCode",
		}}
	}

	if criteoStaplesResponse.Placements == nil || len(criteoStaplesResponse.Placements) <= 0 {
		return nil, []error{&errortypes.NoBidPrice{
			Message: "No Placement For the given Request",
		}}
	}

	var products []*product
	var itemPresent bool
	for _, placement := range criteoStaplesResponse.Placements {
		if placement.Items != nil && len(placement.Items) > 0 {
			itemPresent = true
			for _, item := range placement.Items {
				if item.Products != nil && len(item.Products) > 0 {
					products = append(products, item.Products...)
				}
			}
		}
	}

	if !itemPresent {
		return nil, []error{&errortypes.NoBidPrice{
			Message: "No Item For the given Request",
		}}
	}

	if len(products) <= 0 {
		return nil, []error{&errortypes.NoBidPrice{
			Message: "No Bid For the given Request",
		}}
	}

	impID := internalRequest.Imp[0].ID
	bidderResponse := a.getBidderResponse(internalRequest, &criteoStaplesResponse, impID, products)
	return bidderResponse, nil
}

func (a *CriteoStaplesAdapter) getBidderResponse(request *openrtb2.BidRequest, response *criteoStaplesResponse, requestImpID string, products []*product) *adapters.BidderResponse {

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(products))
	var adbutlerID, zoneID, adbUID string
	var configValueMap = make(map[string]string)

	if len(request.Imp) > 0 {
		commerceExt, _ := adapters.GetImpressionExtComm(&(request.Imp[0]))
		for _, obj := range commerceExt.Bidder.CustomConfig {
			configValueMap[obj.Key] = obj.Value
		}

		val, ok := configValueMap[BIDDERDETAILS_PREFIX+BD_ACCOUNT_ID]
		if ok {
			adbutlerID = val
		}

		val, ok = configValueMap[BIDDERDETAILS_PREFIX+BD_ZONE_ID]
		if ok {
			zoneID = val
		}
		adbUID = request.User.ID

	}

	for index, bid := range products {

		bidID := adapters.GenerateUniqueBidIDComm()
		impID := requestImpID + "_" + strconv.Itoa(index+1)
		bidPrice, _ := strconv.ParseFloat(strings.TrimSpace(bid.Price), 64)
		clickPrice, _ := strconv.ParseFloat(strings.TrimSpace(bid.ComparePrice), 64)
		campaignID := bid.AdId
		productid := bid.ProductID

		impressionUrl := IMP_KEY + adapters.EncodeURl(bid.OnViewBeacon)
		clickUrl := CLICK_KEY + adapters.EncodeURl(bid.OnClickBeacon)
		conversionUrl := adapters.GenerateConversionUrl(adbutlerID, zoneID, adbUID, productid)

		bidExt := &openrtb_ext.ExtBidCommerce{
			ProductId:     productid,
			ClickUrl:      clickUrl,
			ClickPrice:    clickPrice,
			ConversionUrl: conversionUrl,
		}

		bid := &openrtb2.Bid{
			ID:    bidID,
			ImpID: impID,
			Price: bidPrice,
			CID:   campaignID,
			IURL:  impressionUrl,
		}

		adapters.AddDefaultFieldsComm(bid)

		bidExtJSON, err1 := json.Marshal(bidExt)
		if nil == err1 {
			bid.Ext = json.RawMessage(bidExtJSON)
		}

		seat := openrtb_ext.BidderName(SEAT_CRITEO)

		typedbid := &adapters.TypedBid{
			Bid:  bid,
			Seat: seat,
		}
		bidResponse.Bids = append(bidResponse.Bids, typedbid)
	}
	return bidResponse
}

func newCriteoStaplesResponseFromBytes(bytes []byte) (criteoStaplesResponse, error) {
	var err error
	var bidResponse criteoStaplesResponse

	if err = json.Unmarshal(bytes, &bidResponse); err != nil {
		return bidResponse, err
	}

	return bidResponse, nil
}
