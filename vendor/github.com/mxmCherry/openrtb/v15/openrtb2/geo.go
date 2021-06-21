package openrtb2

import "encoding/json"

// 3.2.19 Object: Geo
//
// This object encapsulates various methods for specifying a geographic location.
// When subordinate to a
// Device object, it indicates the location of the device which can also be interpreted as the user’s current location.
// When subordinate to a User object, it indicates the location of the user’s home base (i.e., not necessarily their current location).
//
// The lat/lon attributes should only be passed if they conform to the accuracy depicted in the type attribute.
// For example, the centroid of a geographic region such as postal code should not be passed.
type Geo struct {

	// Attribute:
	//   lat
	// Type:
	//   float
	// Description:
	//   Latitude from -90.0 to +90.0, where negative is south.
	Lat float64 `json:"lat,omitempty"`

	// Attribute:
	//   lon
	// Type:
	//   float
	// Description:
	//   Longitude from -180.0 to +180.0, where negative is west.
	Lon float64 `json:"lon,omitempty"`

	// Attribute:
	//   type
	// Type:
	//   integer
	// Description:
	//   Source of location data; recommended when passing
	//   lat/lon. Refer to List 5.20.
	Type LocationType `json:"type,omitempty"`

	// Attribute:
	//   accuracy
	// Type:
	//   integer
	// Description:
	//   Estimated location accuracy in meters; recommended when
	//   lat/lon are specified and derived from a device’s location
	//   services (i.e., type = 1). Note that this is the accuracy as
	//   reported from the device. Consult OS specific documentation
	//   (e.g., Android, iOS) for exact interpretation.
	Accuracy int64 `json:"accuracy,omitempty"`

	// Attribute:
	//   lastfix
	// Type:
	//   integer
	// Description:
	//   Number of seconds since this geolocation fix was established.
	//   Note that devices may cache location data across multiple
	//   fetches. Ideally, this value should be from the time the actual
	//   fix was taken.
	LastFix int64 `json:"lastfix,omitempty"`

	// Attribute:
	//   ipservice
	// Type:
	//   integer
	// Description:
	//   Service or provider used to determine geolocation from IP
	//   address if applicable (i.e., type = 2). Refer to List 5.23.
	IPService IPLocationService `json:"ipservice,omitempty"`

	// Attribute:
	//   country
	// Type:
	//   string
	// Description:
	//   Country code using ISO-3166-1-alpha-3.
	Country string `json:"country,omitempty"`

	// Attribute:
	//   region
	// Type:
	//   string
	// Description:
	//   Region code using ISO-3166-2; 2-letter state code if USA.
	Region string `json:"region,omitempty"`

	// Attribute:
	//   regionfips104
	// Type:
	//   string
	// Description:
	//   Region of a country using FIPS 10-4 notation. While OpenRTB
	//   supports this attribute, it has been withdrawn by NIST in 2008.
	RegionFIPS104 string `json:"regionfips104,omitempty"`

	// Attribute:
	//   metro
	// Type:
	//   string
	// Description:
	//   Google metro code; similar to but not exactly Nielsen DMAs.
	//   See Appendix A for a link to the codes.
	Metro string `json:"metro,omitempty"`

	// Attribute:
	//   city
	// Type:
	//   string
	// Description:
	//   City using United Nations Code for Trade & Transport
	//   Locations. See Appendix A for a link to the codes.
	City string `json:"city,omitempty"`

	// Attribute:
	//   zip
	// Type:
	//   string
	// Description:
	//   Zip or postal code.
	ZIP string `json:"zip,omitempty"`

	// Attribute:
	//   utcoffset
	// Type:
	//   integer
	// Description:
	//   Local time as the number +/- of minutes from UTC.
	UTCOffset int64 `json:"utcoffset,omitempty"`

	// Attribute:
	//   ext
	// Type:
	//   object
	// Description:
	//   Placeholder for exchange-specific extensions to OpenRTB.
	Ext json.RawMessage `json:"ext,omitempty"`
}
