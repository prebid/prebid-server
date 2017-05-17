package pbs

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/prebid/prebid-server/cache"
	"github.com/golang/glog"
	"github.com/prebid/openrtb"
	"golang.org/x/net/publicsuffix"
)

const MAX_BIDDERS = 8

type ConfigCache interface {
	LoadConfig(string) ([]Bids, error)
}

type Bids struct {
	BidderCode string          `json:"bidder"`
	BidID      string          `json:"bid_id"`
	Params     json.RawMessage `json:"params"`
}

type AdUnit struct {
	Code     string           `json:"code"`
	TopFrame int8             `json:"is_top_frame"`
	Sizes    []openrtb.Format `json:"sizes"`
	Bids     []Bids           `json:"bids"`
	ConfigID string           `json:"config_id"`
}

type PBSAdUnit struct {
	Sizes    []openrtb.Format
	TopFrame int8
	Code     string
	BidID    string
	Params   json.RawMessage
}

type PBSBidder struct {
	BidderCode   string         `json:"bidder"`
	AdUnitCode   string         `json:"ad_unit,omitempty"` // for index to dedup responses
	ResponseTime int            `json:"response_time_ms,omitempty"`
	NumBids      int            `json:"num_bids,omitempty"`
	Error        string         `json:"error,omitempty"`
	NoCookie     bool           `json:"no_cookie,omitempty"`
	NoBid        bool           `json:"no_bid,omitempty"`
	UsersyncInfo *UsersyncInfo  `json:"usersync,omitempty"`
	Debug        []*BidderDebug `json:"debug,omitempty"`

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

type PBSRequest struct {
	AccountID     string          `json:"account_id"`
	Tid           string          `json:"tid"`
	CacheMarkup   int8            `json:"cache_markup"`
	Secure        int8            `json:"secure"`
	TimeoutMillis uint64          `json:"timeout_millis"`
	AdUnits       []AdUnit        `json:"ad_units"`
	IsDebug       bool            `json:"is_debug"`
	App           *openrtb.App    `json:"app"`
	Device        *openrtb.Device `json:"device"`

	// internal
	Bidders []*PBSBidder      `json:"-"`
	UserIDs map[string]string `json:"-"`
	Url     string            `json:"-"`
	Domain  string            `json:"-"`
	Start   time.Time
}

func getConfig(cache cache.Cache, id string) ([]Bids, error) {
	conf, err := cache.Config().Get(id)
	if err != nil {
		return nil, err
	}
	var bids = make([]Bids, 0)
	if err := json.Unmarshal([]byte(conf), &bids); err != nil {
		return nil, err
	}
	return bids, nil
}

func ParsePBSRequest(r *http.Request, cache cache.Cache, defaultTimeoutMS uint64) (*PBSRequest, error) {
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

	if pbsReq.TimeoutMillis == 0 || pbsReq.TimeoutMillis > 2000 {
		pbsReq.TimeoutMillis = defaultTimeoutMS
	}

	if pbsReq.Device == nil {
		pbsReq.Device = &openrtb.Device{}
	}

	// use client-side data for web requests
	if pbsReq.App == nil {
		pc := ParseUIDCookie(r)
		pbsReq.UserIDs = pc.UIDs

		// this would be for the shared adnxs.com domain
		if anid, err := r.Cookie("uuid2"); err == nil {
			pbsReq.UserIDs["adnxs"] = anid.Value
		}

		pbsReq.Device.UA = r.Header.Get("User-Agent")

		if r.Header.Get("X-Real-Ip") != "" {
			pbsReq.Device.IP = r.Header.Get("X-Real-Ip")
		} else {
			ip, _, uerr := net.SplitHostPort(r.RemoteAddr)
			if uerr == nil && net.ParseIP(ip) != nil {
				pbsReq.Device.IP = ip
			}
		}
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

		// all domains must be in the whitelist
		if _, err := cache.Domains().Get(pbsReq.Domain); err != nil {
			return nil, fmt.Errorf("Invalid URL %s", pbsReq.Domain)
		}
	} else {
		if _, err = cache.Apps().Get(pbsReq.App.Bundle); err != nil {
			return nil, fmt.Errorf("Invalid app bundle %s", pbsReq.App.Bundle)
		}
	}

	if r.FormValue("debug") == "1" {
		pbsReq.IsDebug = true
	}

	if pbsReq.Secure == 0 {
		if r.Header.Get("X-Forwarded-Proto") == "https" {
			pbsReq.Secure = 1
		} else if r.TLS != nil {
			pbsReq.Secure = 1
		}
	}

	pbsReq.Bidders = make([]*PBSBidder, 0, MAX_BIDDERS)

	for _, unit := range pbsReq.AdUnits {
		bidders := unit.Bids
		if unit.ConfigID != "" {
			bidders, err = getConfig(cache, unit.ConfigID)
			if err != nil {
				// proceed with other ad units
				glog.Infof("Unable to load config '%s': %v", unit.ConfigID, err)
				continue
			}
		}

		if glog.V(2) {
			glog.Infof("Ad unit %s has %d bidders for %d sizes", unit.Code, len(bidders), len(unit.Sizes))
		}

		for _, b := range bidders {
			var bidder *PBSBidder
			// index requires a different request for each ad unit
			if b.BidderCode != "indexExchange" {
				for _, pb := range pbsReq.Bidders {
					if pb.BidderCode == b.BidderCode {
						bidder = pb
					}
				}
			}
			if bidder == nil {
				bidder = &PBSBidder{BidderCode: b.BidderCode}
				if b.BidderCode == "indexExchange" {
					bidder.AdUnitCode = unit.Code
				}
				pbsReq.Bidders = append(pbsReq.Bidders, bidder)
			}

			bidder.AdUnits = append(bidder.AdUnits, PBSAdUnit{
				Sizes:    unit.Sizes,
				TopFrame: unit.TopFrame,
				Code:     unit.Code,
				Params:   b.Params,
				BidID:    b.BidID,
			})
		}
	}

	return pbsReq, nil
}

func (req PBSRequest) Elapsed() int {
	return int(time.Since(req.Start) / 1000000)
}

func (req PBSRequest) GetUserID(BidderCode string) string {
	if uid, ok := req.UserIDs[BidderCode]; ok {
		return uid
	}
	return ""
}

func (p PBSRequest) String() string {
	b, _ := json.MarshalIndent(p, "", "    ")
	return string(b)
}
