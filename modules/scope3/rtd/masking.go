package scope3

import (
	"math"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/iterutil"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// maskBidRequest creates a deep copy of the bid request with sensitive fields masked
// according to the masking configuration. Returns nil if masking fails to prevent
// accidental exposure of sensitive data.
func (m *Module) maskBidRequest(original *openrtb2.BidRequest) *openrtb2.BidRequest {
	if !m.cfg.Masking.Enabled {
		return original
	}

	// Create a deep copy by marshaling and unmarshaling
	data, err := jsonutil.Marshal(original)
	if err != nil {
		// Never return unmasked data - this prevents potential data leakage
		// The calling function should handle nil gracefully
		return nil
	}

	var masked openrtb2.BidRequest
	if err := jsonutil.Unmarshal(data, &masked); err != nil {
		// Never return unmasked data - this prevents potential data leakage
		// The calling function should handle nil gracefully
		return nil
	}

	// Apply masking to different sections
	m.maskUser(&masked)
	m.maskDevice(&masked)
	m.maskGeo(&masked)

	return &masked
}

// maskUser removes or filters user data according to privacy settings
func (m *Module) maskUser(req *openrtb2.BidRequest) {
	if req.User == nil {
		return
	}

	// Always remove publisher's first-party user ID for privacy
	req.User.ID = ""
	req.User.BuyerUID = ""

	// Always remove potentially sensitive demographic data
	req.User.Yob = 0
	req.User.Gender = ""

	// Remove user data segments (first-party data)
	req.User.Data = nil
	req.User.Keywords = ""

	// Filter user.eids to only preserve allowed identity providers
	req.User.EIDs = m.filterEids(req.User.EIDs)
}

// filterEids filters the user.eids array to only include allowed identity providers
func (m *Module) filterEids(eids []openrtb2.EID) []openrtb2.EID {
	if len(m.cfg.Masking.User.PreserveEids) == 0 {
		return []openrtb2.EID{}
	}

	// Create allowlist map for fast lookup
	allowed := make(map[string]bool)
	for _, source := range m.cfg.Masking.User.PreserveEids {
		allowed[source] = true
	}

	// Filter eids to only include allowed sources
	var filtered []openrtb2.EID
	for eid := range iterutil.SlicePointerValues(eids) {
		if allowed[eid.Source] {
			// Intentionally copy the EID struct to create a new filtered slice
			// that's independent of the original data
			filtered = append(filtered, *eid)
		}
	}

	return filtered
}

// maskDevice removes sensitive device information while preserving targeting-safe data
func (m *Module) maskDevice(req *openrtb2.BidRequest) {
	if req.Device == nil {
		return
	}

	// Always remove IP addresses for privacy
	req.Device.IP = ""
	req.Device.IPv6 = ""

	// Remove mobile advertising IDs unless explicitly preserved
	if !m.cfg.Masking.Device.PreserveMobileIds {
		req.Device.IFA = ""
		req.Device.DPIDMD5 = ""
		req.Device.DPIDSHA1 = ""
		req.Device.DIDMD5 = ""
		req.Device.DIDSHA1 = ""
		req.Device.MACMD5 = ""
		req.Device.MACSHA1 = ""
	}

	// Note: We preserve device characteristics like devicetype, os, browser, etc.
	// as these are not considered personally identifiable and are useful for targeting
}

// maskGeo removes or truncates geographic data according to privacy settings
func (m *Module) maskGeo(req *openrtb2.BidRequest) {
	// Mask device geo
	if req.Device != nil && req.Device.Geo != nil {
		m.maskGeoObject(req.Device.Geo)
	}

	// Mask user geo (if different from device geo)
	if req.User != nil && req.User.Geo != nil {
		m.maskGeoObject(req.User.Geo)
	}
}

// maskGeoObject applies geographic masking rules to a geo object
func (m *Module) maskGeoObject(geo *openrtb2.Geo) {
	// Always preserve country and region (state) as these are not considered PII
	// geo.Country and geo.Region are preserved

	// Handle optional geographic fields based on configuration
	if !m.cfg.Masking.Geo.PreserveMetro {
		geo.Metro = ""
	}
	if !m.cfg.Masking.Geo.PreserveZip {
		geo.ZIP = ""
	}
	if !m.cfg.Masking.Geo.PreserveCity {
		geo.City = ""
	}

	// Handle lat/long based on precision setting
	if m.cfg.Masking.Geo.LatLongPrecision == 0 {
		// Remove completely
		geo.Lat = nil
		geo.Lon = nil
	} else if geo.Lat != nil && geo.Lon != nil {
		// Truncate to specified precision
		truncatedLat := m.truncateCoordinate(*geo.Lat, m.cfg.Masking.Geo.LatLongPrecision)
		truncatedLon := m.truncateCoordinate(*geo.Lon, m.cfg.Masking.Geo.LatLongPrecision)
		geo.Lat = &truncatedLat
		geo.Lon = &truncatedLon
	}

	// Always remove high-precision location data
	geo.Accuracy = 0 // GPS accuracy radius could reveal precision
}

// truncateCoordinate truncates a coordinate to the specified number of decimal places
func (m *Module) truncateCoordinate(coord float64, precision int) float64 {
	if precision <= 0 || precision > 4 {
		return 0
	}

	multiplier := math.Pow(10, float64(precision))
	// Use math.Trunc instead of math.Floor to handle negative numbers correctly
	return math.Trunc(coord*multiplier) / multiplier
}

// getMaskingSummary returns a summary of what fields would be masked for analytics/debugging
func (m *Module) getMaskingSummary() map[string]interface{} {
	if !m.cfg.Masking.Enabled {
		return map[string]interface{}{"enabled": false}
	}

	return map[string]interface{}{
		"enabled": true,
		"geo": map[string]interface{}{
			"preserve_metro":     m.cfg.Masking.Geo.PreserveMetro,
			"preserve_zip":       m.cfg.Masking.Geo.PreserveZip,
			"preserve_city":      m.cfg.Masking.Geo.PreserveCity,
			"lat_long_precision": m.cfg.Masking.Geo.LatLongPrecision,
		},
		"user": map[string]interface{}{
			"preserve_eids": m.cfg.Masking.User.PreserveEids,
		},
		"device": map[string]interface{}{
			"preserve_mobile_ids": m.cfg.Masking.Device.PreserveMobileIds,
		},
		"always_removed": []string{
			"device.ip", "device.ipv6", "user.id", "user.buyeruid",
			"user.yob", "user.gender", "user.data", "user.keywords", "geo.accuracy",
		},
		"never_removed": []string{
			"geo.country", "geo.region", "device.devicetype", "device.os",
			"device.browser", "device.make", "device.model", "site.*", "app.*", "imp.*",
		},
	}
}
