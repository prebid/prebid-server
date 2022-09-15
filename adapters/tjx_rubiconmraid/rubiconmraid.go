package rubiconmraid

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/skanidlist"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Region ...
type Region string

const (
	USEast Region = "us_east"
	USWest Region = "us_west"
	EU     Region = "eu"
	APAC   Region = "apac"
)

const badvLimitSize = 50

type RubiconMRAIDAdapter struct {
	URI              string
	XAPIUsername     string
	XAPIPassword     string
	SupportedRegions map[Region]string
}

type bidRequestExt struct {
	Prebid bidRequestExtPrebid `json:"prebid"`
}

type bidRequestExtPrebid struct {
	Bidders bidRequestExtPrebidBidders `json:"bidders"`
}

type bidRequestExtPrebidBidders struct {
	Rubicon prebidBiddersRubicon `json:"rubiconmraid,omitempty"`
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
	RP                 rubiconImpExtRP    `json:"rp"`
	ViewabilityVendors []string           `json:"viewabilityvendors"`
	SKADN              *openrtb_ext.SKADN `json:"skadn,omitempty"`
}

type rubiconUserExtRP struct {
	Target json.RawMessage `json:"target,omitempty"`
}

type rubiconExtUserTpID struct {
	Source string `json:"source"`
	UID    string `json:"uid"`
}

type rubiconUserDataExt struct {
	TaxonomyName string `json:"taxonomyname"`
}

type rubiconUserExt struct {
	Consent     string                   `json:"consent,omitempty"`
	Eids        []openrtb_ext.ExtUserEid `json:"eids,omitempty"`
	TpID        []rubiconExtUserTpID     `json:"tpid,omitempty"`
	RP          rubiconUserExtRP         `json:"rp"`
	LiverampIdl string                   `json:"liveramp_idl,omitempty"`
}

type rubiconSiteExtRP struct {
	SiteID int `json:"site_id"`
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

type rubiconVideoExt struct {
	Skip      int               `json:"skip,omitempty"`
	SkipDelay int               `json:"skipdelay,omitempty"`
	VideoType string            `json:"videotype,omitempty"`
	RP        rubiconVideoExtRP `json:"rp"`
}

type rubiconVideoExtRP struct {
	SizeID int `json:"size_id,omitempty"`
}

type rubiconDeviceExtRP struct {
	PixelRatio float64 `json:"pixelratio"`
}

type rubiconDeviceExt struct {
	ATTS *openrtb_ext.IOSAppTrackingStatus `json:"atts,omitempty"`
	IFV  string                            `json:"ifv,omitempty"`
	RP   rubiconDeviceExtRP                `json:"rp"`
}

type reqSourceExt struct {
	HeaderBidding int `json:"header_bidding,omitempty"`
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

// hack for aqid in response
type rubiconBidResponse struct {
	SeatBid []rubiconSeatBid `json:"seatbid,omitempty"`
}
type rubiconSeatBid struct {
	Bid []rubiconBid `json:"bid"`
}
type rubiconBid struct {
	AqID string `json:"aqid,omitempty"`
}

func (rbr rubiconBidResponse) AqID() string {
	if len(rbr.SeatBid) == 0 {
		return ""
	}

	if len(rbr.SeatBid[0].Bid) == 0 {
		return ""
	}

	return rbr.SeatBid[0].Bid[0].AqID
}

type rubiconBidExt struct {
	RP     rubiconBidExtRP     `json:"rp,omitempty"`
	SKADN  rubiconBidExtSKADN  `json:"skadn,omitempty"`
	Prebid rubiconBidExtPrebid `json:"prebid,omitempty"`
}

type rubiconBidExtRP struct {
	AdVid  int    `json:"advid,omitempty"`
	Mime   string `json:"mime,omitempty"`
	SizeID int    `json:"size_id,omitempty"`
	AqID   string `json:"aqid,omitempty"`
}

type rubiconBidExtSKADN struct {
	Version    string                       `json:"version,omitempty"`    // Version of SKAdNetwork desired. Must be 2.0 or above.
	Network    string                       `json:"network,omitempty"`    // Ad network identifier used in signature. Should match one of the items in the skadnetids array in the request
	Campaign   string                       `json:"campaign,omitempty"`   // Campaign ID compatible with Apple’s spec. As of 2.0, should be an integer between 1 and 100, expressed as a string
	ITunesItem string                       `json:"itunesitem,omitempty"` // ID of advertiser’s app in Apple’s app store. Should match BidResponse.bid.bundle
	Nonce      string                       `json:"nonce,omitempty"`      // An id unique to each ad response
	SourceApp  string                       `json:"sourceapp,omitempty"`  // ID of publisher’s app in Apple’s app store. Should match BidRequest.imp.ext.skad.sourceapp
	Timestamp  string                       `json:"timestamp,omitempty"`  // Unix time in millis string used at the time of signature
	Signature  string                       `json:"signature,omitempty"`  // SKAdNetwork signature as specified by Apple
	Fidelities []rubiconBidExtSKADNFidelity `json:"fidelities,omitempty"` // Supports multiple fidelity types introduced in SKAdNetwork v2.2
}

type rubiconBidExtSKADNFidelity struct {
	Fidelity  int    `json:"fidelity"`            // The fidelity-type of the attribution to track
	Signature string `json:"signature,omitempty"` // SKAdNetwork signature as specified by Apple
	Nonce     string `json:"nonce,omitempty"`     // An id unique to each ad response
	Timestamp string `json:"timestamp,omitempty"` // Unix time in millis string used at the time of signature
}

type rubiconBidExtPrebid struct {
	Type string `json:"type,omitempty"`
}

type mappedRubiconUidsParam struct {
	tpIds       []rubiconExtUserTpID
	segments    []string
	liverampIdl string
}

// MAS algorithm
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
func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	uri := appendTrackerToUrl(config.Endpoint, config.XAPI.Tracker)

	bidder := &RubiconMRAIDAdapter{
		URI:          uri,
		XAPIUsername: config.XAPI.Username,
		XAPIPassword: config.XAPI.Password,
		SupportedRegions: map[Region]string{
			USEast: appendTrackerToUrl(config.XAPI.EndpointUSEast, config.XAPI.Tracker),
			USWest: appendTrackerToUrl(config.XAPI.EndpointUSWest, config.XAPI.Tracker),
			EU:     appendTrackerToUrl(config.XAPI.EndpointEU, config.XAPI.Tracker),
			APAC:   appendTrackerToUrl(config.XAPI.EndpointAPAC, config.XAPI.Tracker),
		},
	}
	return bidder, nil
}

func (a *RubiconMRAIDAdapter) MakeRequests(request *openrtb2.BidRequest, _ *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	numRequests := len(request.Imp)

	errs := make([]error, 0, len(request.Imp))
	var err error
	requestData := make([]*adapters.RequestData, 0, numRequests)
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")
	headers.Add("User-Agent", "prebid-server/1.0")

	requestImpCopy := request.Imp

	// copy the bidder request
	rubiconRequest := *request

	var srcExt *reqSourceExt
	if request.Source != nil && request.Source.Ext != nil {
		if err := json.Unmarshal(request.Source.Ext, &srcExt); err != nil {
			errs = append(errs, err)
		}
	}

	for i := 0; i < numRequests; i++ {
		skanSent := false
		placementType := adapters.Interstitial

		thisImp := requestImpCopy[i]

		var bidderExt adapters.ExtImpBidder
		if err = json.Unmarshal(thisImp.Ext, &bidderExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		var rubiconExt openrtb_ext.ExtImpTJXRubicon
		if err = json.Unmarshal(bidderExt.Bidder, &rubiconExt); err != nil {
			errs = append(errs, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		// This check is for identifying if the request comes from TJX
		if srcExt != nil && srcExt.HeaderBidding == 1 {
			rubiconRequest.BApp = nil
			rubiconRequest.BAdv = nil

			if rubiconExt.Blocklist.BApp != nil {
				rubiconRequest.BApp = rubiconExt.Blocklist.BApp
			}
			if rubiconExt.Blocklist.BAdv != nil {
				rubiconRequest.BAdv = rubiconExt.Blocklist.BAdv
			}
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
			ViewabilityVendors: rubiconExt.ViewabilityVendors,
		}

		if rubiconExt.SKADNSupported {
			skanIDList := skanidlist.Get(openrtb_ext.BidderRubicon)

			skadn := adapters.FilterPrebidSKADNExt(bidderExt.Prebid, skanIDList)

			// only add if present
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

		resolvedBidFloor, resolvedBidFloorCur := resolveBidFloorAttributes(thisImp.BidFloor, thisImp.BidFloorCur)
		thisImp.BidFloorCur = resolvedBidFloorCur
		thisImp.BidFloor = resolvedBidFloor

		if request.User != nil {
			userCopy := *request.User
			userExtRP := rubiconUserExt{RP: rubiconUserExtRP{Target: rubiconExt.Visitor}}

			if err := updateUserExtWithIabAttribute(&userExtRP, userCopy.Data); err != nil {
				errs = append(errs, err)
				continue
			}

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
			atts, _ := openrtb_ext.ParseDeviceExtATTS(request.Device.Ext)
			ifv, _ := jsonparser.GetString(request.Device.Ext, "ifv")
			deviceExt := rubiconDeviceExt{
				ATTS: atts,
				IFV:  ifv,
				RP: rubiconDeviceExtRP{
					PixelRatio: request.Device.PxRatio,
				},
			}
			deviceCopy.Ext, err = json.Marshal(&deviceExt)
			rubiconRequest.Device = &deviceCopy
		}

		// RubiconMRAID Bidder is responsible only for Banner MRAID requests
		if thisImp.Video != nil {
			thisImp.Video = nil
		}

		if thisImp.Banner != nil {
			if rubiconExt.MRAIDSupported {
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
			} else {
				thisImp.Banner = nil
			}
		}

		// This is a safe guard to prevent making request for nil impressions
		if thisImp.Video == nil && thisImp.Banner == nil {
			continue
		}

		// Overwrite BidFloor if present
		if rubiconExt.BidFloor != nil {
			thisImp.BidFloor = *rubiconExt.BidFloor
		}

		siteExt := rubiconSiteExt{RP: rubiconSiteExtRP{SiteID: rubiconExt.SiteId}}
		pubExt := rubiconPubExt{RP: rubiconPubExtRP{AccountID: rubiconExt.AccountId}}

		if request.Site != nil {
			siteCopy := *request.Site
			siteCopy.Ext, err = json.Marshal(&siteExt)
			siteCopy.Publisher = &openrtb2.Publisher{}
			siteCopy.Publisher.Ext, err = json.Marshal(&pubExt)
			rubiconRequest.Site = &siteCopy
		}
		if request.App != nil {
			appCopy := *request.App
			appCopy.Ext, err = json.Marshal(&siteExt)
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

		uri := a.URI
		if endpoint, ok := a.SupportedRegions[Region(rubiconExt.Region)]; ok {
			uri = endpoint
		}

		reqData := &adapters.RequestData{
			Method:  "POST",
			Uri:     uri,
			Body:    reqJSON,
			Headers: headers,

			TapjoyData: adapters.TapjoyData{
				Bidder:        string(openrtb_ext.BidderRubiconMRAID),
				PlacementType: placementType,
				Region:        rubiconExt.Region,
				SKAN: adapters.SKAN{
					Supported: rubiconExt.SKADNSupported,
					Sent:      skanSent,
				},
				MRAID: adapters.MRAID{
					Supported: rubiconExt.MRAIDSupported,
				},
			},
		}
		reqData.SetBasicAuth(a.XAPIUsername, a.XAPIPassword)
		requestData = append(requestData, reqData)
	}

	return requestData, errs
}

// Will be replaced after https://github.com/prebid/prebid-server/issues/1482 resolution
func resolveBidFloorAttributes(bidFloor float64, bidFloorCur string) (float64, string) {
	if bidFloor > 0 {
		if strings.ToUpper(bidFloorCur) == "EUR" {
			return bidFloor * 1.2, "USD"
		}
	}

	return bidFloor, bidFloorCur
}

func updateUserExtWithIabAttribute(userExtRP *rubiconUserExt, data []openrtb2.Data) error {
	var segmentIdsToCopy = make([]string, 0)

	for _, dataRecord := range data {
		if dataRecord.Ext != nil {
			var dataExtObject rubiconUserDataExt
			err := json.Unmarshal(dataRecord.Ext, &dataExtObject)
			if err != nil {
				continue
			}
			if strings.EqualFold(dataExtObject.TaxonomyName, "iab") {
				for _, segment := range dataRecord.Segment {
					segmentIdsToCopy = append(segmentIdsToCopy, segment.ID)
				}
			}
		}
	}

	userExtRPTarget := make(map[string]interface{})

	if userExtRP.RP.Target != nil {
		if err := json.Unmarshal(userExtRP.RP.Target, &userExtRPTarget); err != nil {
			return &errortypes.BadInput{Message: err.Error()}
		}
	}

	userExtRPTarget["iab"] = segmentIdsToCopy

	if target, err := json.Marshal(&userExtRPTarget); err != nil {
		return &errortypes.BadInput{Message: err.Error()}
	} else {
		userExtRP.RP.Target = target
	}

	return nil
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

func (a *RubiconMRAIDAdapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, response *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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

	var rubiconBidResp rubiconBidResponse
	if err := json.Unmarshal(response.Body, &rubiconBidResp); err != nil {
		// explicitly setting empty RubiconBidResponse
		rubiconBidResp = rubiconBidResponse{}
	}

	var bidResp openrtb2.BidResponse
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
		for i := 0; i < len(sb.Bid); i++ {
			bid := sb.Bid[i]

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

				injectAqID(&bid, rubiconBidResp.AqID(), bidType)

				bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
					Bid:     &bid,
					BidType: bidType,
				})
			}
		}
	}

	return bidResponse, nil
}

func injectAqID(bid *openrtb2.Bid, aqid string, bidType openrtb_ext.BidType) {
	var bidExt rubiconBidExt
	if err := json.Unmarshal(bid.Ext, &bidExt); err != nil {
		return
	}

	bidExt.RP.AqID = aqid
	bidExt.Prebid = rubiconBidExtPrebid{
		Type: string(bidType),
	}
	rawBidExt, err := json.Marshal(bidExt)
	if err != nil {
		return
	}

	bid.Ext = rawBidExt
	return
}

func cmpOverrideFromBidRequest(bidRequest *openrtb2.BidRequest) float64 {
	var bidRequestExt bidRequestExt
	if err := json.Unmarshal(bidRequest.Ext, &bidRequestExt); err != nil {
		return 0
	}

	return bidRequestExt.Prebid.Bidders.Rubicon.Debug.CpmOverride
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
