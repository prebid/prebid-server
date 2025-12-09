package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// RequestInfo contains resolved country, device, and browser information
type RequestInfo struct {
	Country string
	Device  string
	Browser string
}

// Resolver resolves request information (country, device, browser) from OpenRTB request
type Resolver interface {
	Resolve(ctx context.Context, wrapper *openrtb_ext.RequestWrapper) (RequestInfo, []hookanalytics.Activity, error)
}

// DefaultResolver implements Resolver with fallback logic for country and device
type DefaultResolver struct {
	geoResolver GeoResolver
}

// NewDefaultResolver creates a new DefaultResolver
func NewDefaultResolver(geoResolver GeoResolver) *DefaultResolver {
	return &DefaultResolver{
		geoResolver: geoResolver,
	}
}

// Resolve extracts country, device, and browser from the request with fallbacks
func (r *DefaultResolver) Resolve(ctx context.Context, wrapper *openrtb_ext.RequestWrapper) (RequestInfo, []hookanalytics.Activity, error) {
	activities := make([]hookanalytics.Activity, 0, 3)
	info := RequestInfo{}

	if wrapper == nil || wrapper.BidRequest == nil {
		return info, activities, errors.New("invalid request wrapper")
	}

	// Extract country with fallback
	country, err := ExtractCountry(wrapper)
	if err != nil || country == "" {
		derivedCountry, derr := DeriveCountry(ctx, wrapper, r.geoResolver)
		if derr == nil && derivedCountry != "" {
			country = derivedCountry
			activities = append(activities, hookanalytics.Activity{Name: "country_derived", Status: hookanalytics.ActivityStatusSuccess})
		} else {
			return info, activities, errors.New("country unavailable")
		}
	}
	info.Country = country

	// Extract device category with fallback
	deviceCategory, err := ExtractDeviceCategory(wrapper)
	if err != nil || deviceCategory == "" {
		derivedDevice := DeriveDeviceCategory(wrapper)
		if derivedDevice == "" {
			return info, activities, errors.New("device category unavailable")
		}
		deviceCategory = derivedDevice
		activities = append(activities, hookanalytics.Activity{Name: "devicetype_derived", Status: hookanalytics.ActivityStatusSuccess})
	}
	info.Device = deviceCategory

	// Extract browser (no fallback)
	browser, err := ExtractBrowser(wrapper)
	if err != nil {
		return info, activities, err
	}
	info.Browser = browser

	return info, activities, nil
}

// ExtractCountry extracts and normalizes the country code from device.geo.country
func ExtractCountry(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.Device == nil || wrapper.Device.Geo == nil || wrapper.Device.Geo.Country == "" {
		return "", errors.New("missing device.geo.country")
	}

	country := strings.ToUpper(strings.TrimSpace(wrapper.Device.Geo.Country))
	if len(country) != 2 {
		return "", errors.New("invalid country code: expected 2-letter ISO code")
	}

	return country, nil
}

// DeriveCountry derives country from IP address using GeoResolver
func DeriveCountry(ctx context.Context, wrapper *openrtb_ext.RequestWrapper, geoResolver GeoResolver) (string, error) {
	if geoResolver == nil {
		return "", errors.New("geo resolver not configured")
	}
	if wrapper.Device == nil {
		return "", errors.New("device missing")
	}
	ip := strings.TrimSpace(wrapper.Device.IP)
	if ip == "" {
		ip = strings.TrimSpace(wrapper.Device.IPv6)
	}

	// Don't attempt resolution if IP is empty
	if ip == "" {
		return "", errors.New("ip address required for geo resolution")
	}

	country, err := geoResolver.Resolve(ctx, ip)
	if err != nil || country == "" {
		return "", fmt.Errorf("country resolve failed: %w", err)
	}
	return country, nil
}
