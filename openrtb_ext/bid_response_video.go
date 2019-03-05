package openrtb_ext

type BidResponseVideo struct {
	AdPods []AdPod `json:"adPods"`
}

type AdPod struct {
	PodId     int64            `json:"podid"`
	Targeting []VideoTargeting `json:"targeting"`
	Errors    VideoErrors      `json:"errors"`
}

type VideoTargeting struct {
	Hb_pb         string `json:"hb_pb"`
	Hb_pb_cat_dur string `json:"hb_pb_cat_dur"`
	Hb_cache_id   string `json:"hb_cache_id"`
}

type VideoErrors struct {
	Openx []OpenxError `json:"openx"`
}

type OpenxError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
