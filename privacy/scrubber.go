package privacy

import (
	"encoding/json"
	"strings"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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

	// ScrubStrategyUserID removes the user's buyer id.
	ScrubStrategyUserID
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
	ScrubDevice(device *openrtb2.Device, id ScrubStrategyDeviceID, ipv4 ScrubStrategyIPV4, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb2.Device
	ScrubUser(user *openrtb2.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb2.User
}

type scrubber struct{}

// NewScrubber returns an OpenRTB scrubber.
func NewScrubber() Scrubber {
	return scrubber{}
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

	switch strategy {
	case ScrubStrategyUserIDAndDemographic:
		userCopy.BuyerUID = ""
		userCopy.ID = ""
		userCopy.Ext = scrubUserExtIDs(userCopy.Ext)
		userCopy.Yob = 0
		userCopy.Gender = ""
	case ScrubStrategyUserID:
		userCopy.BuyerUID = ""
		userCopy.ID = ""
		userCopy.Ext = scrubUserExtIDs(userCopy.Ext)
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

func scrubUserExtIDs(userExt json.RawMessage) json.RawMessage {
	if len(userExt) == 0 {
		return userExt
	}

	var userExtParsed map[string]json.RawMessage
	err := json.Unmarshal(userExt, &userExtParsed)
	if err != nil {
		return userExt
	}

	_, hasEids := userExtParsed["eids"]
	if hasEids {
		delete(userExtParsed, "eids")
		result, err := json.Marshal(userExtParsed)
		if err == nil {
			return result
		}
	}

	return userExt
}
