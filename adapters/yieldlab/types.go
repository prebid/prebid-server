package yieldlab

import (
	"strconv"
	"time"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type bidResponse struct {
	ID         uint64       `json:"id"`
	Price      uint         `json:"price"`
	Advertiser string       `json:"advertiser"`
	Adsize     string       `json:"adsize"`
	Pid        uint64       `json:"pid"`
	Did        uint64       `json:"did"`
	Pvid       string       `json:"pvid"`
	DSA        *dsaResponse `json:"dsa,omitempty"`
}

// dsaResponse defines Digital Service Act (DSA) parameters from Yieldlab yieldprobe response.
type dsaResponse struct {
	Behalf       string            `json:"behalf,omitempty"`
	Paid         string            `json:"paid,omitempty"`
	Adrender     *int              `json:"adrender,omitempty"`
	Transparency []dsaTransparency `json:"transparency,omitempty"`
}

// openRTBExtRegsWithDSA defines the contract for bidrequest.regs.ext with the missing DSA property.
//
// The openrtb_ext.ExtRegs needs to be extended on yieldlab adapter level until DSA has been implemented
// by the prebid server team (https://github.com/prebid/prebid-server/issues/3424).
type openRTBExtRegsWithDSA struct {
	openrtb_ext.ExtRegs
	DSA *dsaRequest `json:"dsa,omitempty"`
}

// responseExtWithDSA defines seatbid.bid.ext with the DSA object.
type responseExtWithDSA struct {
	DSA dsaResponse `json:"dsa"`
}

// dsaRequest defines Digital Service Act (DSA) parameter
// as specified by the OpenRTB 2.X DSA Transparency community extension.
//
// Should rather come from openrtb_ext package but will be defined here until DSA has been
// implemented by the prebid server team (https://github.com/prebid/prebid-server/issues/3424).
type dsaRequest struct {
	Required     *int              `json:"dsarequired"`
	PubRender    *int              `json:"pubrender"`
	DataToPub    *int              `json:"datatopub"`
	Transparency []dsaTransparency `json:"transparency"`
}

// dsaTransparency Digital Service Act (DSA) transparency object
type dsaTransparency struct {
	Domain string `json:"domain,omitempty"`
	Params []int  `json:"dsaparams,omitempty"`
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
