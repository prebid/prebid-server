package privacy

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v2/util/jsonutil"
	"net"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/prebid/prebid-server/v2/util/iputil"
)

type IPConf struct {
	IPV6 config.IPv6
	IPV4 config.IPv4
}

func ScrubDeviceIDs(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.Device != nil {
		reqWrapper.Device.DIDMD5 = ""
		reqWrapper.Device.DIDSHA1 = ""
		reqWrapper.Device.DPIDMD5 = ""
		reqWrapper.Device.DPIDSHA1 = ""
		reqWrapper.Device.IFA = ""
		reqWrapper.Device.MACMD5 = ""
		reqWrapper.Device.MACSHA1 = ""
	}
}

func ScrubUserIDs(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.User != nil {
		reqWrapper.User.Data = nil
		reqWrapper.User.ID = ""
		reqWrapper.User.BuyerUID = ""
		reqWrapper.User.Yob = 0
		reqWrapper.User.Gender = ""
		reqWrapper.User.Keywords = ""
		reqWrapper.User.KwArray = nil
	}
}

func ScrubUserDemographics(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.User != nil {
		reqWrapper.User.BuyerUID = ""
		reqWrapper.User.ID = ""
		reqWrapper.User.Yob = 0
		reqWrapper.User.Gender = ""
	}
}

func ScrubUserExt(reqWrapper *openrtb_ext.RequestWrapper, fieldName string) error {
	if reqWrapper.User != nil {
		userExt, err := reqWrapper.GetUserExt()
		if err != nil {
			return err
		}
		ext := userExt.GetExt()
		_, hasField := ext[fieldName]
		if hasField {
			delete(ext, fieldName)
			userExt.SetExt(ext)
		}
	}
	return nil
}

func ScrubEids(reqWrapper *openrtb_ext.RequestWrapper) error {
	//transmitEids covers user.eids and user.ext.eids
	if reqWrapper.User != nil {
		reqWrapper.User.EIDs = nil
	}
	return ScrubUserExt(reqWrapper, "eids")
}

func ScrubTID(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.Source != nil {
		reqWrapper.Source.TID = ""
	}
	impWrapper := reqWrapper.GetImp()
	for ind, imp := range impWrapper {
		impExt := scrubExtIDs(imp.Ext, "tid")
		impWrapper[ind].Ext = impExt
	}
	reqWrapper.SetImp(impWrapper)
}

func ScrubGEO(reqWrapper *openrtb_ext.RequestWrapper) {
	//round user's geographic location by rounding off IP address and lat/lng data.
	//this applies to both device.geo and user.geo
	if reqWrapper.User != nil && reqWrapper.User.Geo != nil {
		reqWrapper.User.Geo = scrubGeoPrecision(reqWrapper.User.Geo)
	}

	if reqWrapper.Device != nil {
		if reqWrapper.Device.Geo != nil {
			reqWrapper.Device.Geo = scrubGeoPrecision(reqWrapper.Device.Geo)
		}
	}
}

func ScrubGeoFull(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.User != nil && reqWrapper.User.Geo != nil {
		reqWrapper.User.Geo = &openrtb2.Geo{}
	}
	if reqWrapper.Device != nil && reqWrapper.Device.Geo != nil {
		reqWrapper.Device.Geo = &openrtb2.Geo{}
	}

}

func ScrubDeviceIP(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf) {
	if reqWrapper.Device != nil {
		reqWrapper.Device.IP = scrubIP(reqWrapper.Device.IP, ipConf.IPV4.AnonKeepBits, iputil.IPv4BitSize)
		reqWrapper.Device.IPv6 = scrubIP(reqWrapper.Device.IPv6, ipConf.IPV6.AnonKeepBits, iputil.IPv6BitSize)
	}
}

func ScrubDeviceIDsIPsUserDemoExt(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf, fieldName string, scrubFullGeo bool) {
	ScrubDeviceIDs(reqWrapper)
	ScrubDeviceIP(reqWrapper, ipConf)
	ScrubUserDemographics(reqWrapper)
	ScrubUserExt(reqWrapper, fieldName)

	if scrubFullGeo {
		ScrubGeoFull(reqWrapper)
	} else {
		ScrubGEO(reqWrapper)
	}
}

func ScrubUserFPD(reqWrapper *openrtb_ext.RequestWrapper) {
	ScrubDeviceIDs(reqWrapper)
	ScrubUserIDs(reqWrapper)
	ScrubUserExt(reqWrapper, "data")
	reqWrapper.User.EIDs = nil
}

func ScrubGdprID(reqWrapper *openrtb_ext.RequestWrapper) {
	ScrubDeviceIDs(reqWrapper)
	ScrubUserDemographics(reqWrapper)
	ScrubUserExt(reqWrapper, "eids")
}

func ScrubGeoAndDeviceIP(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf) {
	ScrubDeviceIP(reqWrapper, ipConf)
	ScrubGEO(reqWrapper)
}

func scrubIP(ip string, ones, bits int) string {
	if ip == "" {
		return ""
	}
	ipMask := net.CIDRMask(ones, bits)
	ipMasked := net.ParseIP(ip).Mask(ipMask)
	return ipMasked.String()
}

func scrubGeoPrecision(geo *openrtb2.Geo) *openrtb2.Geo {
	if geo == nil {
		return nil
	}

	geoCopy := *geo
	geoCopy.Lat = float64(int(geo.Lat*100.0+0.5)) / 100.0 // Round Latitude
	geoCopy.Lon = float64(int(geo.Lon*100.0+0.5)) / 100.0 // Round Longitude
	return &geoCopy
}

func scrubExtIDs(ext json.RawMessage, fieldName string) json.RawMessage {
	if len(ext) == 0 {
		return ext
	}

	var userExtParsed map[string]json.RawMessage
	err := jsonutil.Unmarshal(ext, &userExtParsed)
	if err != nil {
		return ext
	}

	_, hasField := userExtParsed[fieldName]
	if hasField {
		delete(userExtParsed, fieldName)
		result, err := jsonutil.Marshal(userExtParsed)
		if err == nil {
			return result
		}
	}

	return ext
}
