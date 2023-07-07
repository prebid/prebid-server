package privacy

import (
	"encoding/json"
	"strings"

	"github.com/prebid/openrtb/v19/openrtb2"
)

// ScrubStrategyIPV4 defines the approach to scrub PII from an IPV4 address.
type ScrubStrategyIPV4 int

const (
	// ScrubStrategyIPV4None does not remove any part of an IPV4 address.
	ScrubStrategyIPV4None ScrubStrategyIPV4 = iota

	// ScrubStrategyIPV4Lowest8 zeroes out the last 8 bits of an IPV4 address.
	ScrubStrategyIPV4Lowest8
)

// ScrubStrategyIPV6 defines the approach to scrub PII from an IPV6 address.
type ScrubStrategyIPV6 int

const (
	// ScrubStrategyIPV6None does not remove any part of an IPV6 address.
	ScrubStrategyIPV6None ScrubStrategyIPV6 = iota

	// ScrubStrategyIPV6Lowest16 zeroes out the last 16 bits of an IPV6 address.
	ScrubStrategyIPV6Lowest16

	// ScrubStrategyIPV6Lowest32 zeroes out the last 32 bits of an IPV6 address.
	ScrubStrategyIPV6Lowest32
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

type scrubber struct{}

// NewScrubber returns an OpenRTB scrubber.
func NewScrubber() Scrubber {
	return scrubber{}
}

func (scrubber) ScrubRequest(bidRequest *openrtb2.BidRequest, enforcement Enforcement) *openrtb2.BidRequest {
	if enforcement.UFPD {
		// transmitUfpd covers user.ext.data, user.data, user.id, user.buyeruid, user.yob, user.gender, user.keywords, user.kwarray
		// and device.{ifa, macsha1, macmd5, dpidsha1, dpidmd5, didsha1, didmd5}
		if bidRequest.Device != nil {
			bidRequest.Device.DIDMD5 = ""
			bidRequest.Device.DIDSHA1 = ""
			bidRequest.Device.DPIDMD5 = ""
			bidRequest.Device.DPIDSHA1 = ""
			bidRequest.Device.IFA = ""
			bidRequest.Device.MACMD5 = ""
			bidRequest.Device.MACSHA1 = ""
		}
		if bidRequest.User != nil {
			bidRequest.User.Data = nil
			bidRequest.User.Ext = scrubExtIDs(bidRequest.User.Ext, "data")
			bidRequest.User.ID = ""
			bidRequest.User.BuyerUID = ""
			bidRequest.User.Yob = 0
			bidRequest.User.Gender = ""
			bidRequest.User.Keywords = ""
			bidRequest.User.KwArray = nil
		}
	}
	if enforcement.Eids {
		//transmitEids covers user.eids and user.ext.eids
		if bidRequest.User != nil {
			bidRequest.User.EIDs = nil
			bidRequest.User.Ext = scrubExtIDs(bidRequest.User.Ext, "eids")
		}
	}

	if enforcement.TID {
		//remove source.tid and imp.ext.tid
		if bidRequest.Source != nil {
			bidRequest.Source.TID = ""
		}
		for ind, imp := range bidRequest.Imp {
			impExt := scrubExtIDs(imp.Ext, "tid")
			bidRequest.Imp[ind].Ext = impExt
		}
	}

	if enforcement.PreciseGeo {
		//round user's geographic location by rounding off IP address and lat/lng data.
		//this applies to both device.geo and user.geo
		if bidRequest.User != nil && bidRequest.User.Geo != nil {
			bidRequest.User.Geo = scrubGeoPrecision(bidRequest.User.Geo)
		}
		if bidRequest.Device != nil && bidRequest.Device.Geo != nil {
			bidRequest.Device.Geo = scrubGeoPrecision(bidRequest.Device.Geo)
		}
	}

	return bidRequest
}

func (scrubber) ScrubDevice(device *openrtb2.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb2.Device {
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
	case ScrubStrategyIPV4Lowest8:
		deviceCopy.IP = scrubIPV4Lowest8(device.IP)
	}

	switch ipv6 {
	case ScrubStrategyIPV6Lowest16:
		deviceCopy.IPv6 = scrubIPV6Lowest16Bits(device.IPv6)
	case ScrubStrategyIPV6Lowest32:
		deviceCopy.IPv6 = scrubIPV6Lowest32Bits(device.IPv6)
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

func scrubIPV4Lowest8(ip string) string {
	i := strings.LastIndex(ip, ".")
	if i == -1 {
		return ""
	}

	return ip[0:i] + ".0"
}

func scrubIPV6Lowest16Bits(ip string) string {
	ip = removeLowestIPV6Segment(ip)

	if ip != "" {
		ip += ":0"
	}

	return ip
}

func scrubIPV6Lowest32Bits(ip string) string {
	ip = removeLowestIPV6Segment(ip)
	ip = removeLowestIPV6Segment(ip)

	if ip != "" {
		ip += ":0:0"
	}

	return ip
}

func removeLowestIPV6Segment(ip string) string {
	i := strings.LastIndex(ip, ":")

	if i == -1 {
		return ""
	}

	return ip[0:i]
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

func scrubExtIDs(userExt json.RawMessage, fieldName string) json.RawMessage {
	if len(userExt) == 0 {
		return userExt
	}

	var userExtParsed map[string]json.RawMessage
	err := json.Unmarshal(userExt, &userExtParsed)
	if err != nil {
		return userExt
	}

	_, hasField := userExtParsed[fieldName]
	if hasField {
		delete(userExtParsed, fieldName)
		result, err := json.Marshal(userExtParsed)
		if err == nil {
			return result
		}
	}

	return userExt
}
