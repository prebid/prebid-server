package openrtb

// 5.23 IP Location Services
//
// Services and/or vendors used for resolving IP addresses to geolocations.
type IPLocationService int8

const (
	IPLocationServiceIP2location IPLocationService = 1 // ip2location
	IPLocationServiceNeustar     IPLocationService = 2 // Neustar (Quova)
	IPLocationServiceMaxMind     IPLocationService = 3 // MaxMind
	IPLocationServiceNetAcuity   IPLocationService = 4 // NetAcuity (Digital Element)
)
