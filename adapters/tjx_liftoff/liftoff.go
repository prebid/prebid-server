package liftoff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/config"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
	EU     Region = "eu"
	APAC   Region = "apac"
)

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// SKAN IDs must be lower case
var liftoffSKADNetIDs = map[string]bool{
	"7ug5zh24hu.skadnetwork": true,
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (a *adapter) Name() string {
	return "liftoff"
}

type liftoffVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
}

type liftoffBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type liftoffImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type liftoffAppExt struct {
	AppStoreID string `json:"appstoreid"`
}

type modifiedReqParams struct {
	ReqNumber   string
	BidFloor    *float64
	ContentType string
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}
type liftoffBidExt struct {
	AuctionID string `json:"auction_id,omitempty"`
}

type reqExt struct {
	MultiBidEnabled bool `json:"multi_bid_enabled"`
}

var CONTENT_TYPE_MRAID_ONLY = "MRAID"
var CONTENT_TYPE_VIDEO_ONLY = "VIDEO"

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			EU:     config.XAPI.EndpointEU,
			APAC:   config.XAPI.EndpointAPAC,
		},
	}
	return bidder, nil
}

// MakeRequests ...
func (a *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	var srcExt *reqSourceExt
	if request.Source != nil && request.Source.Ext != nil {
		if errSrcExt := json.Unmarshal(request.Source.Ext, &srcExt); errSrcExt != nil {
			return nil, []error{&errortypes.BadInput{
				Message: errSrcExt.Error(),
			}}
		}
	}

	// Extract multi bid enabled flag from request extension
	var reqExt reqExt
	if request.Ext != nil {
		if errReqExt := json.Unmarshal(request.Ext, &reqExt); errReqExt != nil {
			return nil, []error{&errortypes.BadInput{
				Message: errReqExt.Error(),
			}}
		}
	}

	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, len(request.Imp))
	var errReqData []error
	if srcExt != nil && srcExt.HeaderBidding == 1 && reqExt.MultiBidEnabled {
		// Diffrent bid floors for each request
		bf := []float64{0.01, 1, 0.01, 1}

		//Data to be modified for each request
		modifiedParams := []modifiedReqParams{
			{
				ReqNumber:   "1",
				BidFloor:    &bf[0],
				ContentType: CONTENT_TYPE_MRAID_ONLY,
			},
			{
				ReqNumber:   "2",
				BidFloor:    &bf[1],
				ContentType: CONTENT_TYPE_MRAID_ONLY,
			},
			{
				ReqNumber:   "3",
				BidFloor:    &bf[2],
				ContentType: CONTENT_TYPE_VIDEO_ONLY,
			},
			{
				ReqNumber:   "4",
				BidFloor:    &bf[3],
				ContentType: CONTENT_TYPE_VIDEO_ONLY,
			},
		}

		for _, param := range modifiedParams {
			liftoffRequest := *request
			reqData, err := a.makeRequestData(&liftoffRequest, numRequests, param, headers, errs, reqExt.MultiBidEnabled)
			requestData = append(requestData, reqData)
			errReqData = append(errReqData, err...)
		}
	} else {
		liftoffRequest := *request
		modifiedParams := modifiedReqParams{}
		reqData, err := a.makeRequestData(&liftoffRequest, numRequests, modifiedParams, headers, errs, reqExt.MultiBidEnabled)
		requestData = append(requestData, reqData)
		errReqData = append(errReqData, err...)
	}

	return requestData, errReqData
}

// MakeBids ...
func (a *adapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResp openrtb.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	if len(bidResp.SeatBid) == 0 {
		return nil, nil
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(bidResp.SeatBid[0].Bid))

	var bidReq openrtb.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidType := openrtb_ext.BidTypeBanner

	if bidReq.Imp[0].Video != nil {
		bidType = openrtb_ext.BidTypeVideo
	}

	for _, sb := range bidResp.SeatBid {
		for _, b := range sb.Bid {
			if b.Price != 0 {
				// copy response.bidid to openrtb_response.seatbid.bid.bidid
				if b.ID == "0" {
					b.ID = bidResp.BidID
				}

				//Fetch auction id from bid id and add to bid extensions
				auctionID := strings.Split(b.ID, ":")[0]
				injectAuctionID(&b, auctionID)

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &b,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}

func (a *adapter) makeRequestData(liftoffRequest *openrtb.BidRequest, numRequests int, modifiedParams modifiedReqParams, headers http.Header, errs []error, multiBidEnabled bool) (*adapters.RequestData, []error) {
	var err error
	var requestData *adapters.RequestData

	if modifiedParams.ReqNumber != "" {
		liftoffRequest.ID = strings.Join([]string{liftoffRequest.ID, modifiedParams.ReqNumber}, "_")
	}

	// Updating app extension
	if liftoffRequest.App != nil {

		// *liftoffRequest.App creates a copy of the object in appCopy -> correct way.
		// if we do liftoffRequest.App just copies the reference -> Not the correct way because
		// if any of the nested property is changed it change others references to and leads to
		// change in other DSPs bidder requests as well.
		appCopy := *liftoffRequest.App
		appCopy.Ext, err = json.Marshal(liftoffAppExt{
			AppStoreID: liftoffRequest.App.Bundle,
		})
		if err != nil {
			errs = append(errs, err)
		}
		liftoffRequest.App = &appCopy
	}

	requestImpCopy := liftoffRequest.Imp

	for i := 0; i < numRequests; i++ {
		skanSent := false

		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var liftoffExt openrtb_ext.ExtImpTJXLiftoff
		if err = json.Unmarshal(bidderExt.Bidder, &liftoffExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if liftoffExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		// Check for passing mraid oly or rv only if content type is set
		if modifiedParams.ContentType == CONTENT_TYPE_MRAID_ONLY {
			thisImp.Video = nil
		}
		if modifiedParams.ContentType == CONTENT_TYPE_VIDEO_ONLY {
			thisImp.Banner = nil
		}

		if thisImp.Video != nil {
			orientation := Horizontal
			if liftoffExt.Video.Width < liftoffExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *thisImp.Video

			videoExt := liftoffVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          liftoffExt.Video.Skip,
				SkipDelay:     liftoffExt.Video.SkipDelay,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if liftoffExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner
				bannerExt := liftoffBannerExt{
					PlacementType:           string(placementType),
					AllowsCustomCloseButton: false,
				}
				bannerCopy.Ext, err = json.Marshal(&bannerExt)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				thisImp.Banner = &bannerCopy
			} else {
				thisImp.Banner = nil
			}
		}
		// Overwrite BidFloor if present
		if liftoffExt.BidFloor != nil {
			thisImp.BidFloor = *liftoffExt.BidFloor
		}

		// Overwrite BidFloor if coming from modified params
		if modifiedParams.BidFloor != nil {
			thisImp.BidFloor = *modifiedParams.BidFloor
		}

		impExt := liftoffImpExt{
			Rewarded: rewarded,
		}
		// Add SKADN if supported and present
		if liftoffExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, liftoffSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				skanSent = true
				impExt.SKADN = &skadn
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		liftoffRequest.Imp = []openrtb.Imp{thisImp}
		liftoffRequest.Cur = nil
		liftoffRequest.Ext = nil

		reqJSON, err := json.Marshal(liftoffRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		uri := a.endpoint

		if endpoint, ok := a.SupportedRegions[Region(liftoffExt.Region)]; ok {
			uri = endpoint
		}

		reqData := adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        a.Name(),
				ContentType:   strings.ToLower(modifiedParams.ContentType),
				ReqNum:        modifiedParams.ReqNumber,
				PlacementType: placementType,
				Region:        liftoffExt.Region,
				SKAN: adapters.SKAN{
					Supported: liftoffExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: liftoffExt.MRAIDSupported,
				},
				MultiBidEnabled: multiBidEnabled,
			},
		}
		requestData = &reqData
	}
	return requestData, errs
}

func injectAuctionID(bid *openrtb2.Bid, auctionID string) {
	var bidExt liftoffBidExt
	if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
		return
	}

	bidExt.AuctionID = auctionID

	rawBidExt, err := json.Marshal(bidExt)
	if err != nil {
		return
	}

	bid.Ext = rawBidExt
	return
}
