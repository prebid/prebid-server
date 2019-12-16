package privacy

import (
	"strings"

	"github.com/mxmCherry/openrtb"
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
	// ScrubStrategyUserNone does not remove user data.
	ScrubStrategyUserNone ScrubStrategyUser = iota

	// ScrubStrategyUserFull removes the user's buyer id, exchange id year of birth, and gender.
	ScrubStrategyUserFull

	// ScrubStrategyUserBuyerIDOnly removes the user's buyer id.
	ScrubStrategyUserBuyerIDOnly
)

// Scrubber removes PII from parts of an OpenRTB request.
type Scrubber interface {
	ScrubDevice(device *openrtb.Device, macAndIFA bool, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb.Device
	ScrubUser(user *openrtb.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb.User
}

type scrubber struct{}

// NewScrubber returns an OpenRTB scrubber.
func NewScrubber() Scrubber {
	return scrubber{}
}

func (scrubber) ScrubDevice(device *openrtb.Device, macAndIFA bool, ipv6 ScrubStrategyIPV6, geo ScrubStrategyGeo) *openrtb.Device {
	if device == nil {
		return nil
	}

	deviceCopy := *device

	deviceCopy.DIDMD5 = ""
	deviceCopy.DIDSHA1 = ""
	deviceCopy.DPIDMD5 = ""
	deviceCopy.DPIDSHA1 = ""
	deviceCopy.IP = scrubIPV4(device.IP)

	if macAndIFA {
		deviceCopy.MACSHA1 = ""
		deviceCopy.MACMD5 = ""
		deviceCopy.IFA = ""
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

func (scrubber) ScrubUser(user *openrtb.User, strategy ScrubStrategyUser, geo ScrubStrategyGeo) *openrtb.User {
	if user == nil {
		return nil
	}

	userCopy := *user

	switch strategy {
	case ScrubStrategyUserFull:
		userCopy.BuyerUID = ""
		userCopy.ID = ""
		userCopy.Yob = 0
		userCopy.Gender = ""
	case ScrubStrategyUserBuyerIDOnly:
		userCopy.BuyerUID = ""
	}

	switch geo {
	case ScrubStrategyGeoFull:
		userCopy.Geo = scrubGeoFull(user.Geo)
	case ScrubStrategyGeoReducedPrecision:
		userCopy.Geo = scrubGeoPrecision(user.Geo)
	}

	return &userCopy
}

func scrubIPV4(ip string) string {
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

func scrubGeoFull(geo *openrtb.Geo) *openrtb.Geo {
	if geo == nil {
		return nil
	}

	geoCopy := *geo
	geoCopy.Lat = 0
	geoCopy.Lon = 0
	geoCopy.Metro = ""
	geoCopy.City = ""
	geoCopy.ZIP = ""
	return &geoCopy
}

func scrubGeoPrecision(geo *openrtb.Geo) *openrtb.Geo {
	if geo == nil {
		return nil
	}

	geoCopy := *geo
	geoCopy.Lat = float64(int(geo.Lat*100.0+0.5)) / 100.0 // Round Latitude
	geoCopy.Lon = float64(int(geo.Lon*100.0+0.5)) / 100.0 // Round Longitude
	return &geoCopy
}
