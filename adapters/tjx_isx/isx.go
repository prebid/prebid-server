package isx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/prebid-server/config"

	openrtb "github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
)

// SKAN IDs must be lower case
var isxSKADNetIDs = map[string]bool{
	"22mmun2rn5.skadnetwork": true,
	"2u9pt9hc89.skadnetwork": true,
	"3qy4746246.skadnetwork": true,
	"3rd42ekr43.skadnetwork": true,
	"424m5254lk.skadnetwork": true,
	"4468km3ulz.skadnetwork": true,
	"44jx6755aq.skadnetwork": true,
	"4fzdc2evr5.skadnetwork": true,
	"578prtvx9j.skadnetwork": true,
	"5tjdwbrq8w.skadnetwork": true,
	"7ug5zh24hu.skadnetwork": true,
	"8s468mfl3y.skadnetwork": true,
	"9t245vhmpl.skadnetwork": true,
	"YCLNXRL5PM.skadnetwork": true,
	"av6w8kgt66.skadnetwork": true,
	"e5fvkxwrpn.skadnetwork": true,
	"f38h382jlk.skadnetwork": true,
	"f7s53z58qe.skadnetwork": true,
	"hs6bdukanm.skadnetwork": true,
	"m8dbw4sv7c.skadnetwork": true,
	"ppxm28t8ap.skadnetwork": true,
	"s39g8k73mm.skadnetwork": true,
	"su67r6k2v3.skadnetwork": true,
	"t38b2kh725.skadnetwork": true,
	"v72qych5uu.skadnetwork": true,
	"zq492l623r.skadnetwork": true,
}

type isxVideoExt struct {
	PlacementType adapters.PlacementType `json:"placementtype"`
}

type isxBannerExt struct {
	PlacementType           adapters.PlacementType `json:"placementtype"`
	AllowsCustomCloseButton bool                   `json:"allowscustomclosebutton"`
}

type isxImpExt struct {
	SKADN *openrtb_ext.SKADN `json:"skadn,omitempty"`
}
type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
}

type adapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (adapter *adapter) Name() string {
	return "isx"
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
		},
	}
	return bidder, nil
}

// MakeRequests ...
func (adapter *adapter) MakeRequests(request *openrtb.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	requestData := make([]*adapters.RequestData, 0, numRequests)

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	errs := make([]error, 0, numRequests)

	// copy the bidder request
	isxRequest := *request

	// clone the request imp array
	requestImpCopy := isxRequest.Imp

	var err error

	for i := 0; i < numRequests; i++ {
		skanSent := false

		// clone current imp
		thisImp := requestImpCopy[i]

		// extract bidder extension
		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// unmarshal bidder extension to isx extension
		var isxExt openrtb_ext.ExtImpTJXISX
		if err = json.Unmarshal(bidderExt.Bidder, &isxExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		isxRequest.BApp = nil
		isxRequest.BAdv = nil
		if isxExt.Blocklist.BApp != nil {
			isxRequest.BApp = isxExt.Blocklist.BApp
		}
		if isxExt.Blocklist.BAdv != nil {
			isxRequest.BAdv = isxExt.Blocklist.BAdv
		}

		// placement type is either Rewarded or Interstitial, default is Interstitial
		placementType := adapters.Interstitial
		if isxExt.PlacementType == string(adapters.Rewarded) {
			placementType = adapters.Rewarded
		}

		if thisImp.Video != nil {
			// instantiate isx video extension struct
			videoExt := isxVideoExt{
				PlacementType: placementType,
			}

			// clone the current video element
			videoCopy := *thisImp.Video

			if isxExt.EndcardHTMLSupported {
				videoCopy.CompanionType = append(videoCopy.CompanionType, openrtb.CompanionTypeHTML)
			}

			// assign isx video extension to cloned video element
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// assign cloned video element to imp object
			thisImp.Video = &videoCopy
		}

		if thisImp.Banner != nil {
			if isxExt.MRAIDSupported {
				bannerCopy := *thisImp.Banner

				bannerExt := isxBannerExt{
					PlacementType:           placementType,
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
		if isxExt.BidFloor != nil {
			thisImp.BidFloor = *isxExt.BidFloor
		}

		// Add impression extensions
		impExt := isxImpExt{}

		// Add SKADN if supported and present
		if isxExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, isxSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				impExt.SKADN = &skadn
				skanSent = true
			}
		}

		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// reinit the values in the request object
		isxRequest.Imp = []openrtb.Imp{thisImp}
		isxRequest.Cur = nil
		isxRequest.Ext = nil

		// clone the request source
		if isxRequest.Source != nil {
			requestSourceCopy := *isxRequest.Source

			// clear PChain
			requestSourceCopy.PChain = ""

			requestSourceExtCopy := requestSourceCopy.Ext

			mapSourceExt := map[string]interface{}{}
			json.Unmarshal([]byte(requestSourceExtCopy), &mapSourceExt)

			// clear SChain
			delete(mapSourceExt, "schain")

			mapSourceExtJson, err := json.Marshal(mapSourceExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			// put source back in the request
			requestSourceCopy.Ext = mapSourceExtJson
			isxRequest.Source = &requestSourceCopy
		}

		// json marshal the request
		reqJSON, err := json.Marshal(isxRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// assign the default uri
		uri := adapter.endpoint

		// assign a region based uri if it exists
		if endpoint, ok := adapter.SupportedRegions[Region(isxExt.Region)]; ok {
			uri = endpoint
		}

		// build request data object
		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        adapter.Name(),
				PlacementType: placementType,
				Region:        isxExt.Region,
				SKAN: adapters.SKAN{
					Supported: isxExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: isxExt.MRAIDSupported,
				},
				Blocklist: adapters.DynamicBlocklist{
					BApp: isxRequest.BApp,
					BAdv: isxRequest.BAdv,
				},
			},
		}

		// append to request data array
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// MakeBids ...
func (adapter *adapter) MakeBids(_ *openrtb.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
				// copy response.bidid or response.id to openrtb_response.seatbid.bid.bidid
				if b.ID == "1" {
					if len(bidResp.BidID) > 0 {
						b.ID = bidResp.BidID
					} else if len(bidResp.ID) > 0 {
						b.ID = bidResp.ID
					}
				}

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &b,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}
