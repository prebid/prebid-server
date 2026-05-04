package doohimpressionvalue

import "github.com/prebid/openrtb/v20/adcom1"

type lookupKey struct {
	AccountID string `json:"-"`
	Path      string `json:"path"`
	Key       string `json:"key"`
}

type impressionValue struct {
	Path       string                                     `json:"path,omitempty"`
	Key        string                                     `json:"key,omitempty"`
	Multiplier float64                                    `json:"multiplier"`
	SourceType adcom1.DOOHMultiplierMeasurementSourceType `json:"sourcetype,omitempty"`
	Vendor     string                                     `json:"vendor,omitempty"`
}

func (lk lookupKey) cacheKey() string {
	return lk.AccountID + "\x1f" + lk.Path + "\x1f" + lk.Key
}
