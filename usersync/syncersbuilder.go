package usersync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
)

type namedSyncerConfig struct {
	name string
	cfg  config.Syncer
}

func BuildSyncers(hostConfig config.UserSync, bidderInfos config.BidderInfos) (map[string]Syncer, error) {
	// map syncer config by bidder
	cfgByBidder := make(map[string]config.Syncer, len(bidderInfos))
	for bidder, cfg := range bidderInfos {
		if cfg.Enabled && cfg.Syncer != nil {
			cfgByBidder[bidder] = *cfg.Syncer
		}
	}

	// map syncer config by key
	cfgBySyncerKey := make(map[string][]namedSyncerConfig, len(bidderInfos))
	for bidder, cfg := range cfgByBidder {
		if cfg.Key == "" {
			cfg.Key = bidder
		}
		cfgBySyncerKey[cfg.Key] = append(cfgBySyncerKey[cfg.Key], namedSyncerConfig{bidder, cfg})
	}

	// create syncers
	errs := []error{}
	syncers := make(map[string]Syncer, len(bidderInfos))
	for key, cfgGroup := range cfgBySyncerKey {
		primaryCfg, err := chooseSyncerConfig(cfgGroup)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		syncer, err := NewSyncer(hostConfig, primaryCfg.cfg)
		if err != nil {
			errs = append(errs, fmt.Errorf("cannot create syncer for bidder %s with key %s. %v", primaryCfg.name, key, err))
			continue
		}

		for _, bidder := range cfgGroup {
			syncers[bidder.name] = syncer
		}
	}

	if len(errs) > 0 {
		return nil, errortypes.NewAggregateError("user sync", errs)
	}
	return syncers, nil
}

func chooseSyncerConfig(biddersSyncerConfig []namedSyncerConfig) (namedSyncerConfig, error) {
	if len(biddersSyncerConfig) == 1 {
		return biddersSyncerConfig[0], nil
	}

	var bidderNames []string
	var bidderNamesWithEndpoints []string
	var syncerConfig namedSyncerConfig
	for _, bidder := range biddersSyncerConfig {
		bidderNames = append(bidderNames, bidder.name)
		if bidder.cfg.IFrame != nil || bidder.cfg.Redirect != nil {
			bidderNamesWithEndpoints = append(bidderNamesWithEndpoints, bidder.name)
			syncerConfig = bidder
		}
	}

	if len(bidderNamesWithEndpoints) == 0 {
		sort.Strings(bidderNames)
		bidders := strings.Join(bidderNames, ", ")
		return namedSyncerConfig{}, fmt.Errorf("bidders %s share the same syncer key, but none define endpoints (iframe and/or redirect)", bidders)
	}

	if len(bidderNamesWithEndpoints) > 1 {
		sort.Strings(bidderNamesWithEndpoints)
		bidders := strings.Join(bidderNamesWithEndpoints, ", ")
		return namedSyncerConfig{}, fmt.Errorf("bidders %s define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints", bidders)
	}

	return syncerConfig, nil
}
