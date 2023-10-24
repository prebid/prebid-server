package criteostaples

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type Placement struct {
	Format   string               `json:"format"`
	Products []map[string]interface{} `json:"products"`
	OnLoadBeacon string     `json:"OnLoadBeacon,omitempty"`
	OnViewBeacon string     `json:"OnViewBeacon,omitempty"`
}

type CriteoResponse struct {
	Status             string `json:"status"`
	OnAvailabilityUpdate interface{} `json:"OnAvailabilityUpdate"`
	Placements         []map[string][]Placement `json:"placements"`
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

	criteoResponse, err := newCriteoStaplesResponseFromBytes(response.Body)
	if err != nil {
		return nil, []error{err}
	}

	if criteoResponse.Status != RESPONSE_OK {
		return nil, []error{&errortypes.BidderFailedSchemaValidation{
			Message: "Error Occured at Criteo for the given request ",
		}}
	}

	if  criteoResponse.Placements == nil || len(criteoResponse.Placements) <= 0 {
		return nil, []error{&errortypes.NoBidPrice{
			Message: "No Bid For the given Request",
		}}
	}

	impID := internalRequest.Imp[0].ID
	bidderResponse := a.getBidderResponse(internalRequest, &criteoResponse, impID)
	return bidderResponse, nil
}

func (a *CriteoStaplesAdapter) getBidderResponse(request *openrtb2.BidRequest, criteoResponse *CriteoResponse, requestImpID string) *adapters.BidderResponse {

	noOfBids := countSponsoredProducts(criteoResponse)
	bidResponse := adapters.NewBidderResponseWithBidsCapacity(noOfBids)
    index := 1
	for _, placementMap := range criteoResponse.Placements {
		for _, placements := range placementMap {
			for _, placement := range placements {
				if placement.Format == FORMAT_SPONSORED {
					for _, productMap := range placement.Products {
						bidID := adapters.GenerateUniqueBidIDComm()
						impID := requestImpID + "_" + strconv.Itoa(index)
						bidPrice, _ := strconv.ParseFloat(strings.TrimSpace(productMap[BID_PRICE].(string)), 64)
						clickPrice, _ := strconv.ParseFloat(strings.TrimSpace(productMap[CLICK_PRICE].(string)), 64)
						productID := productMap[PRODUCT_ID].(string)
		
						impressionURL := IMP_KEY + adapters.EncodeURL(productMap[VIEW_BEACON].(string))
						clickURL := CLICK_KEY + adapters.EncodeURL(productMap[CLICK_BEACON].(string))
						index++

						// Add ProductDetails to bidExtension
						productDetails := make(map[string]interface{})
						for key, value := range productMap {
							productDetails[key] = value
						}

						delete(productDetails, PRODUCT_ID)
						delete(productDetails, BID_PRICE)
						delete(productDetails, CLICK_PRICE)
						delete(productDetails, VIEW_BEACON)
						delete(productDetails, CLICK_BEACON)
					
						bidExt := &openrtb_ext.ExtBidCommerce{
							ProductId:     productID,
							ClickUrl:      clickURL,
							ClickPrice:    clickPrice,
							ProductDetails: productDetails,
						}

						bid := &openrtb2.Bid{
							ID:    bidID,
							ImpID: impID,
							Price: bidPrice,
							IURL:  impressionURL,
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
				}
			}
		}
	}
	return bidResponse
}

func newCriteoStaplesResponseFromBytes(bytes []byte) (CriteoResponse, error) {
	var err error
	var bidResponse CriteoResponse

	if err = json.Unmarshal(bytes, &bidResponse); err != nil {
		return bidResponse, err
	}

	return bidResponse, nil
}

func countSponsoredProducts(adResponse* CriteoResponse) int {
	count := 0

	// Iterate through placements
	for _, placementMap := range adResponse.Placements {
		for _, placements := range placementMap {
			for _, placement := range placements {
				if placement.Format == FORMAT_SPONSORED {
					count += len(placement.Products)
				}
			}
		}
	}

	return count
}
