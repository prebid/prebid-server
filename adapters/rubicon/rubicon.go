package rubicon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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

const badvLimitSize = 50

type RubiconAdapter struct {
	http         *adapters.HTTPAdapter
	URI          string
	XAPIUsername string
	XAPIPassword string
}

// used for cookies and such
func (a *RubiconAdapter) Name() string {
	return "rubicon"
}

func (a *RubiconAdapter) SkipNoCookies() bool {
	return false
}

type rubiconParams struct {
	AccountId int                `json:"accountId"`
	SiteId    int                `json:"siteId"`
	ZoneId    int                `json:"zoneId"`
	Inventory json.RawMessage    `json:"inventory,omitempty"`
	Visitor   json.RawMessage    `json:"visitor,omitempty"`
	Video     rubiconVideoParams `json:"video"`
}

type bidRequestExt struct {
	Prebid bidRequestExtPrebid `json:"prebid"`
}

type bidRequestExtPrebid struct {
	Bidders bidRequestExtPrebidBidders `json:"bidders"`
}

type bidRequestExtPrebidBidders struct {
	Rubicon prebidBiddersRubicon `json:"rubicon,omitempty"`
}

type prebidBiddersRubicon struct {
	Debug prebidBiddersRubiconDebug `json:"debug,omitempty"`
}

type prebidBiddersRubiconDebug struct {
	CpmOverride float64 `json:"cpmoverride,omitempty"`
}

type rubiconImpExtRPTrack struct {
	Mint        string `json:"mint"`
	MintVersion string `json:"mint_version"`
}

type rubiconImpExtRP struct {
	ZoneID int                  `json:"zone_id"`
	Target json.RawMessage      `json:"target,omitempty"`
	Track  rubiconImpExtRPTrack `json:"track"`
}

type rubiconImpExt struct {
	RP rubiconImpExtRP `json:"rp"`
}

type rubiconUserExtRP struct {
	Target json.RawMessage `json:"target,omitempty"`
}

type rubiconExtUserTpID struct {
	Source string `json:"source"`
	UID    string `json:"uid"`
}

type rubiconDataExt struct {
	SegTax int `json:"segtax"`
}

type rubiconUserExt struct {
	Consent     string                   `json:"consent,omitempty"`
	Eids        []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
	TpID        []rubiconExtUserTpID     `json:"tpid,omitempty"`
	RP          rubiconUserExtRP         `json:"rp"`
	LiverampIdl string                   `json:"liveramp_idl,omitempty"`
}

type rubiconSiteExtRP struct {
	SiteID int             `json:"site_id"`
	Target json.RawMessage `json:"target,omitempty"`
}

type rubiconSiteExt struct {
	RP rubiconSiteExtRP `json:"rp"`
}

type rubiconPubExtRP struct {
	AccountID int `json:"account_id"`
}

type rubiconPubExt struct {
	RP rubiconPubExtRP `json:"rp"`
}

type rubiconBannerExtRP struct {
	SizeID     int    `json:"size_id,omitempty"`
	AltSizeIDs []int  `json:"alt_size_ids,omitempty"`
	MIME       string `json:"mime"`
}

type rubiconBannerExt struct {
	RP rubiconBannerExtRP `json:"rp"`
}

type ExtImpContextData struct {
	AdSlot string `json:"adslot,omitempty"`
}

type ExtImpContext struct {
	Data ExtImpContextData `json:"data,omitempty"`
}

type ExtImpWithContext struct {
	Context ExtImpContext `json:"context,omitempty"` // First Party Data context
}

// ***** Video Extension *****
type rubiconVideoParams struct {
	Language     string `json:"language,omitempty"`
	PlayerHeight int    `json:"playerHeight,omitempty"`
	PlayerWidth  int    `json:"playerWidth,omitempty"`
	VideoSizeID  int    `json:"size_id,omitempty"`
	Skip         int    `json:"skip,omitempty"`
	SkipDelay    int    `json:"skipdelay,omitempty"`
}

type rubiconVideoExt struct {
	Skip      int               `json:"skip,omitempty"`
	SkipDelay int               `json:"skipdelay,omitempty"`
	VideoType string            `json:"videotype,omitempty"`
	RP        rubiconVideoExtRP `json:"rp"`
}

type rubiconVideoExtRP struct {
	SizeID int `json:"size_id,omitempty"`
}

type rubiconTargetingExt struct {
	RP rubiconTargetingExtRP `json:"rp"`
}

type rubiconTargetingExtRP struct {
	Targeting []rubiconTargetingObj `json:"targeting"`
}

type rubiconTargetingObj struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type rubiconDeviceExtRP struct {
	PixelRatio float64 `json:"pixelratio"`
}

type rubiconDeviceExt struct {
	RP rubiconDeviceExtRP `json:"rp"`
}

type rubiconUser struct {
	Language string `json:"language"`
}

type rubiconBidResponse struct {
	openrtb2.BidResponse
	SeatBid []rubiconSeatBid `json:"seatbid,omitempty"`
}

type rubiconSeatBid struct {
	openrtb2.SeatBid
	Buyer string `json:"buyer,omitempty"`
}

type extPrebid struct {
	Prebid *openrtb_ext.ExtBidPrebid `json:"prebid,omitempty"`
	Bidder json.RawMessage           `json:"bidder,omitempty"`
}

type rubiSize struct {
	w uint16
	h uint16
}

var rubiSizeMap = map[rubiSize]int{
	{w: 468, h: 60}:    1,
	{w: 728, h: 90}:    2,
	{w: 728, h: 91}:    2,
	{w: 120, h: 90}:    5,
	{w: 125, h: 125}:   7,
	{w: 120, h: 600}:   8,
	{w: 160, h: 600}:   9,
	{w: 300, h: 600}:   10,
	{w: 200, h: 200}:   13,
	{w: 250, h: 250}:   14,
	{w: 300, h: 250}:   15,
	{w: 300, h: 251}:   15,
	{w: 336, h: 280}:   16,
	{w: 240, h: 400}:   17,
	{w: 300, h: 100}:   19,
	{w: 980, h: 120}:   31,
	{w: 250, h: 360}:   32,
	{w: 180, h: 500}:   33,
	{w: 980, h: 150}:   35,
	{w: 468, h: 400}:   37,
	{w: 930, h: 180}:   38,
	{w: 750, h: 100}:   39,
	{w: 750, h: 200}:   40,
	{w: 750, h: 300}:   41,
	{w: 320, h: 50}:    43,
	{w: 300, h: 50}:    44,
	{w: 300, h: 300}:   48,
	{w: 1024, h: 768}:  53,
	{w: 300, h: 1050}:  54,
	{w: 970, h: 90}:    55,
	{w: 970, h: 250}:   57,
	{w: 1000, h: 90}:   58,
	{w: 320, h: 80}:    59,
	{w: 320, h: 150}:   60,
	{w: 1000, h: 1000}: 61,
	{w: 580, h: 500}:   64,
	{w: 640, h: 480}:   65,
	{w: 930, h: 600}:   66,
	{w: 320, h: 480}:   67,
	{w: 1800, h: 1000}: 68,
	{w: 320, h: 320}:   72,
	{w: 320, h: 160}:   73,
	{w: 980, h: 240}:   78,
	{w: 980, h: 300}:   79,
	{w: 980, h: 400}:   80,
	{w: 480, h: 300}:   83,
	{w: 300, h: 120}:   85,
	{w: 548, h: 150}:   90,
	{w: 970, h: 310}:   94,
	{w: 970, h: 100}:   95,
	{w: 970, h: 210}:   96,
	{w: 480, h: 320}:   101,
	{w: 768, h: 1024}:  102,
	{w: 480, h: 280}:   103,
	{w: 250, h: 800}:   105,
	{w: 320, h: 240}:   108,
	{w: 1000, h: 300}:  113,
	{w: 320, h: 100}:   117,
	{w: 800, h: 250}:   125,
	{w: 200, h: 600}:   126,
	{w: 980, h: 600}:   144,
	{w: 980, h: 150}:   145,
	{w: 1000, h: 250}:  152,
	{w: 640, h: 320}:   156,
	{w: 320, h: 250}:   159,
	{w: 250, h: 600}:   179,
	{w: 600, h: 300}:   195,
	{w: 640, h: 360}:   198,
	{w: 640, h: 200}:   199,
	{w: 1030, h: 590}:  213,
	{w: 980, h: 360}:   214,
	{w: 320, h: 180}:   229,
	{w: 2000, h: 1400}: 230,
	{w: 580, h: 400}:   232,
	{w: 480, h: 820}:   256,
	{w: 400, h: 600}:   257,
	{w: 500, h: 200}:   258,
	{w: 998, h: 200}:   259,
	{w: 970, h: 1000}:  264,
	{w: 1920, h: 1080}: 265,
	{w: 1800, h: 200}:  274,
	{w: 320, h: 500}:   278,
	{w: 320, h: 400}:   282,
	{w: 640, h: 380}:   288,
	{w: 500, h: 1000}:  548,
}

// defines the contract for bidrequest.user.ext.eids[i].ext
type rubiconUserExtEidExt struct {
	Segments []string `json:"segments,omitempty"`
}

// defines the contract for bidrequest.user.ext.eids[i].uids[j].ext
type rubiconUserExtEidUidExt struct {
	RtiPartner string `json:"rtiPartner,omitempty"`
}

type mappedRubiconUidsParam struct {
	tpIds       []rubiconExtUserTpID
	segments    []string
	liverampIdl string
}

//MAS algorithm
func findPrimary(alt []int) (int, []int) {
	min, pos, primary := 0, 0, 0
	for i, size := range alt {
		if size == 15 {
			primary = 15
			pos = i
			break
		} else if size == 2 {
			primary = 2
			pos = i
		} else if size == 9 && primary != 2 {
			primary = 9
			pos = i
		} else if size < alt[min] {
			min = i
		}
	}
	if primary == 0 {
		primary = alt[min]
		pos = min
	}

	alt = append(alt[:pos], alt[pos+1:]...)
	return primary, alt
}

func parseRubiconSizes(sizes []openrtb2.Format) (primary int, alt []int, err error) {
	// Fixes #317
	if len(sizes) < 1 {
		err = &errortypes.BadInput{
			Message: "rubicon imps must have at least one imp.format element",
		}
		return
	}
	for _, size := range sizes {
		if rs, ok := rubiSizeMap[rubiSize{w: uint16(size.W), h: uint16(size.H)}]; ok {
			alt = append(alt, rs)
		}
	}
	if len(alt) > 0 {
		primary, alt = findPrimary(alt)
	} else {
		err = &errortypes.BadInput{
			Message: "No primary size found",
		}
	}
	return
}

func (a *RubiconAdapter) callOne(ctx context.Context, reqJSON bytes.Buffer) (result adapters.CallOneResult, err error) {
	httpReq, err := http.NewRequest("POST", a.URI, &reqJSON)
	httpReq.Header.Add("Content-Type", "application/json;charset=utf-8")
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", "prebid-server/1.0")
	httpReq.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)

	rubiResp, e := ctxhttp.Do(ctx, a.http.Client, httpReq)
	if e != nil {
		err = e
		return
	}

	defer rubiResp.Body.Close()
	body, _ := ioutil.ReadAll(rubiResp.Body)
	result.ResponseBody = string(body)

	result.StatusCode = rubiResp.StatusCode

	if rubiResp.StatusCode == 204 {
		return
	}

	if rubiResp.StatusCode == http.StatusBadRequest {
		err = &errortypes.BadInput{
			Message: fmt.Sprintf("HTTP status %d; body: %s", rubiResp.StatusCode, result.ResponseBody),
		}
	}

	if rubiResp.StatusCode != http.StatusOK {
		err = &errortypes.BadServerResponse{
			Message: fmt.Sprintf("HTTP status %d; body: %s", rubiResp.StatusCode, result.ResponseBody),
		}
		return
	}

	var bidResp openrtb2.BidResponse
	err = json.Unmarshal(body, &bidResp)
	if err != nil {
		err = &errortypes.BadServerResponse{
			Message: err.Error(),
		}
		return
	}
	if len(bidResp.SeatBid) == 0 {
		return
	}
	if len(bidResp.SeatBid[0].Bid) == 0 {
		return
	}
	bid := bidResp.SeatBid[0].Bid[0]

	result.Bid = &pbs.PBSBid{
		AdUnitCode:  bid.ImpID,
		Price:       bid.Price,
		Adm:         bid.AdM,
		Creative_id: bid.CrID,
		// for video, the width and height are undefined as there's no corresponding return value from XAPI
		Width:  bid.W,
		Height: bid.H,
		DealId: bid.DealID,
	}

	// Pull out any server-side determined targeting
	var rpExtTrg rubiconTargetingExt

	if err := json.Unmarshal([]byte(bid.Ext), &rpExtTrg); err == nil {
		// Converting string => array(string) to string => string
		targeting := make(map[string]string)

		// Only pick off the first for now
		for _, target := range rpExtTrg.RP.Targeting {
			targeting[target.Key] = target.Values[0]
		}

		result.Bid.AdServerTargeting = targeting
	}

	return
}

type callOneObject struct {
	requestJson bytes.Buffer
	mediaType   pbs.MediaType
}

func (a *RubiconAdapter) Call(ctx context.Context, req *pbs.PBSRequest, bidder *pbs.PBSBidder) (pbs.PBSBidSlice, error) {
	callOneObjects := make([]callOneObject, 0, len(bidder.AdUnits))
	supportedMediaTypes := []pbs.MediaType{pbs.MEDIA_TYPE_BANNER, pbs.MEDIA_TYPE_VIDEO}

	rubiReq, err := adapters.MakeOpenRTBGeneric(req, bidder, a.Name(), supportedMediaTypes)
	if err != nil {
		return nil, err
	}

	rubiReqImpCopy := rubiReq.Imp

	for i, unit := range bidder.AdUnits {
		// Fixes some segfaults. Since this is legacy code, I'm not looking into it too deeply
		if len(rubiReqImpCopy) <= i {
			break
		}
		// Only grab this ad unit
		// Not supporting multi-media-type add-unit yet
		thisImp := rubiReqImpCopy[i]

		// Amend it with RP-specific information
		var params rubiconParams
		err = json.Unmarshal(unit.Params, &params)
		if err != nil {
			return nil, &errortypes.BadInput{
				Message: err.Error(),
			}
		}

		var mint, mintVersion string
		mint = "prebid"
		mintVersion = req.SDK.Source + "_" + req.SDK.Platform + "_" + req.SDK.Version
		track := rubiconImpExtRPTrack{Mint: mint, MintVersion: mintVersion}

		impExt := rubiconImpExt{RP: rubiconImpExtRP{
			ZoneID: params.ZoneId,
			Target: params.Inventory,
			Track:  track,
		}}
		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			continue
		}

		// Copy the $.user object and amend with $.user.ext.rp.target
		// Copy avoids race condition since it points to ref & shared with other adapters
		userCopy := *rubiReq.User
		userExt := rubiconUserExt{RP: rubiconUserExtRP{Target: params.Visitor}}
		userCopy.Ext, err = json.Marshal(&userExt)
		// Assign back our copy
		rubiReq.User = &userCopy

		deviceCopy := *rubiReq.Device
		deviceExt := rubiconDeviceExt{RP: rubiconDeviceExtRP{PixelRatio: rubiReq.Device.PxRatio}}
		deviceCopy.Ext, err = json.Marshal(&deviceExt)
		rubiReq.Device = &deviceCopy

		if thisImp.Video != nil {

			videoSizeId := params.Video.VideoSizeID
			if videoSizeId == 0 {
				resolvedSizeId, err := resolveVideoSizeId(thisImp.Video.Placement, thisImp.Instl, thisImp.ID)
				if err == nil {
					videoSizeId = resolvedSizeId
				} else {
					continue
				}
			}

			videoExt := rubiconVideoExt{Skip: params.Video.Skip, SkipDelay: params.Video.SkipDelay, RP: rubiconVideoExtRP{SizeID: videoSizeId}}
			thisImp.Video.Ext, err = json.Marshal(&videoExt)
		} else {
			primarySizeID, altSizeIDs, err := parseRubiconSizes(unit.Sizes)
			if err != nil {
				continue
			}
			bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: primarySizeID, AltSizeIDs: altSizeIDs, MIME: "text/html"}}
			thisImp.Banner.Ext, err = json.Marshal(&bannerExt)
		}

		siteExt := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: params.SiteId}}
		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: params.AccountId}}
		var rubiconUser rubiconUser
		err = json.Unmarshal(req.PBSUser, &rubiconUser)

		if rubiReq.Site != nil {
			siteCopy := *rubiReq.Site
			siteCopy.Ext, err = json.Marshal(&siteExt)
			siteCopy.Publisher = &openrtb2.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			siteCopy.Content = &openrtb2.Content{}
			siteCopy.Content.Language = rubiconUser.Language
			rubiReq.Site = &siteCopy
		} else {
			site := &openrtb2.Site{}
			site.Content = &openrtb2.Content{}
			site.Content.Language = rubiconUser.Language
			rubiReq.Site = site
		}

		if rubiReq.App != nil {
			appCopy := *rubiReq.App
			appCopy.Ext, err = json.Marshal(&siteExt)
			appCopy.Publisher = &openrtb2.Publisher{}
			appCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiReq.App = &appCopy
		}

		rubiReq.Imp = []openrtb2.Imp{thisImp}

		var reqBuffer bytes.Buffer
		err = json.NewEncoder(&reqBuffer).Encode(rubiReq)
		if err != nil {
			return nil, err
		}
		callOneObjects = append(callOneObjects, callOneObject{reqBuffer, unit.MediaTypes[0]})
	}
	if len(callOneObjects) == 0 {
		return nil, &errortypes.BadInput{
			Message: "Invalid ad unit/imp",
		}
	}

	ch := make(chan adapters.CallOneResult)
	for _, obj := range callOneObjects {
		go func(bidder *pbs.PBSBidder, reqJSON bytes.Buffer, mediaType pbs.MediaType) {
			result, err := a.callOne(ctx, reqJSON)
			result.Error = err
			if result.Bid != nil {
				result.Bid.BidderCode = bidder.BidderCode
				result.Bid.BidID = bidder.LookupBidID(result.Bid.AdUnitCode)
				if result.Bid.BidID == "" {
					result.Error = &errortypes.BadServerResponse{
						Message: fmt.Sprintf("Unknown ad unit code '%s'", result.Bid.AdUnitCode),
					}
					result.Bid = nil
				} else {
					// no need to check whether mediaTypes is nil or length of zero, pbs.ParsePBSRequest will cover
					// these cases.
					// for media types other than banner and video, pbs.ParseMediaType will throw error.
					// we may want to create a map/switch cases to support more media types in the future.
					if mediaType == pbs.MEDIA_TYPE_VIDEO {
						result.Bid.CreativeMediaType = string(openrtb_ext.BidTypeVideo)
					} else {
						result.Bid.CreativeMediaType = string(openrtb_ext.BidTypeBanner)
					}
				}
			}
			ch <- result
		}(bidder, obj.requestJson, obj.mediaType)
	}

	bids := make(pbs.PBSBidSlice, 0)
	for i := 0; i < len(callOneObjects); i++ {
		result := <-ch
		if result.Bid != nil && result.Bid.Price != 0 {
			bids = append(bids, result.Bid)
		}
		if req.IsDebug {
			debug := &pbs.BidderDebug{
				RequestURI:   a.URI,
				RequestBody:  callOneObjects[i].requestJson.String(),
				StatusCode:   result.StatusCode,
				ResponseBody: result.ResponseBody,
			}
			bidder.Debug = append(bidder.Debug, debug)
		}
		if result.Error != nil {
			if glog.V(2) {
				glog.Infof("Error from rubicon adapter: %v", result.Error)
			}
			err = result.Error
		}
	}

	if len(bids) == 0 {
		return nil, err
	}
	return bids, nil
}

func resolveVideoSizeId(placement openrtb2.VideoPlacementType, instl int8, impId string) (sizeID int, err error) {
	if placement != 0 {
		if placement == 1 {
			return 201, nil
		}
		if placement == 3 {
			return 203, nil
		}
	}

	if instl == 1 {
		return 202, nil
	}
	return 0, &errortypes.BadInput{
		Message: fmt.Sprintf("video.size_id can not be resolved in impression with id : %s", impId),
	}
}

func appendTrackerToUrl(uri string, tracker string) (res string) {
	// Append integration method. Adapter init happens once
	urlObject, err := url.Parse(uri)
	// No other exception throwing mechanism in this stack, so ignoring parse errors.
	if err == nil {
		values := urlObject.Query()
		values.Add("tk_xint", tracker)
		urlObject.RawQuery = values.Encode()
		res = urlObject.String()
	} else {
		res = uri
	}
	return
}

// Builder builds a new instance of the Rubicon adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	uri := appendTrackerToUrl(config.Endpoint, config.XAPI.Tracker)

	bidder := &RubiconAdapter{
		URI:          uri,
		XAPIUsername: config.XAPI.Username,
		XAPIPassword: config.XAPI.Password,
	}
	return bidder, nil
}

func NewRubiconLegacyAdapter(httpConfig *adapters.HTTPAdapterConfig, uri string, xuser string, xpass string, tracker string) *RubiconAdapter {
	a := adapters.NewHTTPAdapter(httpConfig)

	uri = appendTrackerToUrl(uri, tracker)

	return &RubiconAdapter{
		http:         a,
		URI:          uri,
		XAPIUsername: xuser,
		XAPIPassword: xpass,
	}
}

func (a *RubiconAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)
	errs := make([]error, 0, len(request.Imp))
	var err error
	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	requestImpCopy := request.Imp

	rubiconRequest := *request
	for i := 0; i < numRequests; i++ {
		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var rubiconExt openrtb_ext.ExtImpRubicon
		if err = json.Unmarshal(bidderExt.Bidder, &rubiconExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		target := rubiconExt.Inventory
		if rubiconExt.Inventory != nil {
			rubiconExtInventory := make(map[string]interface{})
			if err := json.Unmarshal(rubiconExt.Inventory, &rubiconExtInventory); err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}

			var extImpWithContext ExtImpWithContext
			if err := json.Unmarshal(thisImp.Ext, &extImpWithContext); err != nil {
				errs = append(errs, &errortypes.BadInput{
					Message: err.Error(),
				})
				continue
			}

			// Copy imp[].ext.context.data.adslot is copied to imp[].ext.rp.target.dfp_ad_unit_code,
			// but with any leading slash dropped
			adSlot := extImpWithContext.Context.Data.AdSlot
			if adSlot != "" {
				rubiconExtInventory["dfp_ad_unit_code"] = strings.TrimLeft(adSlot, "/")

				target, err = json.Marshal(&rubiconExtInventory)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
		}

		impExt := rubiconImpExt{
			RP: rubiconImpExtRP{
				ZoneID: rubiconExt.ZoneId,
				Target: target,
				Track:  rubiconImpExtRPTrack{Mint: "", MintVersion: ""},
			},
		}
		thisImp.Ext, err = json.Marshal(&impExt)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		resolvedBidFloor, err := resolveBidFloor(thisImp.BidFloor, thisImp.BidFloorCur, reqInfo)
		if err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: fmt.Sprintf("Unable to convert provided bid floor currency from %s to USD",
					thisImp.BidFloorCur),
			})
			continue
		}

		if resolvedBidFloor > 0 {
			thisImp.BidFloorCur = "USD"
			thisImp.BidFloor = resolvedBidFloor
		}

		if request.User != nil {
			userCopy := *request.User

			target, err := updateExtWithIabAttribute(rubiconExt.Visitor, userCopy.Data, []int{4})
			if err != nil {
				errs = append(errs, err)
				continue
			}
			userExtRP := rubiconUserExt{RP: rubiconUserExtRP{Target: target}}

			if request.User.Ext != nil {
				var userExt *openrtb_ext.ExtUser
				if err = json.Unmarshal(userCopy.Ext, &userExt); err != nil {
					errs = append(errs, &errortypes.BadInput{
						Message: err.Error(),
					})
					continue
				}
				userExtRP.Consent = userExt.Consent
				userExtRP.Eids = userExt.Eids

				// set user.ext.tpid
				if len(userExt.Eids) > 0 {
					mappedRubiconUidsParam, errors := getTpIdsAndSegments(userExt.Eids)
					if len(errors) > 0 {
						errs = append(errs, errors...)
						continue
					}

					if err := updateUserExtWithTpIdsAndSegments(&userExtRP, mappedRubiconUidsParam); err != nil {
						errs = append(errs, err)
						continue
					}

					userExtRP.LiverampIdl = mappedRubiconUidsParam.liverampIdl
				}
			}

			userCopy.Ext, err = json.Marshal(&userExtRP)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			rubiconRequest.User = &userCopy
		}

		if request.Device != nil {
			deviceCopy := *request.Device
			deviceExt := rubiconDeviceExt{RP: rubiconDeviceExtRP{PixelRatio: request.Device.PxRatio}}
			deviceCopy.Ext, err = json.Marshal(&deviceExt)
			rubiconRequest.Device = &deviceCopy
		}

		isVideo := isVideo(thisImp)
		if isVideo {
			videoCopy := *thisImp.Video

			videoSizeId := rubiconExt.Video.VideoSizeID
			if videoSizeId == 0 {
				resolvedSizeId, err := resolveVideoSizeId(thisImp.Video.Placement, thisImp.Instl, thisImp.ID)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				videoSizeId = resolvedSizeId
			}

			// if imp.ext.is_rewarded_inventory = 1, set imp.video.ext.videotype = "rewarded"
			var videoType = ""
			if bidderExt.Prebid != nil && bidderExt.Prebid.IsRewardedInventory == 1 {
				videoType = "rewarded"
			}
			videoExt := rubiconVideoExt{Skip: rubiconExt.Video.Skip, SkipDelay: rubiconExt.Video.SkipDelay, VideoType: videoType, RP: rubiconVideoExtRP{SizeID: videoSizeId}}
			videoCopy.Ext, err = json.Marshal(&videoExt)
			thisImp.Video = &videoCopy
			thisImp.Banner = nil
		} else {
			primarySizeID, altSizeIDs, err := parseRubiconSizes(thisImp.Banner.Format)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			bannerExt := rubiconBannerExt{RP: rubiconBannerExtRP{SizeID: primarySizeID, AltSizeIDs: altSizeIDs, MIME: "text/html"}}
			bannerCopy := *thisImp.Banner
			bannerCopy.Ext, err = json.Marshal(&bannerExt)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			thisImp.Banner = &bannerCopy
			thisImp.Video = nil
		}

		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: rubiconExt.AccountId}}

		if request.Site != nil {
			siteCopy := *request.Site
			siteExtRP := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: rubiconExt.SiteId}}
			if siteCopy.Content != nil {
				target, err := updateExtWithIabAttribute(nil, siteCopy.Content.Data, []int{1, 2})
				if err != nil {
					errs = append(errs, err)
					continue
				}
				siteExtRP.RP.Target = target
			}

			siteCopy.Ext, err = json.Marshal(&siteExtRP)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			siteCopy.Publisher = &openrtb2.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiconRequest.Site = &siteCopy
		} else {
			appCopy := *request.App
			appCopy.Ext, err = json.Marshal(rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: rubiconExt.SiteId}})
			appCopy.Publisher = &openrtb2.Publisher{}
			appCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiconRequest.App = &appCopy
		}

		reqBadv := request.BAdv
		if reqBadv != nil {
			if len(reqBadv) > badvLimitSize {
				rubiconRequest.BAdv = reqBadv[:badvLimitSize]
			}
		}

		rubiconRequest.Imp = []openrtb2.Imp{thisImp}
		rubiconRequest.Cur = nil
		rubiconRequest.Ext = nil

		reqJSON, err := json.Marshal(rubiconRequest)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     a.URI,
			Body:    reqJSON,
			Headers: headers,
		}
		reqData.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

func resolveBidFloor(bidFloor float64, bidFloorCur string, reqInfo *adapters.ExtraRequestInfo) (float64, error) {
	if bidFloor > 0 && bidFloorCur != "" && strings.ToUpper(bidFloorCur) != "USD" {
		return reqInfo.ConvertCurrency(bidFloor, bidFloorCur, "USD")
	}

	return bidFloor, nil
}

func updateExtWithIabAttribute(target json.RawMessage, data []openrtb2.Data, segTaxes []int) (json.RawMessage, error) {
	var segmentIdsToCopy = getSegmentIdsToCopy(data, segTaxes)
	if len(segmentIdsToCopy) == 0 {
		return target, nil
	}

	extRPTarget := make(map[string]interface{})

	if target != nil {
		if err := json.Unmarshal(target, &extRPTarget); err != nil {
			return nil, &errortypes.BadInput{Message: err.Error()}
		}
	}

	extRPTarget["iab"] = segmentIdsToCopy

	jsonTarget, err := json.Marshal(&extRPTarget)
	if err != nil {
		return nil, &errortypes.BadInput{Message: err.Error()}
	}
	return jsonTarget, nil
}

func getSegmentIdsToCopy(data []openrtb2.Data, segTaxValues []int) []string {
	var segmentIdsToCopy = make([]string, 0, len(data))

	for _, dataRecord := range data {
		if dataRecord.Ext != nil {
			var dataExtObject rubiconDataExt
			err := json.Unmarshal(dataRecord.Ext, &dataExtObject)
			if err != nil {
				continue
			}
			if contains(segTaxValues, dataExtObject.SegTax) {
				for _, segment := range dataRecord.Segment {
					segmentIdsToCopy = append(segmentIdsToCopy, segment.ID)
				}
			}
		}
	}
	return segmentIdsToCopy
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getTpIdsAndSegments(eids []openrtb_ext.ExtUserEid) (mappedRubiconUidsParam, []error) {
	rubiconUidsParam := mappedRubiconUidsParam{
		tpIds:    make([]rubiconExtUserTpID, 0),
		segments: make([]string, 0),
	}
	errs := make([]error, 0)

	for _, eid := range eids {
		switch eid.Source {
		case "adserver.org":
			uids := eid.Uids
			if len(uids) > 0 {
				uid := uids[0]

				if uid.Ext != nil {
					var eidUidExt rubiconUserExtEidUidExt
					if err := json.Unmarshal(uid.Ext, &eidUidExt); err != nil {
						errs = append(errs, &errortypes.BadInput{
							Message: err.Error(),
						})
						continue
					}

					if eidUidExt.RtiPartner == "TDID" {
						rubiconUidsParam.tpIds = append(rubiconUidsParam.tpIds, rubiconExtUserTpID{Source: "tdid", UID: uid.ID})
					}
				}
			}
		case "liveintent.com":
			uids := eid.Uids
			if len(uids) > 0 {
				uidId := uids[0].ID
				if uidId != "" {
					rubiconUidsParam.tpIds = append(rubiconUidsParam.tpIds, rubiconExtUserTpID{Source: "liveintent.com", UID: uidId})
				}

				if eid.Ext != nil {
					var eidExt rubiconUserExtEidExt
					if err := json.Unmarshal(eid.Ext, &eidExt); err != nil {
						errs = append(errs, &errortypes.BadInput{
							Message: err.Error(),
						})
						continue
					}
					rubiconUidsParam.segments = eidExt.Segments
				}
			}
		case "liveramp.com":
			uids := eid.Uids
			if len(uids) > 0 {
				uidId := uids[0].ID
				if uidId != "" && rubiconUidsParam.liverampIdl == "" {
					rubiconUidsParam.liverampIdl = uidId
				}
			}
		}
	}

	return rubiconUidsParam, errs
}

func updateUserExtWithTpIdsAndSegments(userExtRP *rubiconUserExt, rubiconUidsParam mappedRubiconUidsParam) error {
	if len(rubiconUidsParam.tpIds) > 0 {
		userExtRP.TpID = rubiconUidsParam.tpIds

		if rubiconUidsParam.segments != nil {
			userExtRPTarget := make(map[string]interface{})

			if userExtRP.RP.Target != nil {
				if err := json.Unmarshal(userExtRP.RP.Target, &userExtRPTarget); err != nil {
					return &errortypes.BadInput{Message: err.Error()}
				}
			}

			userExtRPTarget["LIseg"] = rubiconUidsParam.segments

			if target, err := json.Marshal(&userExtRPTarget); err != nil {
				return &errortypes.BadInput{Message: err.Error()}
			} else {
				userExtRP.RP.Target = target
			}
		}
	}
	return nil
}

func isVideo(imp openrtb2.Imp) bool {
	video := imp.Video
	if video != nil {
		// Do any other media types exist? Or check required video fields.
		return imp.Banner == nil || isFullyPopulatedVideo(video)
	}
	return false
}

func isFullyPopulatedVideo(video *openrtb2.Video) bool {
	// These are just recommended video fields for XAPI
	return video.MIMEs != nil && video.Protocols != nil && video.MaxDuration != 0 && video.Linearity != 0 && video.API != nil
}

func (a *RubiconAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var bidResp rubiconBidResponse
	if err := json.Unmarshal(response.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: err.Error(),
		}}
	}

	var bidReq openrtb2.BidRequest
	if err := json.Unmarshal(externalRequest.Body, &bidReq); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(5)

	bidType := openrtb_ext.BidTypeBanner

	isVideo := isVideo(bidReq.Imp[0])
	if isVideo {
		bidType = openrtb_ext.BidTypeVideo
	}

	impToCpmOverride := mapImpIdToCpmOverride(internalRequest.Imp)
	cmpOverride := cmpOverrideFromBidRequest(internalRequest)

	for _, sb := range bidResp.SeatBid {
		buyer, err := strconv.Atoi(sb.Buyer)
		if err != nil {
			buyer = 0
		}
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]

			updatedBidExt := updateBidExtWithMetaNetworkId(bid, buyer)
			if updatedBidExt != nil {
				bid.Ext = updatedBidExt
			}
			bidCmpOverride, ok := impToCpmOverride[bid.ImpID]
			if !ok || bidCmpOverride == 0 {
				bidCmpOverride = cmpOverride
			}

			if bidCmpOverride > 0 {
				bid.Price = bidCmpOverride
			}

			if bid.Price != 0 {
				// Since Rubicon XAPI returns only one bid per response
				// copy response.bidid to openrtb_response.seatbid.bid.bidid
				if bid.ID == "0" {
					bid.ID = bidResp.BidID
				}
				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}

func mapImpIdToCpmOverride(imps []openrtb2.Imp) map[string]float64 {
	impIdToCmpOverride := make(map[string]float64)
	for _, imp := range imps {
		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			continue
		}

		var rubiconExt openrtb_ext.ExtImpRubicon
		if err := json.Unmarshal(bidderExt.Bidder, &rubiconExt); err != nil {
			continue
		}

		impIdToCmpOverride[imp.ID] = rubiconExt.Debug.CpmOverride
	}
	return impIdToCmpOverride
}

func cmpOverrideFromBidRequest(bidRequest *openrtb2.BidRequest) float64 {
	var bidRequestExt bidRequestExt
	if err := json.Unmarshal(bidRequest.Ext, &bidRequestExt); err != nil {
		return 0
	}

	return bidRequestExt.Prebid.Bidders.Rubicon.Debug.CpmOverride
}

func updateBidExtWithMetaNetworkId(bid openrtb2.Bid, buyer int) json.RawMessage {
	if buyer <= 0 {
		return nil
	}
	var bidExt *extPrebid
	if bid.Ext != nil {
		if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
			return nil
		}
	}

	if bidExt != nil {
		if bidExt.Prebid != nil {
			if bidExt.Prebid.Meta != nil {
				bidExt.Prebid.Meta.NetworkID = buyer
			} else {
				bidExt.Prebid.Meta = &openrtb_ext.ExtBidPrebidMeta{NetworkID: buyer}
			}
		} else {
			bidExt.Prebid = &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: buyer}}
		}
	} else {
		bidExt = &extPrebid{Prebid: &openrtb_ext.ExtBidPrebid{Meta: &openrtb_ext.ExtBidPrebidMeta{NetworkID: buyer}}}
	}

	marshalledExt, err := json.Marshal(&bidExt)
	if err == nil {
		return marshalledExt
	}
	return nil
}
