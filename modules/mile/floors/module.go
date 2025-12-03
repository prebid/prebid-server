package floors

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/mile/common"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Builder creates a new floors injector module instance
func Builder(rawConfig json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {

	config, err := parseConfig(rawConfig)
	if err != nil {
		return nil, err
	}
	var geoResolver *common.MaxMindGeoResolver
	if config.GeoEnabled() {
		geoResolver, err = common.NewMaxMindGeoResolver(config.GeoDBPath)
		if err != nil {
			return nil, err
		}
	}
	return &FloorsInjector{geoResolver: geoResolver}, nil
}

type FloorsInjector struct {
	geoResolver *common.MaxMindGeoResolver
}

func (f *FloorsInjector) HandleRawAuctionHook(
	ctx context.Context,
	moduleCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawAuctionRequestPayload,
) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {

	c := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(
		func(orig hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			// Parse the incoming OpenRTB request
			var req map[string]interface{}
			if err := json.Unmarshal(orig, &req); err != nil {
				fmt.Println("Unmarshal failed:", err)
				// If unmarshal fails, return original payload (fail open)
				return orig, err
			}

			// get IP and ua from raw ortb request
			ip := ""
			ua := ""
			country := ""
			if device, ok := req["device"].(map[string]interface{}); ok {
				ua = device["ua"].(string)
				switch ipv := device["ip"].(type) {
				case string:
					ip = ipv
				}
				// fallback to ipv6 if no ipv4
				if ip == "" {
					if ipv6, ok := device["ipv6"].(string); ok {
						ip = ipv6
					}
				}

				// Check if country is already present in device.geo.country
				if geo, ok := device["geo"].(map[string]interface{}); ok {
					fmt.Println("geo is", geo)
					if countryCode, ok := geo["country"].(string); ok && countryCode != "" {
						country = countryCode
						fmt.Println("Country found in request:", country)
					}
				}
			}

			// Only resolve country from IP if not already present in request
			if country == "" && f.geoResolver != nil {
				if ip != "" {
					resolvedCountry, err := f.geoResolver.Resolve(ctx, ip)
					if err != nil {
						fmt.Println("Error resolving country from IP:", err)
						return orig, err
					}
					country = resolvedCountry
					fmt.Println("Country resolved from IP:", country)
				}
			}

			if country == "" {
				fmt.Println("country is empty for ip", ip, ", using OC instead")
				country = "OC"
			}

			platform := common.ClassifyDevicePlatform(ua)
			fmt.Println("platform is", platform)
			if platform == "" {
				fmt.Println("platform is empty for ua", ua, ", skipping floors injection")
				return orig, nil
			}

			siteID := ""
			if site, ok := req["site"].(map[string]interface{}); ok {
				siteID, ok = site["id"].(string)
				if !ok {
					fmt.Println("site id is not a string", site["id"], ", skipping floors injection")
					return orig, nil
				}
			}
			fmt.Println("siteID is", siteID)
			siteConfig, ok := floorsConfig[siteID]
			if !ok {
				fmt.Println("site config not found for site id", siteID, ", skipping floors injection")
				return orig, nil
			}

			if !slices.Contains(siteConfig.countries, country) {
				fmt.Println("country is not in site config countries", siteConfig.countries, ", using OC instead")
				country = "OC"
			}

			// Ensure ext and ext.prebid exist
			ext, ok := req["ext"].(map[string]interface{})
			if !ok {
				ext = make(map[string]interface{})
				req["ext"] = ext
			}
			prebid, ok := ext["prebid"].(map[string]interface{})
			if !ok {
				prebid = make(map[string]interface{})
				ext["prebid"] = prebid
			}

			// Ensure floors exists
			floors, ok := prebid["floors"].(map[string]interface{})
			if !ok {
				floors = make(map[string]interface{})
				prebid["floors"] = floors
			}

			// Inject floorendpoint - this triggers the fetch
			// Don't set 'location' or 'fetchstatus' - Prebid Server sets these after fetch
			floors["floorendpoint"] = map[string]interface{}{
				// "url": "https://floors.atmtd.com/floors.json?siteID=g35tzr",
				"url": "https://t-f.mile.so/floor-server/" + siteConfig.siteUID + "/" + country + "/" + platform + "/floor.json",
			}

			floors["enabled"] = true
			floors["enforcement"] = map[string]interface{}{
				"enforcerate": 100,  // 0-100, where 100 = always enforce
				"enforcepbs":  true, // enforce in PBS
				"floordeals":  true, // enforce for deals too
			}

			// Marshal back to JSON
			mutated, err := json.Marshal(req)
			if err != nil {
				return orig, nil
			}

			return hookstage.RawAuctionRequestPayload(mutated), nil
		}, hookstage.MutationUpdate, "ext", "prebid", "floors",
	)

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{
		Reject:    false,
		ChangeSet: c,
	}, nil
}
