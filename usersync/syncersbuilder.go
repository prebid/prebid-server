package usersync

import (
	"fmt"
	"sort"
	"strings"

	"github.com/prebid/prebid-server/config"
)

type namedSyncerConfig struct {
	name       string
	cfg        config.Syncer
	bidderInfo config.BidderInfo
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
	// map syncer config by key
	cfgBySyncerKey := make(map[string][]namedSyncerConfig)
	for bidder, bidderInfo := range bidderInfos {
		if shouldCreateSyncer(bidderInfo) {
			syncerCopy := *bidderInfo.Syncer
			if syncerCopy.Key == "" {
				var err error
				syncerCopy.Key, err = getSyncerKey(bidderInfos, bidder, bidderInfo, syncerCopy)
				if err != nil {
					return nil, []error{err}
				}
			}
			cfgBySyncerKey[syncerCopy.Key] = append(cfgBySyncerKey[syncerCopy.Key], namedSyncerConfig{bidder, syncerCopy, bidderInfo})
		}
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

	if cfg.Syncer == nil {
		return false
	}

	// a syncer may provide just a Supports field to provide hints to the host. we should only try to create a syncer
	// if there is at least one non-Supports value populated.
	return cfg.Syncer.Key != "" || cfg.Syncer.IFrame != nil || cfg.Syncer.Redirect != nil || cfg.Syncer.SupportCORS != nil
}

func chooseSyncerConfig(biddersSyncerConfig []namedSyncerConfig) (namedSyncerConfig, error) {
	if len(biddersSyncerConfig) == 1 {
		return biddersSyncerConfig[0], nil
	}

	var (
		bidderNames, bidderNamesWithEndpoints  []string
		nonAliasBidderNames, parentBidderNames []string
		nonAliasBidderNameMap                  = make(map[string]struct{}) // needed for O(1) lookup
		syncerConfig                           namedSyncerConfig
	)

	for _, bidder := range biddersSyncerConfig {
		bidderNames = append(bidderNames, bidder.name)
		if bidder.cfg.IFrame != nil || bidder.cfg.Redirect != nil {
			bidderNamesWithEndpoints = append(bidderNamesWithEndpoints, bidder.name)
			syncerConfig = bidder

			if len(bidder.bidderInfo.AliasOf) == 0 {
				nonAliasBidderNames = append(nonAliasBidderNames, bidder.name)
				nonAliasBidderNameMap[bidder.name] = struct{}{}
			} else {
				parentBidderNames = append(parentBidderNames, bidder.bidderInfo.AliasOf)
			}
		}
	}

	if len(bidderNamesWithEndpoints) == 0 {
		sort.Strings(bidderNames)
		bidders := strings.Join(bidderNames, ", ")
		return namedSyncerConfig{}, fmt.Errorf("bidders %s share the same syncer key, but none define endpoints (iframe and/or redirect)", bidders)
	}

	if len(nonAliasBidderNames) > 1 {
		sort.Strings(nonAliasBidderNames)
		bidders := strings.Join(nonAliasBidderNames, ", ")
		return namedSyncerConfig{}, fmt.Errorf("bidders %s define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints", bidders)
	}

	// invalidAliases - stores alias whose syncer key conflicts with bidder other than their parent
	invalidAliases := []string{}
	for _, bidderName := range parentBidderNames {
		if _, ok := nonAliasBidderNameMap[bidderName]; !ok {
			invalidAliases = append(invalidAliases, bidderName)
		}
	}
	if len(invalidAliases) > 0 {
		sort.Strings(invalidAliases)
		return namedSyncerConfig{}, fmt.Errorf("found aliases whose syncer key conflicts with a bidder other than their parent, aliases: %s", strings.Join(invalidAliases, ", "))
	}

	return syncerConfig, nil
}

func getSyncerKey(bidderInfos config.BidderInfos, bidderName string, bidderInfo config.BidderInfo, bidderSyncerCfg config.Syncer) (string, error) {
	// use key from syncer config if,
	// 1. an alias or non-alias bidder has defined key in their syncer config
	// 2. parent has defined key in their alias config and alias has inherited parent syncer info
	if bidderSyncerCfg.Key != "" {
		return bidderSyncerCfg.Key, nil
	}

	if len(bidderInfo.AliasOf) > 0 {
		parentBidderInfo, ok := bidderInfos[bidderInfo.AliasOf]
		if !ok {
			return "", fmt.Errorf("parent bidder %s not found", bidderInfo.AliasOf)
		}

		// alias bidder has inherited syncer info from parent therefore use parent bidder name as syncer key
		if parentBidderInfo.Syncer == bidderInfo.Syncer {
			return bidderInfo.AliasOf, nil
		}
	}

	// use bidder name as syncer key if,
	// 1. bidder is not an alias and bidder has no key defined in syncer config
	// 2. bidder is an alias but syncer config is different from parent
	return bidderName, nil
}
