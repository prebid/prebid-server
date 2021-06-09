package pbs

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/httputil"
	"github.com/prebid/prebid-server/util/iputil"

	"github.com/blang/semver"
	"github.com/buger/jsonparser"
	"github.com/golang/glog"
	"golang.org/x/net/publicsuffix"
)

const MAX_BIDDERS = 8

type MediaType byte

const (
	MEDIA_TYPE_BANNER MediaType = iota
	MEDIA_TYPE_VIDEO
)

type ConfigCache interface {
	LoadConfig(string) ([]Bids, error)
}

type Bids struct {
	BidderCode string          `json:"bidder"`
	BidID      string          `json:"bid_id"`
	Params     json.RawMessage `json:"params"`
}

// Structure for holding video-specific information
type PBSVideo struct {
	//Content MIME types supported. Popular MIME types may include “video/x-ms-wmv” for Windows Media and “video/x-flv” for Flash Video.
	Mimes []string `json:"mimes,omitempty"`

	//Minimum video ad duration in seconds.
	Minduration int64 `json:"minduration,omitempty"`

	// Maximum video ad duration in seconds.
	Maxduration int64 `json:"maxduration,omitempty"`

	//Indicates the start delay in seconds for pre-roll, mid-roll, or post-roll ad placements.
	Startdelay int64 `json:"startdelay,omitempty"`

	// Indicates if the player will allow the video to be skipped ( 0 = no, 1 = yes).
	Skippable int `json:"skippable,omitempty"`

	// Playback method code Description
	// 1 - Initiates on Page Load with Sound On
	// 2 - Initiates on Page Load with Sound Off by Default
	// 3 - Initiates on Click with Sound On
	// 4 - Initiates on Mouse-Over with Sound On
	// 5 - Initiates on Entering Viewport with Sound On
	// 6 - Initiates on Entering Viewport with Sound Off by Default
	PlaybackMethod int8 `json:"playback_method,omitempty"`

	//protocols as specified in ORTB 5.8
	// 1 VAST 1.0
	// 2 VAST 2.0
	// 3 VAST 3.0
	// 4 VAST 1.0 Wrapper
	// 5 VAST 2.0 Wrapper
	// 6 VAST 3.0 Wrapper
	// 7 VAST 4.0
	// 8 VAST 4.0 Wrapper
	// 9 DAAST 1.0
	// 10 DAAST 1.0 Wrapper
	Protocols []int8 `json:"protocols,omitempty"`
}

type AdUnit struct {
	Code       string            `json:"code"`
	TopFrame   int8              `json:"is_top_frame"`
	Sizes      []openrtb2.Format `json:"sizes"`
	Bids       []Bids            `json:"bids"`
	ConfigID   string            `json:"config_id"`
	MediaTypes []string          `json:"media_types"`
	Instl      int8              `json:"instl"`
	Video      PBSVideo          `json:"video"`
}

type PBSAdUnit struct {
	Sizes      []openrtb2.Format
	TopFrame   int8
	Code       string
	BidID      string
	Params     json.RawMessage
	Video      PBSVideo
	MediaTypes []MediaType
	Instl      int8
}

func ParseMediaType(s string) (MediaType, error) {
	mediaTypes := map[string]MediaType{"BANNER": MEDIA_TYPE_BANNER, "VIDEO": MEDIA_TYPE_VIDEO}
	t, ok := mediaTypes[strings.ToUpper(s)]
	if !ok {
		return 0, fmt.Errorf("Invalid MediaType %s", s)
	}
	return t, nil
}

type SDK struct {
	Version  string `json:"version"`
	Source   string `json:"source"`
	Platform string `json:"platform"`
}

type PBSBidder struct {
	BidderCode   string                 `json:"bidder"`
	AdUnitCode   string                 `json:"ad_unit,omitempty"` // for index to dedup responses
	ResponseTime int                    `json:"response_time_ms,omitempty"`
	NumBids      int                    `json:"num_bids,omitempty"`
	Error        string                 `json:"error,omitempty"`
	NoCookie     bool                   `json:"no_cookie,omitempty"`
	NoBid        bool                   `json:"no_bid,omitempty"`
	UsersyncInfo *usersync.UsersyncInfo `json:"usersync,omitempty"`
	Debug        []*BidderDebug         `json:"debug,omitempty"`

	AdUnits []PBSAdUnit `json:"-"`
}

func (bidder *PBSBidder) LookupBidID(Code string) string {
	for _, unit := range bidder.AdUnits {
		if unit.Code == Code {
			return unit.BidID
		}
	}
	return ""
}

func (bidder *PBSBidder) LookupAdUnit(Code string) (unit *PBSAdUnit) {
	for _, unit := range bidder.AdUnits {
		if unit.Code == Code {
			return &unit
		}
	}
	return nil
}

type PBSRequest struct {
	AccountID     string           `json:"account_id"`
	Tid           string           `json:"tid"`
	CacheMarkup   int8             `json:"cache_markup"`
	SortBids      int8             `json:"sort_bids"`
	MaxKeyLength  int8             `json:"max_key_length"`
	Secure        int8             `json:"secure"`
	TimeoutMillis int64            `json:"timeout_millis"`
	AdUnits       []AdUnit         `json:"ad_units"`
	IsDebug       bool             `json:"is_debug"`
	App           *openrtb2.App    `json:"app"`
	Device        *openrtb2.Device `json:"device"`
	PBSUser       json.RawMessage  `json:"user"`
	SDK           *SDK             `json:"sdk"`

	// internal
	Bidders []*PBSBidder        `json:"-"`
	User    *openrtb2.User      `json:"-"`
	Cookie  *usersync.PBSCookie `json:"-"`
	Url     string              `json:"-"`
	Domain  string              `json:"-"`
	Regs    *openrtb2.Regs      `json:"-"`
	Start   time.Time
}

func ConfigGet(cache cache.Cache, id string) ([]Bids, error) {
	conf, err := cache.Config().Get(id)
	if err != nil {
		return nil, err
	}

	bids := make([]Bids, 0)
	err = json.Unmarshal([]byte(conf), &bids)
	if err != nil {
		return nil, err
	}

	return bids, nil
}

func ParseMediaTypes(types []string) []MediaType {
	var mtypes []MediaType
	mtmap := make(map[MediaType]bool)

	if types == nil {
		mtypes = append(mtypes, MEDIA_TYPE_BANNER)
	} else {
		for _, t := range types {
			mt, er := ParseMediaType(t)
			if er != nil {
				glog.Infof("Invalid media type: %s", er)
			} else {
				if !mtmap[mt] {
					mtypes = append(mtypes, mt)
					mtmap[mt] = true
				}
			}
		}
		if len(mtypes) == 0 {
			mtypes = append(mtypes, MEDIA_TYPE_BANNER)
		}
	}
	return mtypes
}

var ipv4Validator iputil.IPValidator = iputil.VersionIPValidator{iputil.IPv4}

func ParsePBSRequest(r *http.Request, cfg *config.AuctionTimeouts, cache cache.Cache, hostCookieConfig *config.HostCookie) (*PBSRequest, error) {
	defer r.Body.Close()

	pbsReq := &PBSRequest{}
	err := json.NewDecoder(r.Body).Decode(&pbsReq)
	if err != nil {
		return nil, err
	}
	pbsReq.Start = time.Now()

	if len(pbsReq.AdUnits) == 0 {
		return nil, fmt.Errorf("No ad units specified")
	}

	pbsReq.TimeoutMillis = int64(cfg.LimitAuctionTimeout(time.Duration(pbsReq.TimeoutMillis)*time.Millisecond) / time.Millisecond)

	if pbsReq.Device == nil {
		pbsReq.Device = &openrtb2.Device{}
	}
	if ip, _ := httputil.FindIP(r, ipv4Validator); ip != nil {
		pbsReq.Device.IP = ip.String()
	}

	if pbsReq.SDK == nil {
		pbsReq.SDK = &SDK{}
	}

	// Early versions of prebid mobile are sending requests with gender indicated by numbers,
	// those traffic can't be parsed by latest Prebid Server after the change of gender to use string so clients using early versions can't be monetized.
	// To handle those traffic, adding a check here to ignore the sent gender for versions lower than 0.0.2.
	v1, err := semver.Make(pbsReq.SDK.Version)
	v2, err := semver.Make("0.0.2")
	if v1.Compare(v2) >= 0 && pbsReq.PBSUser != nil {
		err = json.Unmarshal([]byte(pbsReq.PBSUser), &pbsReq.User)
		if err != nil {
			return nil, err
		}
	}

	if pbsReq.User == nil {
		pbsReq.User = &openrtb2.User{}
	}

	// use client-side data for web requests
	if pbsReq.App == nil {
		pbsReq.Cookie = usersync.ParsePBSCookieFromRequest(r, hostCookieConfig)

		pbsReq.Device.UA = r.Header.Get("User-Agent")

		pbsReq.Url = r.Header.Get("Referer") // must be specified in the header
		// TODO: this should explicitly put us in test mode
		if r.FormValue("url_override") != "" {
			pbsReq.Url = r.FormValue("url_override")
		}
		if strings.Index(pbsReq.Url, "http") == -1 {
			pbsReq.Url = fmt.Sprintf("http://%s", pbsReq.Url)
		}

		url, err := url.Parse(pbsReq.Url)
		if err != nil {
			return nil, fmt.Errorf("Invalid URL '%s': %v", pbsReq.Url, err)
		}

		if url.Host == "" {
			return nil, fmt.Errorf("Host not found from URL '%v'", url)
		}

		pbsReq.Domain, err = publicsuffix.EffectiveTLDPlusOne(url.Host)
		if err != nil {
			return nil, fmt.Errorf("Invalid URL '%s': %v", url.Host, err)
		}
	}

	if r.FormValue("debug") == "1" {
		pbsReq.IsDebug = true
	}

	if httputil.IsSecure(r) {
		pbsReq.Secure = 1
	}

	pbsReq.Bidders = make([]*PBSBidder, 0, MAX_BIDDERS)

	for _, unit := range pbsReq.AdUnits {
		bidders := unit.Bids
		if unit.ConfigID != "" {
			bidders, err = ConfigGet(cache, unit.ConfigID)
			if err != nil {
				if _, notFound := err.(*stored_requests.NotFoundError); !notFound {
					glog.Warningf("Failed to load config '%s' from cache: %v", unit.ConfigID, err)
				}
				// proceed with other ad units
				continue
			}
		}

		if glog.V(2) {
			glog.Infof("Ad unit %s has %d bidders for %d sizes", unit.Code, len(bidders), len(unit.Sizes))
		}

		mtypes := ParseMediaTypes(unit.MediaTypes)
		for _, b := range bidders {
			var bidder *PBSBidder
			for _, pb := range pbsReq.Bidders {
				if pb.BidderCode == b.BidderCode {
					bidder = pb
				}
			}

			if bidder == nil {
				bidder = &PBSBidder{BidderCode: b.BidderCode}
				pbsReq.Bidders = append(pbsReq.Bidders, bidder)
			}
			if b.BidID == "" {
				b.BidID = fmt.Sprintf("%d", rand.Int63())
			}

			pau := PBSAdUnit{
				Sizes:      unit.Sizes,
				TopFrame:   unit.TopFrame,
				Code:       unit.Code,
				Instl:      unit.Instl,
				Params:     b.Params,
				BidID:      b.BidID,
				MediaTypes: mtypes,
				Video:      unit.Video,
			}

			bidder.AdUnits = append(bidder.AdUnits, pau)
		}
	}

	return pbsReq, nil
}

func (req PBSRequest) Elapsed() int {
	return int(time.Since(req.Start) / 1000000)
}

func (p PBSRequest) String() string {
	b, _ := json.MarshalIndent(p, "", "    ")
	return string(b)
}

// parses the "Regs.ext.gdpr" from the request, if it exists. Otherwise returns an empty string.
func (req *PBSRequest) ParseGDPR() string {
	if req == nil || req.Regs == nil || len(req.Regs.Ext) == 0 {
		return ""
	}
	val, err := jsonparser.GetInt(req.Regs.Ext, "gdpr")
	if err != nil {
		return ""
	}
	gdpr := strconv.Itoa(int(val))

	return gdpr
}

// parses the "User.ext.consent" from the request, if it exists. Otherwise returns an empty string.
func (req *PBSRequest) ParseConsent() string {
	if req == nil || req.User == nil {
		return ""
	}
	return parseString(req.User.Ext, "consent")
}

func parseString(data []byte, key string) string {
	if len(data) == 0 {
		return ""
	}
	val, err := jsonparser.GetString(data, key)
	if err != nil {
		return ""
	}
	return val
}
