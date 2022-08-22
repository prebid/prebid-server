package liftoff

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	SKADN       RespSKADN `json:"skadn,omitempty"` // prebid shared
	AuctionID   string    `json:"auction_id,omitempty"`
	Imptrackers []string  `json:"imptrackers,omitempty"`
}

// RespSKADN ...
type RespSKADN struct {
	Version    string     `json:"version"`    // Version of SKAdNetwork desired. Must be 2.0 or above.
	Network    string     `json:"network"`    // Ad network identifier used in signature. Should match one of the items in the skadnetids array in the request
	Campaign   string     `json:"campaign"`   // Campaign ID compatible with Apple’s spec. As of 2.0, should be an integer between 1 and 100, expressed as a string
	ITunesItem string     `json:"itunesitem"` // ID of advertiser’s app in Apple’s app store. Should match BidResponse.bid.bundle
	Nonce      string     `json:"nonce"`      // An id unique to each ad response
	SourceApp  string     `json:"sourceapp"`  // ID of publisher’s app in Apple’s app store. Should match BidRequest.imp.ext.skad.sourceapp
	Timestamp  string     `json:"timestamp"`  // Unix time in millis string used at the time of signature
	Signature  string     `json:"signature"`  // SKAdNetwork signature as specified by Apple
	Fidelities []Fidelity `json:"fidelities"` // Supports multiple fidelity types introduced in SKAdNetwork v2.2
}
type Fidelity struct {
	Fidelity  int    `json:"fidelity"`  // The fidelity-type of the attribution to track
	Signature string `json:"signature"` // SKAdNetwork signature as specified by Apple
	Nonce     string `json:"nonce"`     // An id unique to each ad response
	Timestamp string `json:"timestamp"` // Unix time in millis string used at the time of signature
}

type reqExt struct {
	MultiBidSelector int `json:"multi_bid_selector"`
}

var CONTENT_TYPE_MRAID_ONLY = "MRAID"
var CONTENT_TYPE_VIDEO_ONLY = "VIDEO"

var multiBidBFSandMediatorBidFloorExperiemntStartTime = time.Date(2022, time.August, 23, 0, 0, 0, 0, time.UTC)

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

	now := time.Now()

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

	// to check if the request is rewarded or skippable
	var liftoffExt openrtb_ext.ExtImpTJXLiftoff
	if request.Imp != nil {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(request.Imp[0].Ext, &bidderExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: err.Error(),
			}}
		}

		if err := json.Unmarshal(bidderExt.Bidder, &liftoffExt); err != nil {
			return nil, []error{&errortypes.BadInput{
				Message: err.Error(),
			}}
		}
	}

	// MultiBidSelector 0 is for no experiment running
	// MultiBidSelector 1 is for Experiment using BFS values
	// MultiBidSelector 2 is for Experiment using mediator values
	// Header Bidding is for identifying if the request is coming from TJX or AS
	if srcExt != nil && srcExt.HeaderBidding == 1 && reqExt.MultiBidSelector > 0 && now.Before(multiBidBFSandMediatorBidFloorExperiemntStartTime) {

		// Diffrent bid floors for each request
		//iOS Requests Bid Floors
		bfIOSRewardedMraidA := 3.00
		bfIOSRewardedMraidB := 5.00
		bfIOSRewardedMraidC := 7.00
		bfIOSRewardedMraidD := 10.00

		bfIOSSkippableVastA := 3.00
		bfIOSSkippableVastB := 5.00
		bfIOSSkippableVastC := 7.00
		bfIOSSkippableVastD := 10.00
		bfIOSSkippableMraidA := 5.00
		bfIOSSkippableMraidB := 7.00
		bfIOSSkippableMraidC := 10.00
		bfIOSSkippableMraidD := 14.00

		//Android Request Bid Floors
		bfAndroidRewardedVastA := 5.00
		bfAndroidRewardedVastB := 7.00
		bfAndroidRewardedVastC := 10.00
		bfAndroidRewardedVastD := 14.00
		bfAndroidRewardedMraidA := 1.50
		bfAndroidRewardedMraidB := 4.00
		bfAndroidRewardedMraidC := 5.00
		bfAndroidRewardedMraidD := 7.00

		bfAndroidSkippableVastA := 3.00
		bfAndroidSkippableVastB := 5.00
		bfAndroidSkippableVastC := 7.00
		bfAndroidSkippableVastD := 10.00
		bfAndroidSkippableMraidA := 3.00
		bfAndroidSkippableMraidB := 5.00
		bfAndroidSkippableMraidC := 10.00
		bfAndroidSkippableMraidD := 14.00

		var modifiedParams []modifiedReqParams
		if strings.ToLower(request.Device.OS) == "ios" {
			if liftoffExt.Video.Skip == 0 {
				//Data to be modified for each request
				modifiedParams = []modifiedReqParams{
					{
						ReqNumber:   "1",
						BidFloor:    &bfIOSRewardedMraidA,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "2",
						BidFloor:    &bfIOSRewardedMraidB,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "3",
						BidFloor:    &bfIOSRewardedMraidC,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "4",
						BidFloor:    &bfIOSRewardedMraidD,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
				}
			} else {
				modifiedParams = []modifiedReqParams{
					{
						ReqNumber:   "1",
						BidFloor:    &bfIOSSkippableVastA,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "2",
						BidFloor:    &bfIOSSkippableVastB,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "3",
						BidFloor:    &bfIOSSkippableVastC,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "4",
						BidFloor:    &bfIOSSkippableVastD,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "5",
						BidFloor:    &bfIOSSkippableMraidA,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "6",
						BidFloor:    &bfIOSSkippableMraidB,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "7",
						BidFloor:    &bfIOSSkippableMraidC,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "8",
						BidFloor:    &bfIOSSkippableMraidD,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
				}
			}
		} else {
			if liftoffExt.Video.Skip == 0 {
				//Data to be modified for each request
				modifiedParams = []modifiedReqParams{
					{
						ReqNumber:   "1",
						BidFloor:    &bfAndroidRewardedVastA,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "2",
						BidFloor:    &bfAndroidRewardedVastB,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "3",
						BidFloor:    &bfAndroidRewardedVastC,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "4",
						BidFloor:    &bfAndroidRewardedVastD,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "5",
						BidFloor:    &bfAndroidRewardedMraidA,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "6",
						BidFloor:    &bfAndroidRewardedMraidB,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "7",
						BidFloor:    &bfAndroidRewardedMraidC,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "8",
						BidFloor:    &bfAndroidRewardedMraidD,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
				}
			} else {
				modifiedParams = []modifiedReqParams{
					{
						ReqNumber:   "1",
						BidFloor:    &bfAndroidSkippableVastA,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "2",
						BidFloor:    &bfAndroidSkippableVastB,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "3",
						BidFloor:    &bfAndroidSkippableVastC,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "4",
						BidFloor:    &bfAndroidSkippableVastD,
						ContentType: CONTENT_TYPE_VIDEO_ONLY,
					},
					{
						ReqNumber:   "5",
						BidFloor:    &bfAndroidSkippableMraidA,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "6",
						BidFloor:    &bfAndroidSkippableMraidB,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "7",
						BidFloor:    &bfAndroidSkippableMraidC,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
					{
						ReqNumber:   "8",
						BidFloor:    &bfAndroidSkippableMraidD,
						ContentType: CONTENT_TYPE_MRAID_ONLY,
					},
				}
			}
		}

		for _, param := range modifiedParams {
			liftoffRequest := *request
			reqData, err := a.makeRequestData(&liftoffRequest, numRequests, param, headers, errs, reqExt.MultiBidSelector)
			requestData = append(requestData, reqData)
			errReqData = append(errReqData, err...)
		}
	} else if srcExt != nil && srcExt.HeaderBidding == 1 && reqExt.MultiBidSelector > 0 && now.After(multiBidBFSandMediatorBidFloorExperiemntStartTime) {
		bidFloorA := *liftoffExt.BidFloor
		bidFloorB := *liftoffExt.BidFloor + 1
		bidFloorC := *liftoffExt.BidFloor + 2
		bidFloorD := *liftoffExt.BidFloor + 3
		modifiedParams := []modifiedReqParams{
			{
				ReqNumber: "1",
				BidFloor:  &bidFloorA,
			},
			{
				ReqNumber: "2",
				BidFloor:  &bidFloorB,
			},
			{
				ReqNumber: "3",
				BidFloor:  &bidFloorC,
			},
			{
				ReqNumber: "4",
				BidFloor:  &bidFloorD,
			},
		}
		for _, param := range modifiedParams {
			liftoffRequest := *request
			reqData, err := a.makeRequestData(&liftoffRequest, numRequests, param, headers, errs, reqExt.MultiBidSelector)
			requestData = append(requestData, reqData)
			errReqData = append(errReqData, err...)
		}
	} else {
		liftoffRequest := *request
		modifiedParams := modifiedReqParams{}
		reqData, err := a.makeRequestData(&liftoffRequest, numRequests, modifiedParams, headers, errs, reqExt.MultiBidSelector)
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

func (a *adapter) makeRequestData(liftoffRequest *openrtb.BidRequest, numRequests int, modifiedParams modifiedReqParams, headers http.Header, errs []error, multiBidSelector int) (*adapters.RequestData, []error) {
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

		traceName := "No Experiemnt"
		if multiBidSelector == 1 {
			traceName = "BSF_EXP_" + modifiedParams.ReqNumber
		}
		if multiBidSelector == 2 {
			traceName = "MEDIATOR_EXP_" + modifiedParams.ReqNumber
		}

		reqData := adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        a.Name(),
				TraceName:     strings.ToLower(traceName),
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
				MultiBidSelector: multiBidSelector,
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
