package openrtb_ext

import "encoding/json"

type BidResponseVideo struct {
	AdPods []*AdPod        `json:"adPods"`
	Ext    json.RawMessage `json:"ext"`
}

type AdPod struct {
	PodId     int64            `json:"podid"`
	Targeting []VideoTargeting `json:"targeting"`
	Errors    []string         `json:"errors"`
}

type VideoTargeting struct {
	Hb_pb         string `json:"hb_pb"`
	Hb_pb_cat_dur string `json:"hb_pb_cat_dur"`
	Hb_cache_id   string `json:"hb_cache_id"`
}
