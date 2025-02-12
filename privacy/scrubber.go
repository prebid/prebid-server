package privacy

import (
	"encoding/json"
	"net"

	"github.com/prebid/prebid-server/v3/util/jsonutil"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/iputil"
)

type IPConf struct {
	IPV6 config.IPv6
	IPV4 config.IPv4
}

func scrubDeviceIDs(reqWrapper *openrtb_ext.RequestWrapper) {
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

func scrubUserIDs(reqWrapper *openrtb_ext.RequestWrapper) {
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

func scrubUserDemographics(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.User != nil {
		reqWrapper.User.BuyerUID = ""
		reqWrapper.User.ID = ""
		reqWrapper.User.Yob = 0
		reqWrapper.User.Gender = ""
	}
}

func scrubUserExt(reqWrapper *openrtb_ext.RequestWrapper, fieldName string) error {
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

func ScrubEIDs(reqWrapper *openrtb_ext.RequestWrapper) error {
	//transmitEids removes user.eids and user.ext.eids
	if reqWrapper.User != nil {
		reqWrapper.User.EIDs = nil
	}
	return scrubUserExt(reqWrapper, "eids")
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

func scrubGEO(reqWrapper *openrtb_ext.RequestWrapper) {
	//round user's geographic location by rounding off IP address and lat/lng data.
	//this applies to both device.geo and user.geo
	if reqWrapper.User != nil && reqWrapper.User.Geo != nil {
		reqWrapper.User.Geo = scrubGeoPrecision(reqWrapper.User.Geo)
	}

	if reqWrapper.Device != nil && reqWrapper.Device.Geo != nil {
		reqWrapper.Device.Geo = scrubGeoPrecision(reqWrapper.Device.Geo)
	}
}

func scrubGeoFull(reqWrapper *openrtb_ext.RequestWrapper) {
	if reqWrapper.User != nil && reqWrapper.User.Geo != nil {
		reqWrapper.User.Geo = &openrtb2.Geo{}
	}
	if reqWrapper.Device != nil && reqWrapper.Device.Geo != nil {
		reqWrapper.Device.Geo = &openrtb2.Geo{}
	}

}

func scrubDeviceIP(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf) {
	if reqWrapper.Device != nil {
		reqWrapper.Device.IP = scrubIP(reqWrapper.Device.IP, ipConf.IPV4.AnonKeepBits, iputil.IPv4BitSize)
		reqWrapper.Device.IPv6 = scrubIP(reqWrapper.Device.IPv6, ipConf.IPV6.AnonKeepBits, iputil.IPv6BitSize)
	}
}

func ScrubDeviceIDsIPsUserDemoExt(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf, fieldName string, scrubFullGeo bool) {
	scrubDeviceIDs(reqWrapper)
	scrubDeviceIP(reqWrapper, ipConf)
	scrubUserDemographics(reqWrapper)
	scrubUserExt(reqWrapper, fieldName)

	if scrubFullGeo {
		scrubGeoFull(reqWrapper)
	} else {
		scrubGEO(reqWrapper)
	}
}

func ScrubUserFPD(reqWrapper *openrtb_ext.RequestWrapper) {
	scrubDeviceIDs(reqWrapper)
	scrubUserIDs(reqWrapper)
	scrubUserExt(reqWrapper, "data")
	reqWrapper.User.EIDs = nil
}

func ScrubGdprID(reqWrapper *openrtb_ext.RequestWrapper) {
	scrubDeviceIDs(reqWrapper)
	scrubUserDemographics(reqWrapper)
	scrubUserExt(reqWrapper, "eids")
}

func ScrubGeoAndDeviceIP(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf) {
	scrubDeviceIP(reqWrapper, ipConf)
	scrubGEO(reqWrapper)
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

	if geoCopy.Lat != nil {
		lat := *geo.Lat
		lat = float64(int(lat*100.0+0.5)) / 100.0
		geoCopy.Lat = &lat
	}

	if geoCopy.Lon != nil {
		lon := *geo.Lon
		lon = float64(int(lon*100.0+0.5)) / 100.0
		geoCopy.Lon = &lon
	}

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
