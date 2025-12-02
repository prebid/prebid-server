package trafficshaping

import (
	"context"
	"errors"
	"fmt"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/modules/mile/common"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// buildConfigURL constructs the traffic shaping config URL from request data
// Format: {baseEndpoint}{country}/{device}/{browser}/ts.json
// Returns error if required fields are missing (triggers fail-open behavior)
func buildConfigURL(baseEndpoint string, wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper == nil || wrapper.BidRequest == nil {
		return "", errors.New("invalid request wrapper")
	}

	// Extract country from device.geo.country
	country, err := common.ExtractCountry(wrapper)
	if err != nil {
		return "", err
	}

	// Map device.devicetype to device category
	deviceCategory, err := common.ExtractDeviceCategory(wrapper)
	if err != nil {
		return "", err
	}

	// Parse device.ua to detect browser
	browser, err := common.ExtractBrowser(wrapper)
	if err != nil {
		return "", err
	}

	// Construct URL: {baseEndpoint}{country}/{device}/{browser}/ts.json
	url := fmt.Sprintf("%s%s/%s/%s/ts.json", baseEndpoint, country, deviceCategory, browser)
	return url, nil
}

// extractSiteID extracts the site ID from the bid request
func extractSiteID(wrapper *openrtb_ext.RequestWrapper) (string, error) {
	if wrapper.BidRequest == nil || wrapper.BidRequest.Site == nil || wrapper.BidRequest.Site.ID == "" {
		return "", errors.New("missing site.id")
	}
	return wrapper.BidRequest.Site.ID, nil
}

// buildConfigURLWithFallback resolves missing country/device using fallbacks.
func buildConfigURLWithFallback(ctx context.Context, baseEndpoint string, wrapper *openrtb_ext.RequestWrapper, geoResolver common.GeoResolver) (string, []hookanalytics.Activity, error) {
	if wrapper == nil || wrapper.BidRequest == nil {
		return "", nil, errors.New("invalid request wrapper")
	}

	siteID, err := extractSiteID(wrapper)
	if err != nil {
		return "", nil, errors.New("site.id unavailable")
	}

	// Use the common resolver to extract all information with fallbacks
	resolver := common.NewDefaultResolver(geoResolver)
	info, activities, err := resolver.Resolve(ctx, wrapper)
	if err != nil {
		return "", activities, err
	}

	url := fmt.Sprintf("%s%s/%s/%s/%s/ts.json", baseEndpoint, siteID, info.Country, info.Device, info.Browser)
	return url, activities, nil
}
