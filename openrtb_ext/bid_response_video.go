package openrtb_ext

import "encoding/json"

type BidResponseVideo struct {
	AdPods []*AdPod        `json:"adPods"`
	Ext    json.RawMessage `json:"ext,omitempty"`
}

type AdPod struct {
	PodId     int64            `json:"podid"`
	Targeting []VideoTargeting `json:"targeting"`
	Errors    []string         `json:"errors"`
}

type VideoTargeting struct {
	HbPb       string `json:"hb_pb,omitempty"`
	HbPbCatDur string `json:"hb_pb_cat_dur,omitempty"`
	HbCacheID  string `json:"hb_cache_id,omitempty"`
	HbDeal     string `json:"hb_deal,omitempty"`
}
