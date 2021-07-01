package pubmatic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"golang.org/x/net/context/ctxhttp"
)

const MAX_IMPRESSIONS_PUBMATIC = 30

type PubmaticAdapter struct {
	http *adapters.HTTPAdapter
	URI  string
}

// used for cookies and such
func (a *PubmaticAdapter) Name() string {
	return "pubmatic"
}

func (a *PubmaticAdapter) SkipNoCookies() bool {
	return false
}

// Below is bidder specific parameters for pubmatic adaptor,
// PublisherId and adSlot are mandatory parameters, others are optional parameters
// Keywords is bid specific parameter,
// WrapExt needs to be sent once per bid request
type pubmaticParams struct {
	PublisherId string            `json:"publisherId"`
	AdSlot      string            `json:"adSlot"`
	WrapExt     json.RawMessage   `json:"wrapper,omitempty"`
	Keywords    map[string]string `json:"keywords,omitempty"`
}

type pubmaticBidExtVideo struct {
	Duration *int `json:"duration,omitempty"`
}

type pubmaticBidExt struct {
	BidType           *int                 `json:"BidType,omitempty"`
	VideoCreativeInfo *pubmaticBidExtVideo `json:"video,omitempty"`
}

type ExtImpBidderPubmatic struct {
	adapters.ExtImpBidder
	Data *ExtData `json:"data,omitempty"`
}

type ExtData struct {
	AdServer *ExtAdServer `json:"adserver"`
	PBAdSlot string       `json:"pbadslot"`
}

type ExtAdServer struct {
	Name   string `json:"name"`
	AdSlot string `json:"adslot"`
}

const (
	INVALID_PARAMS    = "Invalid BidParam"
	MISSING_PUBID     = "Missing PubID"
	MISSING_ADSLOT    = "Missing AdSlot"
	INVALID_WRAPEXT   = "Invalid WrapperExt"
	INVALID_ADSIZE    = "Invalid AdSize"
	INVALID_WIDTH     = "Invalid Width"
	INVALID_HEIGHT    = "Invalid Height"
	INVALID_MEDIATYPE = "Invalid MediaType"
	INVALID_ADSLOT    = "Invalid AdSlot"

	dctrKeyName        = "key_val"
	pmZoneIDKeyName    = "pmZoneId"
	pmZoneIDKeyNameOld = "pmZoneID"
	ImpExtAdUnitKey    = "dfp_ad_unit_code"
	AdServerGAM        = "gam"
)

func PrepareLogMessage(tID, pubId, adUnitId, bidID, details string, args ...interface{}) string {
	return fmt.Sprintf("[PUBMATIC] ReqID [%s] PubID [%s] AdUnit [%s] BidID [%s] %s \n",
		tID, pubId, adUnitId, bidID, details)
}

func (a *PubmaticAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	mediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}
	pbReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), mediaTypes)

	if err != nil {
		logf("[PUBMATIC] Failed to make ortb request for request id [%s] \n", pbReq.ID)
		return nil, err
	}

	var errState []string
	adSlotFlag := false
	pubId := ""
	wrapExt := ""
	if len(bidder.AdUnits) > MAX_IMPRESSIONS_PUBMATIC {
		logf("[PUBMATIC] First %d impressions will be considered from request tid %s\n",
			MAX_IMPRESSIONS_PUBMATIC, pbReq.ID)
	}

	for i, unit := range bidder.AdUnits {
		var params pubmaticParams
		err := json.Unmarshal(unit.Params, &params)
		if err != nil {
			errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_PARAMS, unit.Params))
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: invalid JSON  [%s] err [%s]", unit.Params, err.Error())))
			continue
		}

		if params.PublisherId == "" {
			errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, MISSING_PUBID, unit.Params))
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: Publisher Id missing")))
			continue
		}
		pubId = params.PublisherId

		if params.AdSlot == "" {
			errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, MISSING_ADSLOT, unit.Params))
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: adSlot missing")))
			continue
		}

		// Parse Wrapper Extension i.e. ProfileID and VersionID only once per request
		if wrapExt == "" && len(params.WrapExt) != 0 {
			var wrapExtMap map[string]int
			err := json.Unmarshal([]byte(params.WrapExt), &wrapExtMap)
			if err != nil {
				errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_WRAPEXT, unit.Params))
				logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
					fmt.Sprintf("Ignored bid: Wrapper Extension Invalid")))
				continue
			}
			wrapExt = string(params.WrapExt)
		}

		adSlotStr := strings.TrimSpace(params.AdSlot)
		adSlot := strings.Split(adSlotStr, "@")
		if len(adSlot) == 2 && adSlot[0] != "" && adSlot[1] != "" {
			// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
			if len(pbReq.Imp) <= i {
				break
			}
			if pbReq.Imp[i].Banner != nil {
				adSize := strings.Split(strings.ToLower(strings.TrimSpace(adSlot[1])), "x")
				if len(adSize) == 2 {
					width, err := strconv.Atoi(strings.TrimSpace(adSize[0]))
					if err != nil {
						errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_WIDTH, unit.Params))
						logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
							fmt.Sprintf("Ignored bid: invalid adSlot width [%s]", adSize[0])))
						continue
					}

					heightStr := strings.Split(strings.TrimSpace(adSize[1]), ":")
					height, err := strconv.Atoi(strings.TrimSpace(heightStr[0]))
					if err != nil {
						errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_HEIGHT, unit.Params))
						logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
							fmt.Sprintf("Ignored bid: invalid adSlot height [%s]", heightStr[0])))
						continue
					}

					pbReq.Imp[i].TagID = strings.TrimSpace(adSlot[0])
					pbReq.Imp[i].Banner.W = openrtb2.Int64Ptr(int64(width))
					pbReq.Imp[i].Banner.H = openrtb2.Int64Ptr(int64(height))

					if len(params.Keywords) != 0 {
						kvstr := prepareImpressionExt(params.Keywords)
						pbReq.Imp[i].Ext = json.RawMessage([]byte(kvstr))
					} else {
						pbReq.Imp[i].Ext = nil
					}

					adSlotFlag = true
				} else {
					errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_ADSIZE, unit.Params))
					logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
						fmt.Sprintf("Ignored bid: invalid adSize [%s]", adSize)))
					continue
				}
			} else {
				errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_MEDIATYPE, unit.Params))
				logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
					fmt.Sprintf("Ignored bid: invalid Media Type")))
				continue
			}
		} else {
			errState = append(errState, fmt.Sprintf("BidID:%s;Error:%s;param:%s", unit.BidID, INVALID_ADSLOT, unit.Params))
			logf(PrepareLogMessage(pbReq.ID, params.PublisherId, unit.Code, unit.BidID,
				fmt.Sprintf("Ignored bid: invalid adSlot [%s]", params.AdSlot)))
			continue
		}

		if pbReq.Site != nil {
			siteCopy := *pbReq.Site
			siteCopy.Publisher = &openrtb2.Publisher{ID: params.PublisherId, Domain: req.Domain}
			pbReq.Site = &siteCopy
		}
		if pbReq.App != nil {
			appCopy := *pbReq.App
			appCopy.Publisher = &openrtb2.Publisher{ID: params.PublisherId, Domain: req.Domain}
			pbReq.App = &appCopy
		}
	}

	if !(adSlotFlag) {
		return nil, &errortypes.BadInput{
			Message: "Incorrect adSlot / Publisher params, Error list: [" + strings.Join(errState, ",") + "]",
		}
	}

	if wrapExt != "" {
		rawExt := fmt.Sprintf("{\"wrapper\": %s}", wrapExt)
		pbReq.Ext = json.RawMessage(rawExt)
	}

	reqJSON, err := json.Marshal(pbReq)

	debug := &pbs.BidderDebug{
		RequestURI: a.URI,
	}

	if req.IsDebug {
		debug.RequestBody = string(reqJSON)
		bidder.Debug = append(bidder.Debug, debug)
	}

	userId, _, _ := req.Cookie.GetUID(a.Name())
	httpReq, err := http.NewRequest("POST", a.URI, bytes.NewBuffer(reqJSON))
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")
	httpReq.AddCookie(&http.Cookie{
		Name:  "KADUSERCOOKIE",
		Value: userId,
	})

	pbResp, err := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if err != nil {
		return nil, err
	}

	debug.StatusCode = pbResp.StatusCode

	if pbResp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if pbResp.StatusCode == http.StatusBadRequest {
		return nil, &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status: %d", pbResp.StatusCode),
		}
	}

	if pbResp.StatusCode != http.StatusOK {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status: %d", pbResp.StatusCode),
		}
	}

	defer pbResp.Body.Close()
	body, err := ioutil.ReadAll(pbResp.Body)
	if err != nil {
		return nil, err
	}

	if req.IsDebug {
		debug.ResponseBody = string(body)
	}

	var bidResp openrtb2.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		return nil, &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status: %d", pbResp.StatusCode),
		}
	}

	bids := make(pbs.PBSBidSlice, 0)

	numBids := 0
	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			numBids++

			bidID := bidder.LookupBidID(bid.ImpID)
			if bidID == "" {
				return nil, &errortypes.BadServerResponse{
					Message: fmt.Sprintf("Unknown ad unit code '%s'", bid.ImpID),
				}
			}

			pbid := pbs.PBSBid{
				BidID:       bidID,
				AdUnitCode:  bid.ImpID,
				BidderCode:  bidder.BidderCode,
				Price:       bid.Price,
				Adm:         bid.AdM,
				Creative_id: bid.CrID,
				Width:       bid.W,
				Height:      bid.H,
				DealId:      bid.DealID,
			}

			var bidExt pubmaticBidExt
			mediaType := openrtb_ext.BidTypeBanner
			if err := json.Unmarshal(bid.Ext, &bidExt); err == nil {
				mediaType = getBidType(&bidExt)
			}
			pbid.CreativeMediaType = string(mediaType)

			bids = append(bids, &pbid)
			logf("[PUBMATIC] Returned Bid for PubID [%s] AdUnit [%s] BidID [%s] Size [%dx%d] Price [%f] \n",
				pubId, pbid.AdUnitCode, pbid.BidID, pbid.Width, pbid.Height, pbid.Price)
		}
	}

	return bids, nil
}

func (a *PubmaticAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	errs := make([]error, 0, len(request.Imp))

	wrapExt := ""
	pubID := ""

	for i := 0; i < len(request.Imp); i++ {
		err := parseImpressionObject(&request.Imp[i], &wrapExt, &pubID)
		// If the parsing is failed, remove imp and add the error.
		if err != nil {
			errs = append(errs, err)
			request.Imp = append(request.Imp[:i], request.Imp[i+1:]...)
			i--
		}
	}

	// If all the requests are invalid, Call to adaptor is skipped
	if len(request.Imp) == 0 {
		return nil, errs
	}

	if wrapExt != "" {
		rawExt := fmt.Sprintf("{\"wrapper\": %s}", wrapExt)
		request.Ext = json.RawMessage(rawExt)
	}

	if request.Site != nil {
		siteCopy := *request.Site
		if siteCopy.Publisher != nil {
			publisherCopy := *siteCopy.Publisher
			publisherCopy.ID = pubID
			siteCopy.Publisher = &publisherCopy
		} else {
			siteCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.Site = &siteCopy
	} else if request.App != nil {
		appCopy := *request.App
		if appCopy.Publisher != nil {
			publisherCopy := *appCopy.Publisher
			publisherCopy.ID = pubID
			appCopy.Publisher = &publisherCopy
		} else {
			appCopy.Publisher = &openrtb2.Publisher{ID: pubID}
		}
		request.App = &appCopy
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		errs = append(errs, err)
		return nil, errs
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	return []*adapters.RequestData{{
		Method:  "POST",
		Uri:     a.URI,
		Body:    reqJSON,
		Headers: headers,
	}}, errs
}

// validateAdslot validate the optional adslot string
// valid formats are 'adslot@WxH', 'adslot' and no adslot
func validateAdSlot(adslot string, imp *openrtb2.Imp) error {
	adSlotStr := strings.TrimSpace(adslot)

	if len(adSlotStr) == 0 {
		return nil
	}

	if !strings.Contains(adSlotStr, "@") {
		imp.TagID = adSlotStr
		return nil
	}

	adSlot := strings.Split(adSlotStr, "@")
	if len(adSlot) == 2 && adSlot[0] != "" && adSlot[1] != "" {
		imp.TagID = strings.TrimSpace(adSlot[0])

		adSize := strings.Split(strings.ToLower(adSlot[1]), "x")
		if len(adSize) != 2 {
			return fmt.Errorf("Invalid size provided in adSlot %v", adSlotStr)
		}

		width, err := strconv.Atoi(strings.TrimSpace(adSize[0]))
		if err != nil {
			return fmt.Errorf("Invalid width provided in adSlot %v", adSlotStr)
		}

		heightStr := strings.Split(adSize[1], ":")
		height, err := strconv.Atoi(strings.TrimSpace(heightStr[0]))
		if err != nil {
			return fmt.Errorf("Invalid height provided in adSlot %v", adSlotStr)
		}

		//In case of video, size could be derived from the player size
		if imp.Banner != nil {
			imp.Banner = assignBannerWidthAndHeight(imp.Banner, int64(width), int64(height))
		}
	} else {
		return fmt.Errorf("Invalid adSlot %v", adSlotStr)
	}

	return nil
}

func assignBannerSize(banner *openrtb2.Banner) (*openrtb2.Banner, error) {
	if banner.W != nil && banner.H != nil {
		return banner, nil
	}

	return assignBannerWidthAndHeight(banner, banner.Format[0].W, banner.Format[0].H), nil
}

func assignBannerWidthAndHeight(banner *openrtb2.Banner, w, h int64) *openrtb2.Banner {
	bannerCopy := *banner
	bannerCopy.W = openrtb2.Int64Ptr(w)
	bannerCopy.H = openrtb2.Int64Ptr(h)
	return &bannerCopy
}

// parseImpressionObject parse the imp to get it ready to send to pubmatic
func parseImpressionObject(imp *openrtb2.Imp, wrapExt *string, pubID *string) error {
	// PubMatic supports banner and video impressions.
	if imp.Banner == nil && imp.Video == nil {
		return fmt.Errorf("Invalid MediaType. PubMatic only supports Banner and Video. Ignoring ImpID=%s", imp.ID)
	}

	if imp.Audio != nil {
		imp.Audio = nil
	}

	var bidderExt ExtImpBidderPubmatic
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return err
	}

	var pubmaticExt openrtb_ext.ExtImpPubmatic
	if err := json.Unmarshal(bidderExt.Bidder, &pubmaticExt); err != nil {
		return err
	}

	if *pubID == "" {
		*pubID = strings.TrimSpace(pubmaticExt.PublisherId)
	}

	// Parse Wrapper Extension only once per request
	if *wrapExt == "" && len(pubmaticExt.WrapExt) != 0 {
		var wrapExtMap map[string]int
		err := json.Unmarshal([]byte(pubmaticExt.WrapExt), &wrapExtMap)
		if err != nil {
			return fmt.Errorf("Error in Wrapper Parameters = %v  for ImpID = %v WrapperExt = %v", err.Error(), imp.ID, string(pubmaticExt.WrapExt))
		}
		*wrapExt = string(pubmaticExt.WrapExt)
	}

	if err := validateAdSlot(strings.TrimSpace(pubmaticExt.AdSlot), imp); err != nil {
		return err
	}

	if imp.Banner != nil {
		bannerCopy, err := assignBannerSize(imp.Banner)
		if err != nil {
			return err
		}
		imp.Banner = bannerCopy
	}

	extMap := make(map[string]interface{}, 0)
	if pubmaticExt.Keywords != nil && len(pubmaticExt.Keywords) != 0 {
		addKeywordsToExt(pubmaticExt.Keywords, extMap)
	}
	//Give preference to direct values of 'dctr' & 'pmZoneId' params in extension
	if pubmaticExt.Dctr != "" {
		extMap[dctrKeyName] = pubmaticExt.Dctr
	}
	if pubmaticExt.PmZoneID != "" {
		extMap[pmZoneIDKeyName] = pubmaticExt.PmZoneID
	}

	if bidderExt.Data != nil {
		if bidderExt.Data.AdServer != nil && bidderExt.Data.AdServer.Name == AdServerGAM && bidderExt.Data.AdServer.AdSlot != "" {
			extMap[ImpExtAdUnitKey] = bidderExt.Data.AdServer.AdSlot
		} else if bidderExt.Data.PBAdSlot != "" {
			extMap[ImpExtAdUnitKey] = bidderExt.Data.PBAdSlot
		}
	}

	imp.Ext = nil
	if len(extMap) > 0 {
		ext, err := json.Marshal(extMap)
		if err == nil {
			imp.Ext = ext
		}
	}

	return nil

}

func addKeywordsToExt(keywords []*openrtb_ext.ExtImpPubmaticKeyVal, extMap map[string]interface{}) {
	for _, keyVal := range keywords {
		if len(keyVal.Values) == 0 {
			logf("No values present for key = %s", keyVal.Key)
			continue
		} else {
			key := keyVal.Key
			if keyVal.Key == pmZoneIDKeyNameOld {
				key = pmZoneIDKeyName
			}
			extMap[key] = strings.Join(keyVal.Values[:], ",")
		}
	}
}

func prepareImpressionExt(keywords map[string]string) string {

	eachKv := make([]string, 0, len(keywords))
	for key, val := range keywords {
		if len(val) == 0 {
			logf("No values present for key = %s", key)
			continue
		} else {
			eachKv = append(eachKv, fmt.Sprintf("\"%s\":\"%s\"", key, val))
		}
	}

	kvStr := "{" + strings.Join(eachKv, ",") + "}"
	return kvStr
}

func (a *PubmaticAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if response.StatusCode == http.StatusBadRequest {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode),
		}}
	}

	if response.StatusCode != http.StatusOK {
		return nil, []error{fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", response.StatusCode)}
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	var errs []error
	for _, sb := range bidResp.SeatBid {
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]
			impVideo := &openrtb_ext.ExtBidPrebidVideo{}

			if len(bid.Cat) > 1 {
				bid.Cat = bid.Cat[0:1]
			}

			var bidExt *pubmaticBidExt
			bidType := openrtb_ext.BidTypeBanner
			if err := json.Unmarshal(bid.Ext, &bidExt); err == nil && bidExt != nil {
				if bidExt.VideoCreativeInfo != nil && bidExt.VideoCreativeInfo.Duration != nil {
					impVideo.Duration = *bidExt.VideoCreativeInfo.Duration
				}
				bidType = getBidType(bidExt)
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:      &bid,
				BidType:  bidType,
				BidVideo: impVideo,
			})

		}
	}
	return bidResponse, errs
}

// getBidType returns the bid type specified in the response bid.ext
func getBidType(bidExt *pubmaticBidExt) openrtb_ext.BidType {
	// setting "banner" as the default bid type
	bidType := openrtb_ext.BidTypeBanner
	if bidExt != nil && bidExt.BidType != nil {
		switch *bidExt.BidType {
		case 0:
			bidType = openrtb_ext.BidTypeBanner
		case 1:
			bidType = openrtb_ext.BidTypeVideo
		case 2:
			bidType = openrtb_ext.BidTypeNative
		default:
			// default value is banner
			bidType = openrtb_ext.BidTypeBanner
		}
	}
	return bidType
}

func logf(msg string, args ...interface{}) {
	if glog.V(2) {
		glog.Infof(msg, args...)
	}
}

func NewPubmaticLegacyAdapter(config *adapters.HTTPAdapterConfig, uri string) *PubmaticAdapter {
	a := adapters.NewHTTPAdapter(config)

	return &PubmaticAdapter{
		http: a,
		URI:  uri,
	}
}

// Builder builds a new instance of the Pubmatic adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &PubmaticAdapter{
		URI: config.Endpoint,
	}
	return bidder, nil
}
