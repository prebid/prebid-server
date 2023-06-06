package config

import (
	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/analytics/clients"
	"github.com/prebid/prebid-server/analytics/filesystem"
	"github.com/prebid/prebid-server/analytics/pubstack"
	"github.com/prebid/prebid-server/config"
)

// EnalbedAnalytics is a collection of objects that implement the PBSAnalyticsModule interface
type EnabledAnalytics []analytics.PBSAnalyticsModule

// NewPBSAnalytics creates a slice containing all enabled analytics modules
func NewPBSAnalytics(analytics *config.Analytics) EnabledAnalytics {
	modules := make(EnabledAnalytics, 0)
	if len(analytics.File.Filename) > 0 {
		if mod, err := filesystem.NewFileLogger(analytics.File.Filename); err == nil {
			modules = append(modules, mod)
		} else {
			glog.Fatalf("Could not initialize FileLogger for file %v :%v", analytics.File.Filename, err)
		}
	}

	if analytics.Pubstack.Enabled {
		pubstackModule, err := pubstack.NewModule(
			clients.GetDefaultHttpInstance(),
			analytics.Pubstack.ScopeId,
			analytics.Pubstack.IntakeUrl,
			analytics.Pubstack.ConfRefresh,
			analytics.Pubstack.Buffers.EventCount,
			analytics.Pubstack.Buffers.BufferSize,
			analytics.Pubstack.Buffers.Timeout,
			clock.New())
		if err == nil {
			modules = append(modules, pubstackModule)
		} else {
			glog.Errorf("Could not initialize PubstackModule: %v", err)
		}
	}
	return modules
}