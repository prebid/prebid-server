package models

import (
	"encoding/json"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb3"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type BidExt struct {
	openrtb_ext.ExtBid

	ErrorCode               int                   `json:"errorCode,omitempty"`
	ErrorMsg                string                `json:"errorMessage,omitempty"`
	RefreshInterval         int                   `json:"refreshInterval,omitempty"`
	CreativeType            string                `json:"crtype,omitempty"`
	Summary                 []Summary             `json:"summary,omitempty"`
	SKAdnetwork             json.RawMessage       `json:"skadn,omitempty"`
	Video                   *ExtBidVideo          `json:"video,omitempty"`
	Banner                  *ExtBidBanner         `json:"banner,omitempty"`
	DspId                   int                   `json:"dspid,omitempty"`
	Winner                  int                   `json:"winner,omitempty"`
	NetECPM                 float64               `json:"netecpm,omitempty"`
	OriginalBidCPM          float64               `json:"origbidcpm,omitempty"`
	OriginalBidCur          string                `json:"origbidcur,omitempty"`
	OriginalBidCPMUSD       float64               `json:"origbidcpmusd,omitempty"`
	Nbr                     *openrtb3.NoBidReason `json:"-"` // Reason for not bidding
	Fsc                     int                   `json:"fsc,omitempty"`
	AdPod                   *AdpodBidExt          `json:"adpod,omitempty"`
	MultiBidMultiFloorValue float64               `json:"-"`
	InBannerVideo           bool                  `json:"ibv,omitempty"`
	ClickTrackers           []string              `json:"clicktrackers,omitempty"`
	OWSDK                   map[string]any        `json:"owsdk,omitempty"`
	Act                     int                   `json:"act,omitempty"`
	// BidExpEnf is partner bid.ext.bidexp_enf (parsed into BidCtx for impression tracker bexpef; not echoed on OW bid.ext).
	BidExpEnf int `json:"bidexp_enf,omitempty"`
}

type AdpodBidExt struct {
	IsAdpodBid bool              `json:"isAdpodBid,omitempty"`
	Targeting  map[string]string `json:"targeting,omitempty"`
	Debug      AdpodDebug        `json:"debug,omitempty"`
}

type AdpodDebug struct {
	Targeting map[string]string
}

// ExtBidVideo defines the contract for bidresponse.seatbid.bid[i].ext.video
type ExtBidVideo struct {
	MinDuration    int64                      `json:"minduration,omitempty"`    // Minimum video ad duration in seconds.
	MaxDuration    int64                      `json:"maxduration,omitempty"`    // Maximum video ad duration in seconds.
	Skip           *int8                      `json:"skip,omitempty"`           // Indicates if the player will allow the video to be skipped,where 0 = no, 1 = yes.
	SkipMin        int64                      `json:"skipmin,omitempty"`        // Videos of total duration greater than this number of seconds can be skippable; only applicable if the ad is skippable.
	SkipAfter      int64                      `json:"skipafter,omitempty"`      // Number of seconds a video must play before skipping is enabled; only applicable if the ad is skippable.
	BAttr          []adcom1.CreativeAttribute `json:"battr,omitempty"`          // Blocked creative attributes
	PlaybackMethod []adcom1.PlaybackMethod    `json:"playbackmethod,omitempty"` // Allowed playback methods
	ClientConfig   json.RawMessage            `json:"clientconfig,omitempty"`
}

// ExtBidBanner defines the contract for bidresponse.seatbid.bid[i].ext.banner
type ExtBidBanner struct {
	ClientConfig json.RawMessage `json:"clientconfig,omitempty"`
}

// ExtBidPrebidCache defines the contract for  bidresponse.seatbid.bid[i].ext.prebid.cache
type ExtBidPrebidCache struct {
	Key string `json:"key"`
	Url string `json:"url"`
}

// Prebid Response Ext with DspId
type OWExt struct {
	openrtb_ext.ExtOWBid
	DspId int `json:"dspid,omitempty"`

	OriginalBidCPM    float64 `json:"origbidcpm,omitempty"`
	OriginalBidCur    string  `json:"origbidcur,omitempty"`
	OriginalBidCPMUSD float64 `json:"origbidcpmusd,omitempty"`
}

// ExtBidderMessage defines an error object to be returned, consiting of a machine readable error code, and a human readable error message string.
type ExtBidderMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// BidType describes the allowed values for bidresponse.seatbid.bid[i].ext.prebid.type
type BidType string

// Targeting map of string of strings
type Targeting map[string]string

type Summary struct {
	VastTagID    string  `json:"vastTagID,omitempty"`
	Bidder       string  `json:"bidder,omitempty"`
	Bid          float64 `json:"bid,omitempty"`
	ErrorCode    int     `json:"errorCode,omitempty"`
	ErrorMessage string  `json:"errorMessage,omitempty"`
	Width        int     `json:"width,omitempty"`
	Height       int     `json:"height,omitempty"`
	Regex        string  `json:"regex,omitempty"`
}
