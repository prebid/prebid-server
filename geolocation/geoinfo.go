package geolocation

type GeoInfo struct {
	// Name of the geo location data provider.
	Vendor string

	// Continent code in two-letter format.
	Continent string

	// Country code in ISO-3166-1-alpha-2 format.
	Country string

	// Region code in ISO-3166-2 format.
	Region string

	// Numeric region code.
	RegionCode int

	City string

	// Google Metro code.
	MetroGoogle string

	// Nielsen Designated Market Areas (DMA's).
	MetroNielsen int

	Zip string

	ConnectionSpeed string

	Lat float64

	Lon float64

	TimeZone string
}
