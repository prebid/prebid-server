package ix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/version"

	"github.com/prebid/openrtb/v19/native1"
	native1response "github.com/prebid/openrtb/v19/native1/response"
	"github.com/prebid/openrtb/v19/openrtb2"
)

type IxAdapter struct {
	URI string
}

type ExtRequest struct {
	Prebid *openrtb_ext.ExtRequestPrebid `json:"prebid"`
	SChain *openrtb2.SupplyChain         `json:"schain,omitempty"`
	IxDiag *IxDiag                       `json:"ixdiag,omitempty"`
}

type IxDiag struct {
	PbsV            string `json:"pbsv,omitempty"`
	PbjsV           string `json:"pbjsv,omitempty"`
	MultipleSiteIds string `json:"multipleSiteIds,omitempty"`
}

func (a *IxAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requests := make([]*adapters.RequestData, 0, len(request.Imp))
	errs := make([]error, 0)

	headers := http.Header{
		"Content-Type": {"application/json;charset=utf-8"},
		"Accept":       {"application/json"}}

	uniqueSiteIDs := make(map[string]struct{})
	filteredImps := make([]openrtb2.Imp, 0, len(request.Imp))
	requestCopy := *request

	ixDiag := &IxDiag{}

	for _, imp := range requestCopy.Imp {
		var err error
		ixExt, err := unmarshalToIxExt(&imp)

		if err != nil {
			errs = append(errs, err)
			continue
		}

		if err = parseSiteId(ixExt, uniqueSiteIDs); err != nil {
			errs = append(errs, err)
			continue
		}

		if err := moveSid(&imp, ixExt); err != nil {
			errs = append(errs, err)
		}

		if imp.Banner != nil {
			bannerCopy := *imp.Banner

			if len(bannerCopy.Format) == 0 && bannerCopy.W != nil && bannerCopy.H != nil {
				bannerCopy.Format = []openrtb2.Format{{W: *bannerCopy.W, H: *bannerCopy.H}}
			}

			if len(bannerCopy.Format) == 1 {
				bannerCopy.W = openrtb2.Int64Ptr(bannerCopy.Format[0].W)
				bannerCopy.H = openrtb2.Int64Ptr(bannerCopy.Format[0].H)
			}
			imp.Banner = &bannerCopy
		}
		filteredImps = append(filteredImps, imp)
	}
	requestCopy.Imp = filteredImps

	setSitePublisherId(&requestCopy, uniqueSiteIDs, ixDiag)

	err := setIxDiagIntoExtRequest(&requestCopy, ixDiag)
	if err != nil {
		errs = append(errs, err)
	}

	if len(requestCopy.Imp) != 0 {
		if requestData, err := createRequestData(a, &requestCopy, &headers); err == nil {
			requests = append(requests, requestData)
		} else {
			errs = append(errs, err)
		}
	}

	return requests, errs
}

func setSitePublisherId(requestCopy *openrtb2.BidRequest, uniqueSiteIDs map[string]struct{}, ixDiag *IxDiag) {
	if requestCopy.Site != nil {
		site := *requestCopy.Site
		if site.Publisher == nil {
			site.Publisher = &openrtb2.Publisher{}
		} else {
			publisher := *site.Publisher
			site.Publisher = &publisher
		}

		siteIDs := make([]string, 0, len(uniqueSiteIDs))
		for key := range uniqueSiteIDs {
			siteIDs = append(siteIDs, key)
		}
		if len(siteIDs) == 1 {
			site.Publisher.ID = siteIDs[0]
		}
		if len(siteIDs) > 1 {
			// Sorting siteIDs for predictable output as Go maps don't guarantee order
			sort.Strings(siteIDs)
			multipleSiteIDs := strings.Join(siteIDs, ", ")
			ixDiag.MultipleSiteIds = multipleSiteIDs
		}
		requestCopy.Site = &site
	}
}

func unmarshalToIxExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpIx, error) {
	var bidderExt adapters.ExtImpBidder
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	var ixExt openrtb_ext.ExtImpIx
	if err := json.Unmarshal(bidderExt.Bidder, &ixExt); err != nil {
		return nil, err
	}

	return &ixExt, nil
}

func parseSiteId(ixExt *openrtb_ext.ExtImpIx, uniqueSiteIDs map[string]struct{}) error {
	if ixExt == nil {
		return fmt.Errorf("Nil Ix Ext")
	}
	if ixExt.SiteId != "" {
		uniqueSiteIDs[ixExt.SiteId] = struct{}{}
	}
	return nil
}

func createRequestData(a *IxAdapter, request *openrtb2.BidRequest, headers *http.Header) (*adapters.RequestData, error) {
	body, err := json.Marshal(request)
	return &adapters.RequestData{
		Method:  "POST",
		Uri:     a.URI,
		Body:    body,
		Headers: *headers,
	}, err
}

func (a *IxAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	switch {
	case response.StatusCode == http.StatusNoContent:
		return nil, nil
	case response.StatusCode == http.StatusBadRequest:
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	case response.StatusCode != http.StatusOK:
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResponse); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("JSON parsing error: %v", err),
		}}
	}

	// Store media type per impression in a map for later use to set in bid.ext.prebid.type
	// Won't work for multiple bid case with a multi-format ad unit. We expect to get type from exchange on such case.
	impMediaTypeReq := map[string]openrtb_ext.BidType{}
	for _, imp := range internalRequest.Imp {
		if imp.Banner != nil {
			impMediaTypeReq[imp.ID] = openrtb_ext.BidTypeBanner
		} else if imp.Video != nil {
			impMediaTypeReq[imp.ID] = openrtb_ext.BidTypeVideo
		} else if imp.Native != nil {
			impMediaTypeReq[imp.ID] = openrtb_ext.BidTypeNative
		} else if imp.Audio != nil {
			impMediaTypeReq[imp.ID] = openrtb_ext.BidTypeAudio
		}
	}

	// capacity 0 will make channel unbuffered
	bidderResponse := adapters.NewBidderResponseWithBidsCapacity(0)
	bidderResponse.Currency = bidResponse.Cur

	var errs []error

	for _, seatBid := range bidResponse.SeatBid {
		for i := range seatBid.Bid {
			bid := seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(bid, impMediaTypeReq)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			var bidExtVideo *openrtb_ext.ExtBidPrebidVideo
			var bidExt openrtb_ext.ExtBid
			if bidType == openrtb_ext.BidTypeVideo {
				unmarshalExtErr := json.Unmarshal(bid.Ext, &bidExt)
				if unmarshalExtErr == nil && bidExt.Prebid != nil && bidExt.Prebid.Video != nil {
					bidExtVideo = &openrtb_ext.ExtBidPrebidVideo{
						Duration: bidExt.Prebid.Video.Duration,
					}
					if len(bid.Cat) == 0 {
						bid.Cat = []string{bidExt.Prebid.Video.PrimaryCategory}
					}
				}
			}

			var bidNative1v1 *Native11Wrapper
			if bidType == openrtb_ext.BidTypeNative {
				err := json.Unmarshal([]byte(bid.AdM), &bidNative1v1)
				if err == nil && len(bidNative1v1.Native.EventTrackers) > 0 {
					mergeNativeImpTrackers(&bidNative1v1.Native)
					if json, err := marshalJsonWithoutUnicode(bidNative1v1); err == nil {
						bid.AdM = string(json)
					}
				}
			}

			var bidNative1v2 *native1response.Response
			if bidType == openrtb_ext.BidTypeNative {
				err := json.Unmarshal([]byte(bid.AdM), &bidNative1v2)
				if err == nil && len(bidNative1v2.EventTrackers) > 0 {
					mergeNativeImpTrackers(bidNative1v2)
					if json, err := marshalJsonWithoutUnicode(bidNative1v2); err == nil {
						bid.AdM = string(json)
					}
				}
			}

			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:      &bid,
				BidType:  bidType,
				BidVideo: bidExtVideo,
			})
		}
	}

	return bidderResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid, impMediaTypeReq map[string]openrtb_ext.BidType) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupAudio:
		return openrtb_ext.BidTypeAudio, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	}

	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			prebidType := string(bidExt.Prebid.Type)
			if prebidType != "" {
				return openrtb_ext.ParseBidType(prebidType)
			}
		}
	}

	if bidType, ok := impMediaTypeReq[bid.ImpID]; ok {
		return bidType, nil
	} else {
		return "", fmt.Errorf("unmatched impression id: %s", bid.ImpID)
	}
}

// Builder builds a new instance of the Ix adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &IxAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}

// native 1.2 to 1.1 tracker compatibility handling

type Native11Wrapper struct {
	Native native1response.Response `json:"native,omitempty"`
}

func mergeNativeImpTrackers(bidNative *native1response.Response) {

	// create unique list of imp pixels urls from `imptrackers` and `eventtrackers`
	uniqueImpPixels := map[string]struct{}{}
	for _, v := range bidNative.ImpTrackers {
		uniqueImpPixels[v] = struct{}{}
	}

	for _, v := range bidNative.EventTrackers {
		if v.Event == native1.EventTypeImpression && v.Method == native1.EventTrackingMethodImage {
			uniqueImpPixels[v.URL] = struct{}{}
		}
	}

	// rewrite `imptrackers` with new deduped list of imp pixels
	bidNative.ImpTrackers = make([]string, 0)
	for k := range uniqueImpPixels {
		bidNative.ImpTrackers = append(bidNative.ImpTrackers, k)
	}

	// sort so tests pass correctly
	sort.Strings(bidNative.ImpTrackers)
}

func marshalJsonWithoutUnicode(v interface{}) (string, error) {
	// json.Marshal uses HTMLEscape for strings inside JSON which affects URLs
	// this is a problem with Native responses that embed JSON within JSON
	// a custom encoder can be used to disable this encoding.
	// https://pkg.go.dev/encoding/json#Marshal
	// https://pkg.go.dev/encoding/json#Encoder.SetEscapeHTML
	sb := &strings.Builder{}
	encoder := json.NewEncoder(sb)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return "", err
	}
	// json.Encode also writes a newline, need to remove
	// https://pkg.go.dev/encoding/json#Encoder.Encode
	return strings.TrimSuffix(sb.String(), "\n"), nil
}

func setIxDiagIntoExtRequest(request *openrtb2.BidRequest, ixDiag *IxDiag) error {
	extRequest := &ExtRequest{}
	if request.Ext != nil {
		if err := json.Unmarshal(request.Ext, &extRequest); err != nil {
			return err
		}
	}

	if extRequest.Prebid != nil && extRequest.Prebid.Channel != nil {
		ixDiag.PbjsV = extRequest.Prebid.Channel.Version
	}
	// Slice commit hash out of version
	if strings.Contains(version.Ver, "-") {
		ixDiag.PbsV = version.Ver[:strings.Index(version.Ver, "-")]
	} else if version.Ver != "" {
		ixDiag.PbsV = version.Ver
	}

	// Only set request.ext if ixDiag is not empty
	if *ixDiag != (IxDiag{}) {
		extRequest := &ExtRequest{}
		if request.Ext != nil {
			if err := json.Unmarshal(request.Ext, &extRequest); err != nil {
				return err
			}
		}
		extRequest.IxDiag = ixDiag
		extRequestJson, err := json.Marshal(extRequest)
		if err != nil {
			return err
		}
		request.Ext = extRequestJson
	}
	return nil
}

// moves sid from imp[].ext.bidder.sid to imp[].ext.sid
func moveSid(imp *openrtb2.Imp, ixExt *openrtb_ext.ExtImpIx) error {
	if ixExt == nil {
		return fmt.Errorf("Nil Ix Ext")
	}

	if ixExt.Sid != "" {
		var m map[string]interface{}
		if err := json.Unmarshal(imp.Ext, &m); err != nil {
			return err
		}
		m["sid"] = ixExt.Sid
		ext, err := json.Marshal(m)
		if err != nil {
			return err
		}
		imp.Ext = ext
	}
	return nil
}
