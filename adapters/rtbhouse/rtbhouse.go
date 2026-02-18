package rtbhouse

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	BidderCurrency string = "USD"
)

// publisherExtPrebid defines the structure for publisher.ext.prebid used by RTBHouse adapter
type publisherExtPrebid struct {
	PublisherId string `json:"publisherId,omitempty"`
}

// publisherExt defines the structure for publisher.ext used by RTBHouse adapter
type publisherExt struct {
	Prebid *publisherExtPrebid `json:"prebid,omitempty"`
}

// RTBHouseAdapter implements the Bidder interface.
type RTBHouseAdapter struct {
	endpoint string
}

// Builder builds a new instance of the RTBHouse adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &RTBHouseAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (adapter *RTBHouseAdapter) MakeRequests(
	openRTBRequest *openrtb2.BidRequest,
	reqInfo *adapters.ExtraRequestInfo,
) (
	requestsToBidder []*adapters.RequestData,
	errs []error,
) {

	reqCopy := *openRTBRequest
	reqCopy.Imp = []openrtb2.Imp{}

	var publisherId string

	for _, imp := range openRTBRequest.Imp {
		var impExtMap map[string]interface{}
		err := jsonutil.Unmarshal(imp.Ext, &impExtMap)
		if err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: "Bidder extension not provided or can't be unmarshalled",
			}}
		}

		rtbhouseExt, err := getImpressionExt(impExtMap)
		if err != nil {
			return nil, []error{err}
		}

		// Extract publisherId from the first impression that has one
		if publisherId == "" && rtbhouseExt.PublisherId != "" {
			publisherId = rtbhouseExt.PublisherId
		}

		var bidFloorCur = imp.BidFloorCur
		var bidFloor = imp.BidFloor
		if bidFloorCur == "" && bidFloor == 0 {
			if rtbhouseExt.BidFloor > 0 {
				bidFloor = rtbhouseExt.BidFloor
				bidFloorCur = BidderCurrency
				if len(reqCopy.Cur) > 0 {
					bidFloorCur = reqCopy.Cur[0]
				}
			}
		}

		// Check if imp comes with bid floor amount defined in a foreign currency
		if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != BidderCurrency {
			// Convert to US dollars
			convertedValue, err := reqInfo.ConvertCurrency(bidFloor, bidFloorCur, BidderCurrency)
			if err != nil {
				return nil, []error{err}
			}

			bidFloorCur = BidderCurrency
			bidFloor = convertedValue
		}

		if bidFloor > 0 && bidFloorCur == BidderCurrency {
			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = bidFloorCur
			imp.BidFloor = bidFloor
		}

		if imp.TagID == "" {
			imp.TagID = getTagIDFromImpExt(impExtMap, imp.ID)
		}

		// remove PAAPI signals
		clearAuctionEnvironment(impExtMap)
		newImpExt, err := json.Marshal(impExtMap)
		if err != nil {
			errs = append(errs, err)
			return nil, errs
		}
		imp.Ext = newImpExt

		// Remove PMP from impression
		imp.PMP = nil

		// Set the CUR of bid to BIDDER_CURRENCY after converting all floors
		reqCopy.Cur = []string{BidderCurrency}
		reqCopy.Imp = append(reqCopy.Imp, imp)
	}

	// Set publisher ID in site.publisher.ext.prebid.publisherId or app.publisher.ext.prebid.publisherId if we found one
	if publisherId != "" {
		if err := setPublisherID(&reqCopy, publisherId); err != nil {
			errs = append(errs, err)
			return nil, errs
		}
	}

	openRTBRequestJSON, err := json.Marshal(reqCopy)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	requestToBidder := &adapters.RequestData{
		Method:  "POST",
		Uri:     adapter.endpoint,
		Body:    openRTBRequestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(reqCopy.Imp),
	}
	requestsToBidder = append(requestsToBidder, requestToBidder)

	return requestsToBidder, errs
}

// setPublisherID sets the publisherId in site.publisher.ext.prebid.publisherId or app.publisher.ext.prebid.publisherId
func setPublisherID(request *openrtb2.BidRequest, publisherId string) error {
	var publisher *openrtb2.Publisher
	if request.Site != nil {
		// Create a copy of the site to avoid modifying the original request
		siteCopy := *request.Site
		request.Site = &siteCopy
		publisher = request.Site.Publisher
	} else if request.App != nil {
		// Create a copy of the app to avoid modifying the original request
		appCopy := *request.App
		request.App = &appCopy
		publisher = request.App.Publisher
	} else {
		// If neither site nor app exists, create a site object
		request.Site = &openrtb2.Site{}
	}

	if publisher != nil {
		// Create a copy of the publisher to avoid modifying the original request
		publisherCopy := *publisher
		publisher = &publisherCopy
	} else {
		publisher = &openrtb2.Publisher{}
	}

	// Set publisherId in publisher.ext.prebid.publisherId using local struct
	var pubExt publisherExt
	if publisher.Ext != nil {
		if err := jsonutil.Unmarshal(publisher.Ext, &pubExt); err != nil {
			return err
		}
	}
	if pubExt.Prebid == nil {
		pubExt.Prebid = &publisherExtPrebid{}
	}
	pubExt.Prebid.PublisherId = publisherId

	publisherExtJSON, err := jsonutil.Marshal(pubExt)
	if err != nil {
		return err
	}
	publisher.Ext = publisherExtJSON

	// Assign the updated publisher back to the appropriate object
	if request.Site != nil {
		request.Site.Publisher = publisher
	} else if request.App != nil {
		request.App.Publisher = publisher
	}

	return nil
}

func clearAuctionEnvironment(impExtMap map[string]interface{}) {
	keysToDelete := []string{"ae", "igs", "paapi"}
	for _, key := range keysToDelete {
		delete(impExtMap, key)
	}
}

func getTagIDFromImpExt(impExtMap map[string]interface{}, impID string) string {
	if gpid, ok := impExtMap["gpid"].(string); ok && gpid != "" {
		return gpid
	}

	dataMap, hasData := impExtMap["data"].(map[string]interface{})
	if hasData {
		// imp.ext.data.adserver.adslot
		if adserver, ok := dataMap["adserver"].(map[string]interface{}); ok {
			if adslot, ok := adserver["adslot"].(string); ok && adslot != "" {
				return adslot
			}
		}

		if pbAdSlot, ok := dataMap["pbadslot"].(string); ok && pbAdSlot != "" {
			return pbAdSlot
		}
	}

	// imp.ID as fallback
	if impID != "" {
		return impID
	}

	return ""
}

func getImpressionExt(impExtMap map[string]interface{}) (*openrtb_ext.ExtImpRTBHouse, error) {
	// Check for bidder parameters in imp.ext.bidder
	bidderVal, ok := impExtMap["bidder"]
	if !ok {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided",
		}
	}

	bidderBytes, err := jsonutil.Marshal(bidderVal)
	if err != nil {
		return nil, &errortypes.BadInput{
			Message: "Bidder extension not provided or can't be unmarshalled",
		}
	}

	var rtbhouseExt openrtb_ext.ExtImpRTBHouse
	if err := jsonutil.Unmarshal(bidderBytes, &rtbhouseExt); err != nil {
		return nil, &errortypes.BadInput{
			Message: "Error while unmarshaling bidder extension",
		}
	}
	return &rtbhouseExt, nil
}

const unexpectedStatusCodeFormat = "" +
	"Unexpected status code: %d. Run with request.debug = 1 for more info"

// MakeBids unpacks the server's response into Bids.
func (adapter *RTBHouseAdapter) MakeBids(
	openRTBRequest *openrtb2.BidRequest,
	requestToBidder *adapters.RequestData,
	bidderRawResponse *adapters.ResponseData,
) (
	bidderResponse *adapters.BidderResponse,
	errs []error,
) {
	switch bidderRawResponse.StatusCode {
	case http.StatusOK:
		break
	case http.StatusNoContent:
		return nil, nil
	case http.StatusBadRequest:
		err := &errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	default:
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, bidderRawResponse.StatusCode),
		}
		return nil, []error{err}
	}

	var openRTBBidderResponse openrtb2.BidResponse
	if err := jsonutil.Unmarshal(bidderRawResponse.Body, &openRTBBidderResponse); err != nil {
		return nil, []error{err}
	}

	bidsCapacity := len(openRTBBidderResponse.SeatBid[0].Bid)
	bidderResponse = adapters.NewBidderResponseWithBidsCapacity(bidsCapacity)
	var typedBid *adapters.TypedBid
	for _, seatBid := range openRTBBidderResponse.SeatBid {
		for _, bid := range seatBid.Bid {
			bid := bid
			bidType, err := getMediaTypeForBid(bid)
			resolveMacros(&bid)
			if err != nil {
				errs = append(errs, err)
				continue
			} else {
				typedBid = &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				}

				// for native bid responses fix Adm field
				if typedBid.BidType == openrtb_ext.BidTypeNative {
					bid.AdM, err = getNativeAdm(bid.AdM)
					if err != nil {
						errs = append(errs, err)
						continue
					}
				}

				bidderResponse.Bids = append(bidderResponse.Bids, typedBid)
			}
		}
	}

	bidderResponse.Currency = BidderCurrency

	return bidderResponse, errs

}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("unrecognized bid type in response from rtbhouse for bid %s", bid.ImpID)
	}
}

func getNativeAdm(adm string) (string, error) {
	nativeAdm := make(map[string]interface{})
	err := jsonutil.Unmarshal([]byte(adm), &nativeAdm)
	if err != nil {
		return adm, errors.New("unable to unmarshal native adm")
	}

	// move bid.adm.native to bid.adm
	if _, ok := nativeAdm["native"]; ok {
		//using jsonparser to avoid marshaling, encode escape, etc.
		value, dataType, _, err := jsonparser.Get([]byte(adm), string(openrtb_ext.BidTypeNative))
		if err != nil || dataType != jsonparser.Object {
			return adm, errors.New("unable to get native adm")
		}
		adm = string(value)
	}

	return adm, nil
}

func resolveMacros(bid *openrtb2.Bid) {
	if bid != nil {
		price := strconv.FormatFloat(bid.Price, 'f', -1, 64)
		bid.NURL = strings.Replace(bid.NURL, "${AUCTION_PRICE}", price, -1)
		bid.AdM = strings.Replace(bid.AdM, "${AUCTION_PRICE}", price, -1)
	}
}
