package ix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/prebid/prebid-server/v3/version"

	"github.com/prebid/openrtb/v20/native1"
	native1response "github.com/prebid/openrtb/v20/native1/response"
	"github.com/prebid/openrtb/v20/openrtb2"
)

type IxAdapter struct {
	URI string
}

type ExtRequest struct {
	Prebid *openrtb_ext.ExtRequestPrebid `json:"prebid,omitempty"`
	SChain *openrtb2.SupplyChain         `json:"schain,omitempty"`
	IxDiag json.RawMessage               `json:"ixdiag,omitempty"`
}

type auctionConfig struct {
	BidId  string          `json:"bidId,omitempty"`
	Config json.RawMessage `json:"config,omitempty"`
}

type ixRespExt struct {
	AuctionConfig []auctionConfig `json:"protectedAudienceAuctionConfigs,omitempty"`
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

	ixDiagFields := make(map[string]interface{})

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
				bannerCopy.W = ptrutil.ToPtr(bannerCopy.Format[0].W)
				bannerCopy.H = ptrutil.ToPtr(bannerCopy.Format[0].H)
			}
			imp.Banner = &bannerCopy
		}
		filteredImps = append(filteredImps, imp)
	}
	requestCopy.Imp = filteredImps

	setPublisherId(&requestCopy, uniqueSiteIDs, ixDiagFields)

	err := setIxDiagIntoExtRequest(&requestCopy, ixDiagFields, version.Ver)
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

func setPublisherId(requestCopy *openrtb2.BidRequest, uniqueSiteIDs map[string]struct{}, ixDiagFields map[string]interface{}) {
	siteIDs := make([]string, 0, len(uniqueSiteIDs))
	for key := range uniqueSiteIDs {
		siteIDs = append(siteIDs, key)
	}
	if requestCopy.Site != nil {
		site := *requestCopy.Site
		if site.Publisher == nil {
			site.Publisher = &openrtb2.Publisher{}
		} else {
			publisher := *site.Publisher
			site.Publisher = &publisher
		}
		if len(siteIDs) == 1 {
			site.Publisher.ID = siteIDs[0]
		}
		requestCopy.Site = &site
	}

	if requestCopy.App != nil {
		app := *requestCopy.App

		if app.Publisher == nil {
			app.Publisher = &openrtb2.Publisher{}
		} else {
			publisher := *app.Publisher
			app.Publisher = &publisher
		}
		if len(siteIDs) == 1 {
			app.Publisher.ID = siteIDs[0]
		}
		requestCopy.App = &app
	}

	if len(siteIDs) > 1 {
		// Sorting siteIDs for predictable output as Go maps don't guarantee order
		sort.Strings(siteIDs)
		multipleSiteIDs := strings.Join(siteIDs, ", ")
		ixDiagFields["multipleSiteIds"] = multipleSiteIDs
	}
}

func unmarshalToIxExt(imp *openrtb2.Imp) (*openrtb_ext.ExtImpIx, error) {
	var bidderExt adapters.ExtImpBidder
	if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, err
	}

	var ixExt openrtb_ext.ExtImpIx
	if err := jsonutil.Unmarshal(bidderExt.Bidder, &ixExt); err != nil {
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
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
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
	if err := jsonutil.Unmarshal(response.Body, &bidResponse); err != nil {
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
				unmarshalExtErr := jsonutil.Unmarshal(bid.Ext, &bidExt)
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
				err := jsonutil.Unmarshal([]byte(bid.AdM), &bidNative1v1)
				if err == nil && len(bidNative1v1.Native.EventTrackers) > 0 {
					mergeNativeImpTrackers(&bidNative1v1.Native)
					if json, err := marshalJsonWithoutUnicode(bidNative1v1); err == nil {
						bid.AdM = string(json)
					}
				}
			}

			var bidNative1v2 *native1response.Response
			if bidType == openrtb_ext.BidTypeNative {
				err := jsonutil.Unmarshal([]byte(bid.AdM), &bidNative1v2)
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

	if bidResponse.Ext != nil {
		var bidRespExt ixRespExt
		if err := jsonutil.Unmarshal(bidResponse.Ext, &bidRespExt); err != nil {
			return nil, append(errs, err)
		}

		if bidRespExt.AuctionConfig != nil {
			bidderResponse.FledgeAuctionConfigs = make([]*openrtb_ext.FledgeAuctionConfig, 0, len(bidRespExt.AuctionConfig))
			for _, config := range bidRespExt.AuctionConfig {
				if config.Config != nil {
					fledgeAuctionConfig := &openrtb_ext.FledgeAuctionConfig{
						ImpId:  config.BidId,
						Config: config.Config,
					}
					bidderResponse.FledgeAuctionConfigs = append(bidderResponse.FledgeAuctionConfigs, fledgeAuctionConfig)
				}
			}
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
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
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

// extractVersionWithoutCommitHash takes a version string like '0.23.1-g4ee257d8' and returns
// the prefix without the commit hash: '0.23.1' -
// the substring preceding the first hyphen.
func extractVersionWithoutCommitHash(ver string) string {
	if strings.Contains(ver, "-") {
		return ver[:strings.Index(ver, "-")]
	}
	return ver // if no hyphen, return the original string
}

func setIxDiagIntoExtRequest(request *openrtb2.BidRequest, ixDiagAdditionalFields map[string]interface{}, ver string) error {
	var extRequest ExtRequest
	if request.Ext != nil {
		if err := jsonutil.Unmarshal(request.Ext, &extRequest); err != nil {
			return err
		}
	}

	if extRequest.Prebid != nil && extRequest.Prebid.Channel != nil {
		ixDiagAdditionalFields["pbjsv"] = extRequest.Prebid.Channel.Version
	}
	// Slice commit hash out of version
	prebidServerVersion := "unknown" // Default value when the version cannot be determined
	if ver != "" {
		prebidServerVersion = extractVersionWithoutCommitHash(ver)
	}
	ixDiagAdditionalFields["pbsv"] = prebidServerVersion
	ixDiagAdditionalFields["pbsp"] = "go" // indicate prebid server implementation use Go version

	var ixDiagMap map[string]interface{}
	if extRequest.IxDiag != nil && len(extRequest.IxDiag) > 0 {
		if err := jsonutil.Unmarshal(extRequest.IxDiag, &ixDiagMap); err != nil {
			return err
		}
	} else {
		ixDiagMap = make(map[string]interface{})
	}

	for k, v := range ixDiagAdditionalFields {
		ixDiagMap[k] = v
	}

	ixDiagJSON, err := json.Marshal(ixDiagMap)
	if err != nil {
		return err
	}

	extRequest.IxDiag = ixDiagJSON

	extRequestJSON, err := json.Marshal(extRequest)
	if err != nil {
		return err
	}

	request.Ext = extRequestJSON
	return nil
}

// moves sid from imp[].ext.bidder.sid to imp[].ext.sid
func moveSid(imp *openrtb2.Imp, ixExt *openrtb_ext.ExtImpIx) error {
	if ixExt == nil {
		return fmt.Errorf("Nil Ix Ext")
	}

	if ixExt.Sid != "" {
		var m map[string]interface{}
		if err := jsonutil.Unmarshal(imp.Ext, &m); err != nil {
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
