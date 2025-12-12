package floors

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
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

	// Parse the incoming OpenRTB request FIRST to extract values for analytics
	var req map[string]interface{}
	if err := json.Unmarshal(payload, &req); err != nil {
		fmt.Println("Unmarshal failed:", err)
		// If unmarshal fails, return original payload (fail open)
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, err
	}

	// Extract values needed for analytics and mutation
	ip := ""
	ua := ""
	siteUID := ""
	country := ""

	if site, ok := req["site"].(map[string]interface{}); ok {
		if ext, ok := site["ext"].(map[string]interface{}); ok {
			if data, ok := ext["data"].(map[string]interface{}); ok {
				if siteUIDVal, exists := data["site_uid"]; exists {
					siteUID, ok = siteUIDVal.(string)
					if !ok {
						fmt.Println("site_uid is not a string, got type:", fmt.Sprintf("%T", siteUIDVal), "value:", siteUIDVal, ", skipping floors injection")
						return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
					}
				} else {
					fmt.Println("site_uid key not found in site.ext.data, skipping floors injection")
					return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
				}
			} else {
				fmt.Println("site.ext.data not found, skipping floors injection")
				return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
			}
		} else {
			fmt.Println("site.ext not found, skipping floors injection")
			return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
		}
	} else {
		fmt.Println("site not found in request, skipping floors injection")
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
	}
	fmt.Println("siteID is", siteUID)
	if siteUID == "" {
		fmt.Println("siteUID is empty after extraction, skipping floors injection")
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
	}

	siteConfig, ok := floorsConfig[siteUID]
	if !ok {
		// Get available config keys for debugging
		availableKeys := make([]string, 0, len(floorsConfig))
		for k := range floorsConfig {
			availableKeys = append(availableKeys, k)
		}
		fmt.Println("site config not found for site id", siteUID, ", available configs:", availableKeys, ", skipping floors injection")
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
	}

	if device, ok := req["device"].(map[string]interface{}); ok {
		if uaStr, ok := device["ua"].(string); ok {
			ua = uaStr
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

		// First try to get IP from ModuleContext (passed from Entrypoint hook)
		if moduleCtx.ModuleContext != nil {
			if ipFromHeader, ok := moduleCtx.ModuleContext[deviceIPCtxKey].(string); ok && ipFromHeader != "" {
				ip = ipFromHeader
				fmt.Println("Using IP from headers (via ModuleContext):", ip)
			}
		}
		if ip != "" {
			resolvedCountry, err := f.geoResolver.Resolve(ctx, ip)
			if err != nil {
				fmt.Println("Error resolving country from IP:", err)
				return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, err
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
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
	}

	if !slices.Contains(siteConfig.countries, country) {
		fmt.Println("country is not in site config countries", country, ", available countries:", siteConfig.countries, ", using OC instead")
		country = "OC"
	}

	if !slices.Contains(siteConfig.platforms, platform) {
		fmt.Println("platform is not in site config platforms, got:", platform, ", available platforms:", siteConfig.platforms, ", skipping floors injection")
		return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, nil
	}

	// Build the floor URL using the extracted values
	floorURL := "https://t-f.mile.so/floor-server/" + siteConfig.siteUID + "/" + country + "/" + platform + "/floor.json"
	fmt.Println("Generated floor URL:", floorURL)

	// Now create the mutation using the extracted values
	c := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(
		func(orig hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			// Parse the request again in the mutation
			var req map[string]interface{}
			if err := json.Unmarshal(orig, &req); err != nil {
				fmt.Println("Unmarshal failed in mutation:", err)
				return orig, err
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
			// Use the floorURL variable from the outer scope
			floors["floorendpoint"] = map[string]interface{}{
				"url": floorURL,
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

	// Prepare analytics tags with floor information using the pre-computed values
	analyticsTags := hookstage.HookResult[hookstage.RawAuctionRequestPayload]{
		Reject:    false,
		ChangeSet: c,
		AnalyticsTags: hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{
				{
					Name:   "floor-injection",
					Status: hookanalytics.ActivityStatusSuccess,
					Results: []hookanalytics.Result{
						{
							Status: hookanalytics.ResultStatusModify,
							Values: map[string]interface{}{
								"site_uid":  siteUID,
								"country":   country,
								"platform":  platform,
								"floor_url": floorURL,
							},
							AppliedTo: hookanalytics.AppliedTo{
								Request: true,
							},
						},
					},
				},
			},
		},
	}

	return analyticsTags, nil
}

// HandleBidderRequestHook captures bidder-specific floor values after SSP floors are applied
func (f *FloorsInjector) HandleBidderRequestHook(
	ctx context.Context,
	moduleCtx hookstage.ModuleInvocationContext,
	payload hookstage.BidderRequestPayload,
) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {

	// Extract bidder name from payload
	bidderName := payload.Bidder

	// Collect floor values organized by impression ID -> bidder -> floor data
	impressionFloors := make(map[string]interface{})

	if payload.Request != nil {
		for _, impWrapper := range payload.Request.GetImp() {
			if impWrapper.BidFloor > 0 {
				// Structure: impression_id -> bidder_name -> floor data
				impressionFloors[impWrapper.ID] = map[string]interface{}{
					bidderName: map[string]interface{}{
						"floor":    impWrapper.BidFloor,
						"currency": impWrapper.BidFloorCur,
					},
				}
			}
		}
	}

	// Only return analytics tags if there are floors for this bidder
	if len(impressionFloors) == 0 {
		return hookstage.HookResult[hookstage.BidderRequestPayload]{}, nil
	}

	// Return analytics tags with impression-centric floor data
	result := hookstage.HookResult[hookstage.BidderRequestPayload]{
		AnalyticsTags: hookanalytics.Analytics{
			Activities: []hookanalytics.Activity{
				{
					Name:   "bidder-floors",
					Status: hookanalytics.ActivityStatusSuccess,
					Results: []hookanalytics.Result{
						{
							Status: hookanalytics.ResultStatusModify,
							Values: impressionFloors,
							AppliedTo: hookanalytics.AppliedTo{
								Bidder:  bidderName,
								Request: true,
							},
						},
					},
				},
			},
		},
	}

	return result, nil
}
