package native

import (
	"encoding/json"
	"fmt"
	"net/http"

	nativeRequests "github.com/mxmCherry/openrtb/v16/native1/request"
	nativeResponse "github.com/mxmCherry/openrtb/v16/native1/response"
	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint string
	xapiUser string
	xapiPass string
}

func printJson(itemToPrint interface{}) {
	json, err := json.MarshalIndent(itemToPrint, "", "  ")
	if err != nil {
		fmt.Println()
		fmt.Println()
		fmt.Println("Error converting to json")
		fmt.Println()
		fmt.Println()
		fmt.Println(err)
		return
	}
	fmt.Println()
	fmt.Println()
	fmt.Printf("%+v", string(json))
}

type nativeOutbound struct {
	RequestObj nativeRequests.Request `json:"requestobj"`
	Ver        string                 `json:"ver"`
	Api        []int                  `json:"api"`
}

type target struct {
	Context []string `json:"context"`
	Test    []string `json:"test"`
}

type rp struct {
	Target target      `json:"target"`
	ZoneId json.Number `json:"zone_id"`
}

type impExt struct {
	Rp rp `json:"rp"`
}

type impOutbound struct {
	openrtb2.Imp
	Native   nativeOutbound `json:"native"`
	Ext      impExt         `json:"ext"`
	Bidfloor float64        `json:"bidfloor"`
}

type native1point0BidRequest struct {
	openrtb2.BidRequest
	Imp []impOutbound `json:"imp"`
}

type siteExtData struct {
	Context []string `json:"context,omitempty"`
	Test    []string `json:"test,omitempty"`
	Section []string `json:"section,omitempty"`
}

type siteExt struct {
	Data siteExtData `json:"data"`
}

func makeNativeOnePointZeroImpression(imp openrtb2.Imp, siteExt siteExt, nativeImpExt openrtb_ext.ExtImpNative, errors []error) (nativeOutbound, []error) {
	var nativeRequest nativeRequests.Request
	if err := json.Unmarshal([]byte(imp.Native.Request), &nativeRequest); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})
	}

	nativeRequest.Layout = 3
	nativeRequest.EventTrackers = nil
	nativeRequest.Ver = "1.0"
	api := make([]int, 0, 6)
	api = append(api, 1, 2, 3, 4, 5, 6, 7)

	imp.TagID = nativeImpExt.ZoneId.String()
	native := nativeOutbound{
		RequestObj: nativeRequest,
		Ver:        "1.0",
		Api:        api,
	}

	return native, errors

}

// Builder builds a new instance of the Native adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		xapiUser: config.XAPI.Username,
		xapiPass: config.XAPI.Password,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	var onePointTwoImps = make([]openrtb2.Imp, 0, len(request.Imp))
	var onePointZeroImps = make([]impOutbound, 0, len(request.Imp))

	// get site.ext
	var siteExt siteExt
	if err := json.Unmarshal(request.Site.Ext, &siteExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})
	}

	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var nativeImpExt openrtb_ext.ExtImpNative
		if err := json.Unmarshal(bidderExt.Bidder, &nativeImpExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var nativeRequest nativeRequests.Request
		if err := json.Unmarshal([]byte(imp.Native.Request), &nativeRequest); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		impExt := impExt{
			Rp: rp{
				Target: target{
					Context: siteExt.Data.Context,
					Test:    siteExt.Data.Test,
				},
				ZoneId: nativeImpExt.ZoneId,
			},
		}

		// hardcoded values for now, add dynamicism later

		nativeRequest.Layout = 3
		nativeRequest.EventTrackers = nil
		nativeRequest.Ver = "1.0"
		api := make([]int, 0, 6)
		api = append(api, 1, 2, 3, 4, 5, 6, 7)

		imp.TagID = nativeImpExt.ZoneId.String()
		native := nativeOutbound{
			RequestObj: nativeRequest,
			Ver:        "1.0",
			Api:        api,
		}

		onePointZeroImp := impOutbound{
			imp,
			native,
			impExt,
			0.01,
		}

		onePointZeroImps = append(onePointZeroImps, onePointZeroImp)
		onePointTwoImps = append(onePointTwoImps, imp)

	}

	onePointZeroRequest := native1point0BidRequest{
		*request,
		onePointZeroImps,
	}
	onePointZeroRequest.Device.IP = "161.149.146.201"
	onePointZeroRequest.Device.Lmt = new(int8)
	onePointZeroRequest.Ext = nil
	onePointZeroRequest.User.BuyerUID = "L1P293UM-27-4FAD"
	onePointZeroRequest.AT = 0
	// printJson((onePointZeroRequest.Source.Ext))
	onePointZeroRequestJSON, err := json.MarshalIndent(onePointZeroRequest, "", "  ")
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	// printJson(onePointZeroRequest)

	request.Imp = onePointTwoImps

	requestJSON, err := json.MarshalIndent(request, "", "  ")
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
	}

	onePointZeroRequestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    onePointZeroRequestJSON,
		Headers: headers,
	}

	requestData.SetBasicAuth(a.xapiUser, a.xapiPass)
	reqData := make([]*adapters.RequestData, 0)
	reqData = append(reqData, requestData, onePointZeroRequestData)
	return reqData, errors
}

type rubiconNative struct {
	Native nativeResponse.Response `json:"native"`
}
type rubiconBidExtRp struct {
	AdType string      `json:"adtype,omitempty"`
	Advid  json.Number `json:"advid,omitempty"`
	Mime   string      `json:"mime,omitempty"`
	SizeId json.Number `json:"size_id,omitempty"`
}

type rubiconBidExt struct {
	Rp rubiconBidExtRp `json:"rp"`
}

type rubiconBid struct {
	openrtb2.Bid
	Admobject rubiconNative `json:"admobject,omitempty"`
	Ext       rubiconBidExt `json:"ext"`
}

type rubiconSeatBid struct {
	openrtb2.SeatBid
	Buyer string       `json:"buyer,omitempty"`
	Bid   []rubiconBid `json:"bid"`
}

type rubiconBidResponse struct {
	openrtb2.BidResponse
	SeatBid []rubiconSeatBid `json:"seatbid,omitempty"`
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {

	printJson((responseData))
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response rubiconBidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error
	for _, seatBid := range response.SeatBid {
		for _, bid := range seatBid.Bid {
			admString, err := json.Marshal(bid.Admobject.Native)
			if err != nil {
				errors = append(errors, err)
				return nil, errors
			}
			var newBid openrtb2.Bid
			newBid.AdM = string(admString)
			newBid.ID = bid.ID
			newBid.ImpID = bid.ImpID
			newBid.Price = bid.Price
			newBid.NURL = bid.NURL
			newBid.BURL = bid.BURL
			newBid.LURL = bid.LURL
			newBid.AdID = bid.AdID
			newBid.ADomain = bid.ADomain
			newBid.Bundle = bid.Bundle
			newBid.IURL = bid.IURL
			newBid.CID = bid.CID
			newBid.CrID = bid.CrID
			newBid.Tactic = bid.Tactic
			newBid.CatTax = bid.CatTax
			newBid.Cat = bid.Cat
			newBid.Attr = bid.Attr
			newBid.API = bid.API
			newBid.Protocol = bid.Protocol
			newBid.QAGMediaRating = bid.QAGMediaRating
			newBid.Language = bid.Language
			newBid.LangB = bid.LangB
			newBid.DealID = bid.DealID
			newBid.W = bid.W
			newBid.H = bid.H
			newBid.WRatio = bid.WRatio
			newBid.Exp = bid.Exp
			newBid.Dur = bid.Dur
			newBid.MType = bid.MType
			newBid.SlotInPod = bid.SlotInPod

			bidExt, err := json.MarshalIndent(bid.Ext, "", "  ")
			if err != nil {
				errors = append(errors, err)
				return nil, errors
			}
			newBid.Ext = bidExt

			bid.AdM = string(admString)

			bidType, err := getMediaTypeForBid(bid)
			if err != nil {
				errors = append(errors, err)
				continue
			}

			printJson(newBid)
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &newBid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errors
}

func getMediaTypeForBid(bid rubiconBid) (openrtb_ext.BidType, error) {

	if bid.Ext.Rp.AdType != "" {
		return openrtb_ext.ParseBidType(string(bid.Ext.Rp.AdType))

	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}
