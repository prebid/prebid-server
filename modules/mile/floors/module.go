package floors

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/mile/common"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/util/httputil"
	"github.com/prebid/prebid-server/v3/util/iputil"
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

const (
	deviceIPCtxKey = "device_ip"
)

// HandleEntrypointHook extracts the device IP from HTTP headers and stores it in ModuleContext
func (f *FloorsInjector) HandleEntrypointHook(
	ctx context.Context,
	moduleCtx hookstage.ModuleInvocationContext,
	payload hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {

	// Extract IP from headers using httputil which checks X-Forwarded-For, X-Real-IP, etc.
	ip, _ := httputil.FindIP(payload.Request, iputil.PublicNetworkIPValidator{})

	// Create module context to pass IP to later stages
	modCtx := make(hookstage.ModuleContext)
	if ip != nil {
		modCtx[deviceIPCtxKey] = ip.String()
		fmt.Println("Extracted IP from headers:", ip.String())
	}

	return hookstage.HookResult[hookstage.EntrypointPayload]{
		ModuleContext: modCtx,
	}, nil
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

			// First try to get IP from ModuleContext (passed from Entrypoint hook)
			if moduleCtx.ModuleContext != nil {
				if ipFromHeader, ok := moduleCtx.ModuleContext[deviceIPCtxKey].(string); ok && ipFromHeader != "" {
					ip = ipFromHeader
					fmt.Println("Using IP from headers (via ModuleContext):", ip)
				}
			}

			siteUID := ""
			if site, ok := req["site"].(map[string]interface{}); ok {
				if ext, ok := site["ext"].(map[string]interface{}); ok {
					if data, ok := ext["data"].(map[string]interface{}); ok {
						siteUID, ok = data["site_uid"].(string)
						if !ok {
							fmt.Println("site_uid is not a string", data["site_uid"], ", skipping floors injection")
							return orig, nil
						}
					}
				}
			}
			fmt.Println("siteID is", siteUID)
			siteConfig, ok := floorsConfig[siteUID]
			if !ok {
				fmt.Println("site config not found for site id", siteUID, ", skipping floors injection")
				return orig, nil
			}

			if device, ok := req["device"].(map[string]interface{}); ok {
				ua = device["ua"].(string)

				if ip == "" {
					fmt.Println("ip is empty")
				}

				// Check if country is already present in device.geo.country
				if geo, ok := device["geo"].(map[string]interface{}); ok {
					fmt.Println("geo is", geo, "from request")
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

			if !slices.Contains(siteConfig.countries, country) {
				fmt.Println("country is not in site config countries", country, ", using OC instead")
				country = "OC"
			}

			if !slices.Contains(siteConfig.platforms, platform) {
				fmt.Println("platform is not in site config platforms", platform, "skipping floors injection")
				return orig, nil
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
