package trafficshaping

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
)

// Builder creates a new traffic shaping module instance
func Builder(rawConfig json.RawMessage, deps moduledeps.ModuleDeps) (interface{}, error) {
	config, err := parseConfig(rawConfig)
	if err != nil {
		return nil, err
	}

	client := NewConfigClient(deps.HTTPClient, config)

	var geoResolver GeoResolver
	if config.GeoEnabled() {
		geoResolver, err = NewHTTPGeoResolver(config.GeoLookupEndpoint, config.GetGeoCacheTTL(), deps.HTTPClient)
		if err != nil {
			return nil, err
		}
	}

	return &Module{
		config:      config,
		client:      client,
		geoResolver: geoResolver,
		httpClient:  deps.HTTPClient,
	}, nil
}

// Module implements the traffic shaping hook
type Module struct {
	config      *Config
	client      *ConfigClient
	geoResolver GeoResolver
	httpClient  *http.Client
}

// HandleProcessedAuctionHook implements the ProcessedAuctionRequest hook
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	moduleCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	result := hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{
		AnalyticsTags: hookanalytics.Analytics{},
	}

	wrapper := payload.Request
	if wrapper == nil || wrapper.BidRequest == nil {
		return result, nil
	}

	// Get the shaping config (dynamic or static mode)
	var shapingConfig *ShapingConfig

	if m.config.IsDynamicMode() {
		// Dynamic mode: construct URL from request data
		configURL, tags, err := buildConfigURLWithFallback(ctx, m.config.BaseEndpoint, wrapper, m.geoResolver)
		if err != nil {
			// Fail-open: skip shaping if URL cannot be built
			result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
				hookanalytics.Activity{Name: "skipped_url_construction_failed", Status: hookanalytics.ActivityStatusSuccess},
				hookanalytics.Activity{Name: "skipped", Status: hookanalytics.ActivityStatusSuccess})
			return result, nil
		}

		result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities, tags...)

		shapingConfig = m.client.GetConfigForURL(configURL)
		if shapingConfig == nil {
			result.Warnings = append(result.Warnings, "trafficshaping: config unavailable for "+configURL)
			result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
				hookanalytics.Activity{Name: "skipped_no_config", Status: hookanalytics.ActivityStatusSuccess},
				hookanalytics.Activity{Name: "fetch_failed", Status: hookanalytics.ActivityStatusSuccess},
				hookanalytics.Activity{Name: "skipped", Status: hookanalytics.ActivityStatusSuccess})
			return result, nil
		}
	} else {
		// Static mode (legacy): use single endpoint
		shapingConfig = m.client.GetConfig()
		if shapingConfig == nil {
			result.Warnings = append(result.Warnings, "trafficshaping: config unavailable")
			result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
				hookanalytics.Activity{Name: "skipped_no_config", Status: hookanalytics.ActivityStatusSuccess},
				hookanalytics.Activity{Name: "fetch_failed", Status: hookanalytics.ActivityStatusSuccess},
				hookanalytics.Activity{Name: "skipped", Status: hookanalytics.ActivityStatusSuccess})
			return result, nil
		}
	}

	// Check skipRate
	if shouldSkipByRate(wrapper.ID, shapingConfig.SkipRate, m.config.SampleSalt) {
		result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
			hookanalytics.Activity{Name: "skipped_by_skiprate", Status: hookanalytics.ActivityStatusSuccess},
			hookanalytics.Activity{Name: "skipped", Status: hookanalytics.ActivityStatusSuccess})
		return result, nil
	}

	// Apply account-level config overrides if present
	accountConfig := m.getAccountConfig(moduleCtx.AccountConfig)

	// Determine which country list to use (account overrides host)
	allowedCountries := m.config.GetAllowedCountriesMap()
	if accountConfig != nil && accountConfig.AllowedCountries != nil {
		allowedCountries = accountConfig.GetAllowedCountriesMap()
	}

	// Check country gating with the final allowed countries list
	if shouldSkipByCountry(wrapper, allowedCountries) {
		result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
			hookanalytics.Activity{Name: "skipped_country", Status: hookanalytics.ActivityStatusSuccess},
			hookanalytics.Activity{Name: "skipped", Status: hookanalytics.ActivityStatusSuccess})
		return result, nil
	}

	// Apply shaping per impression
	impWrappers := wrapper.GetImp()
	shapedCount := 0
	missingGPIDCount := 0

	// Collect all allowed bidders across all impressions
	globalAllowedBidders := make(map[string]struct{})

	for _, impWrapper := range impWrappers {
		gpid := getGPID(impWrapper)
		if gpid == "" {
			missingGPIDCount++
			continue
		}

		allowedBidders := getAllowedBidders(gpid, shapingConfig)
		if allowedBidders == nil {
			missingGPIDCount++
			continue
		}

		// Merge allowed bidders for this impression
		for bidder := range allowedBidders {
			globalAllowedBidders[bidder] = struct{}{}
		}

		// Filter banner sizes
		if impWrapper.Imp != nil && impWrapper.Imp.Banner != nil {
			rule := shapingConfig.GPIDRules[gpid]
			if rule != nil {
				filterBannerSizes(impWrapper.Imp, rule.AllowedSizes)
			}
		}

		shapedCount++
	}

	// Apply bidder filtering using mutations if we have any shaped impressions
	if shapedCount > 0 && len(globalAllowedBidders) > 0 {
		changeSet := &result.ChangeSet
		changeSet.ProcessedAuctionRequest().Bidders().Add(globalAllowedBidders)
	}

	// Prune EIDs if enabled
	pruneUserIds := m.config.PruneUserIds
	if accountConfig != nil && accountConfig.PruneUserIds != nil {
		pruneUserIds = *accountConfig.PruneUserIds
	}

	if pruneUserIds && len(shapingConfig.UserIdVendors) > 0 {
		if err := pruneEIDs(wrapper, shapingConfig.UserIdVendors); err != nil {
			result.Warnings = append(result.Warnings, "trafficshaping: failed to prune eids")
		}
	}

	// Add analytics tags
	if shapedCount > 0 {
		result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
			hookanalytics.Activity{Name: "applied", Status: hookanalytics.ActivityStatusSuccess},
			hookanalytics.Activity{Name: "shaped", Status: hookanalytics.ActivityStatusSuccess})
	}
	if missingGPIDCount > 0 {
		result.AnalyticsTags.Activities = append(result.AnalyticsTags.Activities,
			hookanalytics.Activity{Name: "missing_gpid", Status: hookanalytics.ActivityStatusSuccess})
	}

	return result, nil
}

// AccountConfig represents account-level overrides
type AccountConfig struct {
	Endpoint         *string   `json:"endpoint"`
	PruneUserIds     *bool     `json:"prune_user_ids"`
	AllowedCountries *[]string `json:"allowed_countries"`
}

// GetAllowedCountriesMap returns a map of allowed countries for fast lookup
func (c *AccountConfig) GetAllowedCountriesMap() map[string]struct{} {
	if c.AllowedCountries == nil || len(*c.AllowedCountries) == 0 {
		return nil
	}

	countries := make(map[string]struct{}, len(*c.AllowedCountries))
	for _, country := range *c.AllowedCountries {
		countries[country] = struct{}{}
	}
	return countries
}

// getAccountConfig parses account-level config overrides
func (m *Module) getAccountConfig(rawConfig json.RawMessage) *AccountConfig {
	if len(rawConfig) == 0 {
		return nil
	}

	var accountConfig AccountConfig
	if err := json.Unmarshal(rawConfig, &accountConfig); err != nil {
		return nil
	}

	return &accountConfig
}
