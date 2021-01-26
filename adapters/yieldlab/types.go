package yieldlab

import (
	"strconv"
	"time"
)

type bidResponse struct {
	ID         uint64 `json:"id"`
	Price      uint   `json:"price"`
	Advertiser string `json:"advertiser"`
	Adsize     string `json:"adsize"`
	Pid        uint64 `json:"pid"`
	Did        uint64 `json:"did"`
	Pvid       string `json:"pvid"`
}

type cacheBuster func() string

type weekGenerator func() string

var defaultCacheBuster cacheBuster = func() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

var defaultWeekGenerator weekGenerator = func() string {
	_, week := time.Now().ISOWeek()
	return strconv.Itoa(week)
}
