// Package model defines VAST XML data structures for CTV ad processing.
package model

// VastAd represents a parsed VAST ad with its components.
// This is a higher-level domain object; for XML marshaling use the Vast struct.
type VastAd struct {
	// ID is the unique identifier for this ad.
	ID string
	// AdSystem identifies the ad server that returned the ad.
	AdSystem string
	// AdTitle is the common name of the ad.
	AdTitle string
	// Description is a longer description of the ad.
	Description string
	// Advertiser is the name of the advertiser.
	Advertiser string
	// DurationSec is the duration of the creative in seconds.
	DurationSec int
	// ErrorURLs contains error tracking URLs.
	ErrorURLs []string
	// ImpressionURLs contains impression tracking URLs.
	ImpressionURLs []string
	// Sequence indicates the position in an ad pod.
	Sequence int
	// RawVAST contains the original VAST XML if preserved.
	RawVAST []byte
}

