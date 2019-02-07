package exchange

import (
	"encoding/json"
	"strings"

	"github.com/mxmCherry/openrtb"
)

// ExtractGDPR will pull the gdpr flag from an openrtb request
func extractGDPR(bidRequest *openrtb.BidRequest, usersyncIfAmbiguous bool) (gdpr int) {
	var re regsExt
	var err error
	if bidRequest.Regs != nil {
		err = json.Unmarshal(bidRequest.Regs.Ext, &re)
	}
	if re.GDPR == nil || err != nil {
		if usersyncIfAmbiguous {
			gdpr = 0
		} else {
			gdpr = 1
		}
	} else {
		gdpr = *re.GDPR
	}
	return
}

// ExtractConsent will pull the consent string from an openrtb request
func extractConsent(bidRequest *openrtb.BidRequest) (consent string) {
	var ue userExt
	var err error
	if bidRequest.User != nil {
		err = json.Unmarshal(bidRequest.User.Ext, &ue)
	}
	if err != nil {
		return
	}
	consent = ue.Consent
	return
}

type userExt struct {
	Consent string `json:"consent,omitempty"`
}

type regsExt struct {
	GDPR *int `json:"gdpr,omitempty"`
}

// cleanPI removes IP address last byte, device ID, buyer ID, and rounds off latitude/longitude
func cleanPI(bidRequest *openrtb.BidRequest, isAMP bool) {
	if bidRequest.User != nil {
		// Need to duplicate pointer objects
		user := *bidRequest.User
		bidRequest.User = &user
		if isAMP == false {
			bidRequest.User.BuyerUID = ""
		}
		bidRequest.User.Geo = cleanGeo(bidRequest.User.Geo)
	}
	if bidRequest.Device != nil {
		// Need to duplicate pointer objects
		device := *bidRequest.Device
		bidRequest.Device = &device
		bidRequest.Device.DIDMD5 = ""
		bidRequest.Device.DIDSHA1 = ""
		bidRequest.Device.DPIDMD5 = ""
		bidRequest.Device.DPIDSHA1 = ""
		bidRequest.Device.IP = cleanIP(bidRequest.Device.IP)
		bidRequest.Device.IPv6 = cleanIPv6(bidRequest.Device.IPv6)
		bidRequest.Device.Geo = cleanGeo(bidRequest.Device.Geo)
	}
}

// Zero the last byte of an IP address
func cleanIP(fullIP string) string {
	i := strings.LastIndex(fullIP, ".")
	if i == -1 {
		return ""
	}
	return fullIP[0:i] + ".0"
}

// Zero the last two bytes of an IPv6 address
func cleanIPv6(fullIP string) string {
	i := strings.LastIndex(fullIP, ":")
	if i == -1 {
		return ""
	}
	return fullIP[0:i] + ":0000"
}

// Return a cleaned Geo object pointer (round off the latitude/longitude)
func cleanGeo(geo *openrtb.Geo) *openrtb.Geo {
	if geo == nil {
		return nil
	}
	newGeo := *geo
	newGeo.Lat = float64(int(geo.Lat*100.0+0.5)) / 100.0
	newGeo.Lon = float64(int(geo.Lon*100.0+0.5)) / 100.0
	return &newGeo
}
