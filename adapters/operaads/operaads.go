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
}

type operaAdsBannerExt struct {
	PlacementType           string `json:"placementtype"`
	AllowsCustomCloseButton bool   `json:"allowscustomclosebutton"`
}

type operaAdsImpExt struct {
	Rewarded int                `json:"rewarded"`
	SKADN    *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

// Orientation ...
type Orientation string

const (
	Horizontal Orientation = "h"
	Vertical   Orientation = "v"
)

// SKAN IDs must be lower case
var operaAdsSKADNetIDs = map[string]bool{
	"a2p9lx4jpn.skadnetwork": true,
}

var (
	errBannerFormatMiss = errors.New("size information missing for banner")
	errDeviceOrOSMiss   = errors.New("impression is missing device OS information")
)

// Builder builds a new instance of the operaads adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
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
	reqInfo *adapters.ExtraRequestInfo,
) (
	[]*adapters.RequestData,
	[]error,
) {
	impCount := len(request.Imp)
	requestData := make([]*adapters.RequestData, 0, impCount)
	errs := []error{}
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

	for _, imp := range request.Imp {
		skanSent := false

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// ExtImpOperaAds
		var operaadsExt openrtb_ext.ExtImpOperaAds
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

		formats := make([]interface{}, 0, 1)
		if imp.Native != nil {
			formats = append(formats, imp.Native)
		}
		if imp.Video != nil {
			formats = append(formats, imp.Video)
		}
		if imp.Banner != nil {
			formats = append(formats, imp.Banner)
		}

		impExt := operaAdsImpExt{
			Rewarded: rewarded,
		}

		// Add SKADN if supported and present
		if operaadsExt.SKADNSupported {
			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, operaAdsSKADNetIDs)
			if len(skadn.SKADNetIDs) > 0 {
				skanSent = true
				impExt.SKADN = &skadn
			}
		}

		imp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, format := range formats {
			req, err := flatImp(*request, imp, headers, uri, format, placementType, operaadsExt, skanSent)
			if err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}
			if req != nil {
				requestData = append(requestData, req)
			}
		}
	}
	return requestData, errs
}

func flatImp(
	requestCopy openrtb2.BidRequest,
	impCopy openrtb2.Imp,
	headers http.Header,
	uri string,
	format interface{},
	placementType adapters.PlacementType,
	operaAdsExt openrtb_ext.ExtImpOperaAds,
	skanSent bool,
) (
	*adapters.RequestData, error,
) {
	switch format.(type) {

	case *openrtb2.Banner:
		impCopy.ID = buildOperaImpId(impCopy.ID, openrtb_ext.BidTypeBanner)
		impCopy.Native = nil
		impCopy.Video = nil
		impCopy.TagID = operaAdsExt.PlacementId.Banner // TagID must be overwritten by correct placementID for each request type.

	// Native is not supported
	case *openrtb2.Native:
		impCopy.Banner = nil
		impCopy.ID = buildOperaImpId(impCopy.ID, openrtb_ext.BidTypeNative)
		impCopy.Video = nil
		impCopy.TagID = operaAdsExt.PlacementId.Native

	case *openrtb2.Video:
		impCopy.Banner = nil
		impCopy.Native = nil
		impCopy.ID = buildOperaImpId(impCopy.ID, openrtb_ext.BidTypeVideo)
		impCopy.TagID = operaAdsExt.PlacementId.Video // TagID must be overwritten by dedicated placementID for each request type

	default: // do not need flat
		return nil, nil
	}
	err := convertImpression(&impCopy)
	if err != nil {
		return nil, err
	}
	requestCopy.Imp = []openrtb2.Imp{impCopy}
	reqJSON, err := json.Marshal(&requestCopy)
	if err != nil {
		return nil, err
	}
	return &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     uri,
		Body:    reqJSON,
		Headers: headers,

		TapjoyData: adapters.TapjoyData{
			Bidder:        string(openrtb_ext.BidderOperaAds),
			PlacementType: placementType,
			Region:        operaAdsExt.Region,
			SKAN: adapters.SKAN{
				Supported: operaAdsExt.SKADNSupported,
				Sent:      skanSent,
			},
			MRAID: adapters.MRAID{
				Supported: operaAdsExt.MRAIDSupported,
			},
		},
	}, nil
}

func checkRequest(request *openrtb2.BidRequest) error {
	if request.Device == nil || len(request.Device.OS) == 0 {
		return errDeviceOrOSMiss
	}

	return nil
}

func convertImpression(imp *openrtb2.Imp) error {
	if imp.Banner != nil {
		bannerCopy, err := convertBanner(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
	}
	if imp.Native != nil && imp.Native.Request != "" {
		v := make(map[string]interface{})
		err := json.Unmarshal([]byte(imp.Native.Request), &v)
		if err != nil {
			return err
		}
		_, ok := v["native"]
		if !ok {
			body, err := json.Marshal(struct {
				Native interface{} `json:"native"`
			}{
				Native: v,
			})
			if err != nil {
				return err
			}
			native := *imp.Native
			native.Request = string(body)
			imp.Native = &native
		}
	}
	return nil
}

// make sure that banner has openrtb 2.3-compatible size information
func convertBanner(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W == nil || banner.H == nil || *banner.W == 0 || *banner.H == 0 {
		if len(banner.Format) > 0 {
			f := banner.Format[0]
			bannerCopy := *banner
			bannerCopy.W = openrtb2.Int64Ptr(f.W)
			bannerCopy.H = openrtb2.Int64Ptr(f.H)
			return &bannerCopy, nil
		} else {
			return nil, errBannerFormatMiss
		}
	}
	return banner, nil
}

const unexpectedStatusCodeFormat = "" +
	"Unexpected status code: %d. Run with request.debug = 1 for more info"

func (a *OperaAdsAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

func buildOperaImpId(originId string, bidType openrtb_ext.BidType) string {
	return strings.Join([]string{originId, "opa", string(bidType)}, ":")
}

func parseOriginImpId(impId string) (originId string, bidType openrtb_ext.BidType) {
	items := strings.Split(impId, ":")
	if len(items) < 2 {
		return impId, ""
	}
	return strings.Join(items[:len(items)-2], ":"), openrtb_ext.BidType(items[len(items)-1])
}
