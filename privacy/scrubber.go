package privacy

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/util/iputil"
)

// ScrubStrategyIPV4 defines the approach to scrub PII from an IPV4 address.
type ScrubStrategyIPV4 int

const (
	// ScrubStrategyIPV4None does not remove any part of an IPV4 address.
	ScrubStrategyIPV4None ScrubStrategyIPV4 = iota

	// ScrubStrategyIPV4Subnet zeroes out the last 8 bits of an IPV4 address.
	ScrubStrategyIPV4Subnet
)

// ScrubStrategyIPV6 defines the approach to scrub PII from an IPV6 address.
type ScrubStrategyIPV6 int

const (
	// ScrubStrategyIPV6None does not remove any part of an IPV6 address.
	ScrubStrategyIPV6None ScrubStrategyIPV6 = iota

	// ScrubStrategyIPV6Subnet zeroes out the last 16 bits of an IPV6 sub net address.
	ScrubStrategyIPV6Subnet
)

// ScrubStrategyGeo defines the approach to scrub PII from geographical data.
type ScrubStrategyGeo int

const (
	// ScrubStrategyGeoNone does not remove any geographical data.
	ScrubStrategyGeoNone ScrubStrategyGeo = iota

	// ScrubStrategyGeoFull removes all geographical data.
	ScrubStrategyGeoFull

	// ScrubStrategyGeoReducedPrecision anonymizes geographical data with rounding.
	ScrubStrategyGeoReducedPrecision
)

// ScrubStrategyUser defines the approach to scrub PII from user data.
type ScrubStrategyUser int

const (
	// ScrubStrategyUserNone does not remove non-location data.
	ScrubStrategyUserNone ScrubStrategyUser = iota

	// ScrubStrategyUserIDAndDemographic removes the user's buyer id, exchange id year of birth, and gender.
	ScrubStrategyUserIDAndDemographic
)

// ScrubStrategyDeviceID defines the approach to remove hardware id and device id data.
type ScrubStrategyDeviceID int

const (
	// ScrubStrategyDeviceIDNone does not remove hardware id and device id data.
	ScrubStrategyDeviceIDNone ScrubStrategyDeviceID = iota

	// ScrubStrategyDeviceIDAll removes all hardware and device id data (ifa, mac hashes device id hashes)
	ScrubStrategyDeviceIDAll
)

// Scrubber removes PII from parts of an OpenRTB request.
type Scrubber interface {
	ScrubRequest(bidRequest *openrtb2.BidRequest, enforcement Enforcement) *openrtb2.BidRequest
	ScrubDevice(device *openrtb2.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb2.Device
	ScrubUser(user *openrtb2.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb2.User
}

type scrubber struct {
	ipV6 config.IPv6
	ipV4 config.IPv4
}

type IPConf struct {
	IPV6 config.IPv6
	IPV4 config.IPv4
}

// NewScrubber returns an OpenRTB scrubber.
func NewScrubber(ipV6 config.IPv6, ipV4 config.IPv4) Scrubber {
	return scrubber{
		ipV6: ipV6,
		ipV4: ipV4,
	}
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
	//do we need to copy imps?
	for ind, imp := range impWrapper {
		impExt := scrubExtIDs(imp.Ext, "tid")
		impWrapper[ind].Ext = impExt
	}
	reqWrapper.SetImp(impWrapper)
}

func ScrubGEO(reqWrapper *openrtb_ext.RequestWrapper, ipConf IPConf) {
	//round user's geographic location by rounding off IP address and lat/lng data.
	//this applies to both device.geo and user.geo
	if reqWrapper.User != nil && reqWrapper.User.Geo != nil {
		reqWrapper.User.Geo = scrubGeoPrecision(reqWrapper.User.Geo)
	}

	if reqWrapper.Device != nil {
		if reqWrapper.Device.Geo != nil {
			reqWrapper.Device.Geo = scrubGeoPrecision(reqWrapper.Device.Geo)
		}
		reqWrapper.Device.IP = scrubIP(reqWrapper.Device.IP, ipConf.IPV4.AnonKeepBits, iputil.IPv4BitSize)
		reqWrapper.Device.IPv6 = scrubIP(reqWrapper.Device.IPv6, ipConf.IPV6.AnonKeepBits, iputil.IPv6BitSize)
	}
}

func (s scrubber) ScrubDevice(device *openrtb2.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb2.Device {
	if device == nil {
		return nil
	}

	deviceCopy := *device

	switch id {
	case ScrubStrategyDeviceIDAll:
		deviceCopy.DIDMD5 = ""
		deviceCopy.DIDSHA1 = ""
		deviceCopy.DPIDMD5 = ""
		deviceCopy.DPIDSHA1 = ""
		deviceCopy.IFA = ""
		deviceCopy.MACMD5 = ""
		deviceCopy.MACSHA1 = ""
	}

	switch ipv4 {
	case ScrubStrategyIPV4Subnet:
		deviceCopy.IP = scrubIP(device.IP, s.ipV4.AnonKeepBits, iputil.IPv4BitSize)
	}

	switch ipv6 {
	case ScrubStrategyIPV6Subnet:
		deviceCopy.IPv6 = scrubIP(device.IPv6, s.ipV6.AnonKeepBits, iputil.IPv6BitSize)
	}

	switch geo {
	case ScrubStrategyGeoFull:
		deviceCopy.Geo = scrubGeoFull(device.Geo)
	case ScrubStrategyGeoReducedPrecision:
		deviceCopy.Geo = scrubGeoPrecision(device.Geo)
	}

	return &deviceCopy
}

func (scrubber) ScrubUser(user *openrtb2.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb2.User {
	if user == nil {
		return nil
	}

	userCopy := *user

	if strategy == ScrubStrategyUserIDAndDemographic {
		userCopy.BuyerUID = ""
		userCopy.ID = ""
		userCopy.Ext = scrubExtIDs(userCopy.Ext, "eids")
		userCopy.Yob = 0
		userCopy.Gender = ""
	}

	switch geo {
	case ScrubStrategyGeoFull:
		userCopy.Geo = scrubGeoFull(user.Geo)
	case ScrubStrategyGeoReducedPrecision:
		userCopy.Geo = scrubGeoPrecision(user.Geo)
	}

	return &userCopy
}

func scrubIP(ip string, ones, bits int) string {
	if ip == "" {
		return ""
	}
	ipMask := net.CIDRMask(ones, bits)
	ipMasked := net.ParseIP(ip).Mask(ipMask)
	return ipMasked.String()
}

func scrubGeoFull(geo *openrtb2.Geo) *openrtb2.Geo {
	if geo == nil {
		return nil
	}

	return &openrtb2.Geo{}
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
	err := json.Unmarshal(ext, &userExtParsed)
	if err != nil {
		return ext
	}

	_, hasField := userExtParsed[fieldName]
	if hasField {
		delete(userExtParsed, fieldName)
		result, err := json.Marshal(userExtParsed)
		if err == nil {
			return result
		}
	}

	return ext
}
