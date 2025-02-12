package usersync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/v3/config"
)

type namedSyncerConfig struct {
	name string
	cfg  config.Syncer
}

// SyncerBuildError represents an error with building a syncer.
type SyncerBuildError struct {
	Bidder    string
	SyncerKey string
	Err       error
}

// Error implements the standard error interface.
func (e SyncerBuildError) Error() string {
	return fmt.Sprintf("cannot create syncer for bidder %s with key %s: %v", e.Bidder, e.SyncerKey, e.Err)
}

func BuildSyncers(hostConfig *config.Configuration, bidderInfos config.BidderInfos) (map[string]Syncer, []error) {
	// map syncer config by bidder
	cfgByBidder := make(map[string]config.Syncer, len(bidderInfos))
	for bidder, cfg := range bidderInfos {
		if shouldCreateSyncer(cfg) {
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

	// resolve host endpoint
	hostUserSyncConfig := hostConfig.UserSync
	if hostUserSyncConfig.ExternalURL == "" {
		hostUserSyncConfig.ExternalURL = hostConfig.ExternalURL
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

		for _, bidder := range cfgGroup {
			syncer, err := NewSyncer(hostUserSyncConfig, primaryCfg.cfg, bidder.name)
			if err != nil {
				errs = append(errs, SyncerBuildError{
					Bidder:    primaryCfg.name,
					SyncerKey: key,
					Err:       err,
				})
				continue
			}
			syncers[bidder.name] = syncer
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}
	return syncers, nil
}

func shouldCreateSyncer(cfg config.BidderInfo) bool {
	if cfg.Disabled {
		return false
	}

	// a syncer may provide just a Supports field to provide hints to the host. we should only try to create a syncer
	// if there is at least one non-Supports value populated.
	return cfg.Syncer.Defined()
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
