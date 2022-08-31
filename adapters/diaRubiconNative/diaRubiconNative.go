package diaRubiconNative

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

// func printJson(itemToPrint interface{}) {
// 	json, err := json.MarshalIndent(itemToPrint, "", "  ")
// 	if err != nil {
// 		fmt.Println()
// 		fmt.Println()
// 		fmt.Println("Error converting to json")
// 		fmt.Println()
// 		fmt.Println()
// 		fmt.Println(err)
// 		return
// 	}
// 	fmt.Println()
// 	fmt.Println()
// 	fmt.Printf("%+v", string(json))
// }

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
type node struct {
	Asi string `json:"asi"`
	Sid string `json:"sid"`
	Hp  int8   `json:"hp"`
	Rid string `json:"rid"`
}

type schain struct {
	Ver      string `json:"ver"`
	Complete int8   `json:"complete"`
	Nodes    []node `json:"nodes"`
}

type sourceExt struct {
	Schain schain `json:"schain"`
}

type impOutbound struct {
	openrtb2.Imp
	Native   nativeOutbound `json:"native"`
	Ext      impExt         `json:"ext"`
	Bidfloor float64        `json:"bidfloor"`
}

type rubiconBidRequest struct {
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
	var onePointTwoImps = make([]impOutbound, 0, len(request.Imp))
	var onePointZeroImps = make([]impOutbound, 0, len(request.Imp))
	// site ext is used in all requests
	if request.Site == nil {
		errors = append(errors, &errortypes.BadInput{
			Message: "No site details",
		})
		return nil, errors
	}
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
		if imp.Native == nil {
			errors = append(errors, &errortypes.BadInput{
				Message: "Invalid BidType, expected native",
			})
			continue
		}
		// get a 1.2 native object here (default as recieved)
		var onePointTwoNativeRequest nativeRequests.Request

		if err := json.Unmarshal([]byte(imp.Native.Request), &onePointTwoNativeRequest); err != nil {

			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue

		}
		// get another native object, currently 1.2 but transform it to 1.0 below
		var onePointZeroNativeRequest nativeRequests.Request
		if err := json.Unmarshal([]byte(imp.Native.Request), &onePointZeroNativeRequest); err != nil {
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
		// convert 1.2 native to 1.0 here
		// hardcoded values for now, add dynamicism later
		onePointZeroNativeRequest.Layout = 3
		onePointZeroNativeRequest.EventTrackers = nil
		onePointZeroNativeRequest.Ver = "1.0"
		api := make([]int, 0, 6)
		api = append(api, 1, 2, 3, 4, 5, 6, 7)

		imp.TagID = nativeImpExt.ZoneId.String()
		// convert the default request to something that rubicon likes:
		nativeOnePointZero := nativeOutbound{
			RequestObj: onePointZeroNativeRequest,
			Ver:        "1.0",
			Api:        api,
		}
		onePointZeroImp := impOutbound{
			imp,
			nativeOnePointZero,
			impExt,
			0.01,
		}

		// rebuild native 1.2 impression
		nativeOnePointTwo := nativeOutbound{
			RequestObj: onePointTwoNativeRequest,
			Ver:        "1.2",
			Api:        api,
		}
		onePointTwoImp := impOutbound{
			imp,
			nativeOnePointTwo,
			impExt,
			0.01,
		}

		onePointZeroImps = append(onePointZeroImps, onePointZeroImp)
		onePointTwoImps = append(onePointTwoImps, onePointTwoImp)

	}
	if len(errors) != 0 {
		return nil, errors
	}
	// turn the 1.0 imps into a rubicon happy request
	onePointZeroRequest := rubiconBidRequest{
		*request,
		onePointZeroImps,
	}
	// turn the 1.2 imps into a rubicon happy request
	onePointTwoRequest := rubiconBidRequest{
		*request,
		onePointTwoImps,
	}
	// for my dev testing, remove when going live
	// if onePointZeroRequest.Device.IP == "" {
	// 	onePointZeroRequest.Device.IP = "75.54.23.3"
	// }
	// if onePointZeroRequest.User.BuyerUID == "" {
	// 	onePointZeroRequest.User.BuyerUID = "L5QJF35F-1O-MBBY"
	// }
	// not sure if we need this or not, without getting responses it's very hard
	onePointZeroRequest.Ext = nil

	onePointZeroRequest.AT = 0

	var sourceExt sourceExt

	if err := json.Unmarshal(request.Source.Ext, &sourceExt); err != nil {
		errors = append(errors, &errortypes.BadInput{
			Message: err.Error(),
		})
	}

	// make sure request id is in the schain
	for index := range sourceExt.Schain.Nodes {
		if sourceExt.Schain.Nodes[index].Rid == "" {
			sourceExt.Schain.Nodes[index].Rid = request.ID
		}
	}
	// add source.ext to both 1.0 and 1.2 requests
	sourceExtJSON, err := json.Marshal(sourceExt)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}
	onePointZeroRequest.Source.Ext = sourceExtJSON
	onePointTwoRequest.Source.Ext = sourceExtJSON

	// make the json body for 1.0
	onePointZeroRequestJSON, err := json.Marshal(onePointZeroRequest)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	// make the json body for 1.2
	onePointTwoRequestJSON, err := json.Marshal(onePointTwoRequest)
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
		Body:    onePointTwoRequestJSON,
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

type rubiconNativeResponse struct {
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
	Admobject rubiconNativeResponse `json:"admobject,omitempty"`
	Ext       rubiconBidExt         `json:"ext"`
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

			// use json.Marshal and json.Unmarshal to convert my
			// rubicon struct back into a ortb2 struct
			// and put the new admObject as admString on it in the adm position
			bidString, err := json.Marshal(bid)
			if err != nil {
				errors = append(errors, err)
			}

			var newBid openrtb2.Bid
			if err := json.Unmarshal(bidString, &newBid); err != nil {
				errors = append(errors, err)
			}
			newBid.AdM = string(admString)

			bidExt, err := json.Marshal(bid.Ext)
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
