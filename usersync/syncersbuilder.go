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
	biddersWithSyncerCfg := make(map[string]config.BidderInfo, len(bidderInfos))
	for bidder, bidderInfo := range bidderInfos {
		if shouldCreateSyncer(bidderInfo) {
			biddersWithSyncerCfg[bidder] = bidderInfo
		}
	}

	// map syncer config by key
	cfgBySyncerKey := make(map[string][]namedSyncerConfig, len(bidderInfos))
	for bidder, bidderInfo := range biddersWithSyncerCfg {
		syncerCopy := *bidderInfo.Syncer
		var err error
		syncerCopy.Key, err = getSyncerKey(biddersWithSyncerCfg, bidder, bidderInfo)
		if err != nil {
			return nil, []error{err}
		}
		cfgBySyncerKey[syncerCopy.Key] = append(cfgBySyncerKey[syncerCopy.Key], namedSyncerConfig{bidder, syncerCopy, bidderInfo})
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
		bidderNames         []string
		nonAliasBidderNames []string
		aliasBidderNames    []string
		syncerConfig        namedSyncerConfig
		parentBidderNameMap = make(map[string]struct{})
	)

	for _, bidder := range biddersSyncerConfig {
		bidderNames = append(bidderNames, bidder.name)
		if bidder.cfg.IFrame != nil || bidder.cfg.Redirect != nil {
			if len(bidder.bidderInfo.AliasOf) > 0 {
				aliasBidderNames = append(aliasBidderNames, bidder.name)
				parentBidderNameMap[bidder.bidderInfo.AliasOf] = struct{}{}
			} else {
				nonAliasBidderNames = append(nonAliasBidderNames, bidder.name)
			}
			syncerConfig = bidder
		}
	}

	// check if bidders have same syncer key but no endpoints
	if len(nonAliasBidderNames)+len(aliasBidderNames) == 0 {
		sort.Strings(bidderNames)
		bidders := strings.Join(bidderNames, ", ")
		return namedSyncerConfig{}, fmt.Errorf("bidders %s share the same syncer key, but none define endpoints (iframe and/or redirect)", bidders)
	}

	// check if non-alias bidders have same syncer key
	if len(nonAliasBidderNames) > 1 {
		sort.Strings(nonAliasBidderNames)
		bidders := strings.Join(nonAliasBidderNames, ", ")
		return namedSyncerConfig{}, fmt.Errorf("bidders %s define endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints", bidders)
	}

	// check if alias bidders of different parent have same syncer key
	if len(parentBidderNameMap) > 1 {
		sort.Strings(aliasBidderNames)
		return namedSyncerConfig{}, fmt.Errorf("alias bidders %s of different parents defines endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints", strings.Join(aliasBidderNames, ", "))
	}

	// check if aliases of same parent and non-alias bidder have same syncer key
	if len(parentBidderNameMap) != 0 {
		if _, ok := parentBidderNameMap[nonAliasBidderNames[0]]; !ok {
			sort.Strings(aliasBidderNames)
			return namedSyncerConfig{}, fmt.Errorf("alias bidders %s and non-alias bidder %s defines endpoints (iframe and/or redirect) for the same syncer key, but only one bidder is permitted to define endpoints", strings.Join(aliasBidderNames, ", "), nonAliasBidderNames[0])
		}
	}

	return syncerConfig, nil
}

func getSyncerKey(biddersWithSyncerCfg map[string]config.BidderInfo, bidderName string, bidderInfo config.BidderInfo) (string, error) {
	if bidderInfo.Syncer == nil {
		return "", fmt.Errorf("found no syncer config for bidder %s", bidderName)
	}

	var (
		bidderSyncerCfg       = bidderInfo.Syncer
		isAlias               = len(bidderInfo.AliasOf) > 0
		hasInheritedSyncerCfg = false
		parentBidderInfo      config.BidderInfo
		parentBidderName      string
	)

	if isAlias {
		parentBidderName = bidderInfo.AliasOf
		var parentHasSyncerCfg bool
		parentBidderInfo, parentHasSyncerCfg = biddersWithSyncerCfg[parentBidderName]
		hasInheritedSyncerCfg = parentHasSyncerCfg && parentBidderInfo.Syncer == bidderSyncerCfg
	}

	if bidderSyncerCfg.Key == "" {
		if hasInheritedSyncerCfg {
			return parentBidderName, nil
		}
		return bidderName, nil
	}

	if isAlias && !hasInheritedSyncerCfg && bidderSyncerCfg.Key == parentBidderInfo.Syncer.Key {
		return "", fmt.Errorf("syncer key of alias bidder %s is same as the syncer key for its parent bidder %s", bidderName, parentBidderName)
	}

	return bidderSyncerCfg.Key, nil
}
