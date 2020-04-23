package openrtb

// 5.20 Location Type
//
// Options to indicate how the geographic information was determined.
type LocationType int8

const (
	LocationTypeGPSLocationServices LocationType = 1 // GPS/Location Services
	LocationTypeIPAddress           LocationType = 2 // IP Address
	LocationTypeUserProvided        LocationType = 3 // User provided (e.g., registration data)
)
