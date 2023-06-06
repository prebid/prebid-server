package config

import (
	"context"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/gdpr"
)

// EnabledModuleLogger satisfies the AnalyticsLogger interface
type EnabledModuleLogger struct {
	modules       EnabledAnalytics
	privacyPolicy gdpr.PrivacyPolicy
	ctx           context.Context
}

// NewEnabledModuleLogger creates an instance of EnabledModuleLogger with an allow all privacy policy
func NewEnabledModuleLogger(modules EnabledAnalytics, ctx context.Context) *EnabledModuleLogger {
	return &EnabledModuleLogger{
		modules:       modules,
		privacyPolicy: &gdpr.AllowAllAnalytics{},
		ctx:           ctx,
	}
}

// SetPrivacyPolicy sets the privacy policy on the logger
func (eml *EnabledModuleLogger) SetPrivacyPolicy(pp gdpr.PrivacyPolicy) {
	eml.privacyPolicy = pp
}

// SetContext sets the context on the logger
func (eml *EnabledModuleLogger) SetContext(ctx context.Context) {
	eml.ctx = ctx
}

// LogAuctionObject satisfies the AnalyticsLogger interface. The enabled analytics modules that are
// allowed to receive data in accordance with the privacy policy are fed auction endpoint data.
func (eml *EnabledModuleLogger) LogAuctionObject(ao *analytics.AuctionObject) {
	for _, module := range eml.modules {
		allow, err := eml.privacyPolicy.Allow(eml.ctx, module.GetName(), module.GetVendorID())
		if allow && err == nil {
			module.LogAuctionObject(ao)
		}	
	}
}

// LogVideoObject satisfies the AnalyticsLogger interface. The enabled analytics modules that are
// allowed to receive data in accordance with the privacy policy are fed video endpoint data.
func (eml *EnabledModuleLogger) LogVideoObject(vo *analytics.VideoObject) {
	for _, module := range eml.modules {
		allow, err := eml.privacyPolicy.Allow(eml.ctx, module.GetName(), module.GetVendorID())
		if allow && err == nil {
			module.LogVideoObject(vo)
		}	
	}
}

// LogCookieSyncObject satisfies the AnalyticsLogger interface. The enabled analytics modules that are
// allowed to receive data in accordance with the privacy policy are fed cookie sync endpoint data.
func (eml *EnabledModuleLogger) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	for _, module := range eml.modules {
		allow, err := eml.privacyPolicy.Allow(eml.ctx, module.GetName(), module.GetVendorID())
		if allow && err == nil {
			module.LogCookieSyncObject(cso)
		}	
	}
}

// LogCookieSyncObject satisfies the AnalyticsLogger interface. The enabled analytics modules that are
// allowed to receive data in accordance with the privacy policy are fed setuid endpoint data.
func (eml *EnabledModuleLogger) LogSetUIDObject(so *analytics.SetUIDObject) {
	for _, module := range eml.modules {
		allow, err := eml.privacyPolicy.Allow(eml.ctx, module.GetName(), module.GetVendorID())
		if allow && err == nil {
			module.LogSetUIDObject(so)
		}	
	}
}

// LogAmpObject satisfies the AnalyticsLogger interface. The enabled analytics modules that are
// allowed to receive data in accordance with the privacy policy are fed amp endpoint data.
func (eml *EnabledModuleLogger) LogAmpObject(ao *analytics.AmpObject) {
	for _, module := range eml.modules {
		allow, err := eml.privacyPolicy.Allow(eml.ctx, module.GetName(), module.GetVendorID())
		if allow && err == nil {
			module.LogAmpObject(ao)
		}	
	}
}

// LogNotificationEventObject satisfies the AnalyticsLogger interface. The enabled analytics
// modules that are allowed to receive data in accordance with the privacy policy are fed
// event endpoint data.
func (eml *EnabledModuleLogger) LogNotificationEventObject(ne *analytics.NotificationEvent) {
	for _, module := range eml.modules {
		allow, err := eml.privacyPolicy.Allow(eml.ctx, module.GetName(), module.GetVendorID())
		if allow && err == nil {
			module.LogNotificationEventObject(ne)
		}	
	}
}