package operaads

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
)

type Region string

const (
	USEast Region = "us_east"
	EU     Region = "eu"
	APAC   Region = "apac"
)

type operaAdsVideoExt struct {
	PlacementType string `json:"placementtype"`
	Orientation   string `json:"orientation"`
	Skip          int    `json:"skip"`
	SkipDelay     int    `json:"skipdelay"`
	Rewarded      int    `json:"rewarded"`
}

type operaAdsBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type operaAdsImpExt struct {
	SKADN *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// DeviceType ...
type DeviceType string

const (
	Phone  DeviceType = "phone"
	Tablet DeviceType = "tablet"
)

// SKAN IDs must be lower case
var operaAdsSKADNetIDs = map[string]bool{
	"a2p9lx4jpn.skadnetwork": true,
	"22mmun2rn5.skadnetwork": true,
	"e5fvkxwrpn.skadnetwork": true,
	"8s468mfl3y.skadnetwork": true,
	"7ug5zh24hu.skadnetwork": true,
	"zq492l623r.skadnetwork": true,
}

var (
	errDeviceOrOSMiss = errors.New("impression is missing device OS information")
)

// Builder builds a new instance of the operaads adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &OperaAdsAdapter{
		endpoint: config.Endpoint,
		SupportedRegions: map[Region]string{
			USEast: config.XAPI.EndpointUSEast,
			EU:     config.XAPI.EndpointEU,
			APAC:   config.XAPI.EndpointAPAC,
		},
	}
	return bidder, nil
}

type OperaAdsAdapter struct {
	endpoint         string
	SupportedRegions map[Region]string
}

func (a *OperaAdsAdapter) MakeRequests(
	request *openrtb2.BidRequest,
	_ *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	impCount := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, impCount)
	var errs []error
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	err := checkRequest(request)
	if err != nil {
		errs = append(errs, &errortypes.BadInput{
			Message: err.Error(),
		})
		return nil, errs
	}

	// copy the bidder request
	operaadsRequest := *request

	for _, imp := range operaadsRequest.Imp {
		skanSent := false

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// ExtImpOperaAds
		var operaadsExt openrtb_ext.ExtImpTJXOperaAds
		if err := json.Unmarshal(bidderExt.Bidder, &operaadsExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		uri := a.endpoint
		if endpoint, ok := a.SupportedRegions[Region(operaadsExt.Region)]; ok {
			uri = endpoint
		}

		imp.TagID = ""

		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// default is interstitial
		placementType := adapters.Interstitial
		rewarded := 0
		if operaadsExt.Video.Skip == 0 {
			placementType = adapters.Rewarded
			rewarded = 1
		}

		if imp.Video != nil {
			orientation := Horizontal
			if operaadsExt.Video.Width < operaadsExt.Video.Height {
				orientation = Vertical
			}

			videoCopy := *imp.Video

			videoExt := operaAdsVideoExt{
				PlacementType: string(placementType),
				Orientation:   string(orientation),
				Skip:          operaadsExt.Video.Skip,
				SkipDelay:     operaadsExt.Video.SkipDelay,
				Rewarded:      rewarded,
			}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			imp.Video = &videoCopy
		}

		if imp.Banner != nil {
			if operaadsExt.MRAIDSupported {
				bannerCopy := *imp.Banner

				// custom override for banner sizes
				tempW, tempH := bannerSize(bannerCopy, request.Device)
				bannerCopy.W = &tempW
				bannerCopy.H = &tempH

				bannerExt := operaAdsBannerExt{
					PlacementType:           string(placementType),
					AllowsCustomCloseButton: false,
				}
				bannerCopy.Ext, err = json.Marshal(&bannerExt)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				imp.Banner = &bannerCopy
			} else {
				imp.Banner = nil
			}
		}

		// Overwrite BidFloor if present
		if operaadsExt.BidFloor != nil {
			imp.BidFloor = *operaadsExt.BidFloor
		}

		impExt := operaAdsImpExt{}

		// Add SKADN if supported and present
		if operaadsExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, operaAdsSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				skanSent = true
				impExt.SKADN = &skadn
			}
		}

		imp.ID = buildOperaImpId(imp.ID, openrtb_ext.BidTypeVideo)

		imp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		operaadsRequest.Imp = []openrtb2.Imp{imp}
		operaadsRequest.Cur = nil
		operaadsRequest.Ext = nil

		reqJSON, err := json.Marshal(operaadsRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        string(openrtb_ext.BidderOperaads),
				PlacementType: placementType,
				Region:        operaadsExt.Region,
				SKAN: adapters.SKAN{
					Supported: operaadsExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: operaadsExt.MRAIDSupported,
				},
			},
		}

		requestData = append(requestData, reqData)
	}
	return requestData, errs
}

func checkRequest(request *openrtb2.BidRequest) error {
	if request.Device == nil || len(request.Device.OS) == 0 {
		return errDeviceOrOSMiss
	}

	return nil
}

func buildOperaImpId(originId string, bidType openrtb_ext.BidType) string {
	return strings.Join([]string{originId, "opa", string(bidType)}, ":")
}

const unexpectedStatusCodeFormat = "" +
	"Unexpected status code: %d. Run with request.debug = 1 for more info"

func (a *OperaAdsAdapter) MakeBids(_ *openrtb2.BidRequest, _ *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf(unexpectedStatusCodeFormat, response.StatusCode),
		}}
	}

	var parsedResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &parsedResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(1)
	for _, sb := range parsedResponse.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			if bid.Price != 0 {
				var bidType openrtb_ext.BidType
				bid.ImpID, bidType = parseOriginImpId(bid.ImpID)
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}
	return bidResponse, nil
}

func parseOriginImpId(impId string) (originId string, bidType openrtb_ext.BidType) {
	items := strings.Split(impId, ":")
	if len(items) < 2 {
		return impId, ""
	}
	return strings.Join(items[:len(items)-2], ":"), openrtb_ext.BidType(items[len(items)-1])
}

func bannerSize(banner openrtb2.Banner, device *openrtb2.Device) (int64, int64) {
	orientation := Horizontal
	if banner.W != nil && banner.H != nil && *banner.W < *banner.H {
		orientation = Vertical
	}

	// request.Device.DeviceType is either DeviceTypeTablet or DeviceTypePhone
	// https://github.com/Tapjoy/go-prebid/blob/e9c29cf39d683cf7a4beae55eb922d9a4a5a57d6/prebid/request.go#L563
	deviceType := Phone
	if device != nil && device.DeviceType == openrtb2.DeviceTypeTablet {
		deviceType = Tablet
	}

	switch {
	case orientation == Horizontal && deviceType == Phone:
		return 480, 320

	case orientation == Vertical && deviceType == Phone:
		return 320, 480

	case orientation == Horizontal && deviceType == Tablet:
		return 1024, 768

	case orientation == Vertical && deviceType == Tablet:
		return 768, 1024

	}

	return 0, 0
}
