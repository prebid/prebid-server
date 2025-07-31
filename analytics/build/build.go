package build

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/ortb"
	"github.com/prebid/prebid-server/v3/privacy"
)

// Modules that need to be logged to need to be initialized here
func New(analyticsConfig *config.Analytics) analytics.Runner {
	modules := make(enabledAnalytics)

	// Enable host-level modules
	for name, builder := range moduleRegistry {
		if cfg, exists := analyticsConfig.Modules[name]; exists {
			module, err := builder.Build(cfg)
			if err != nil {
				glog.Errorf("Failed to initialize analytics module %s: %v", name, err)
				continue
			}
			modules[name] = module
		}
	}

	return modules
}

// Collection of all the correctly configured analytics modules - implements the PBSAnalyticsModule interface
type enabledAnalytics map[string]analytics.Module

func (ea enabledAnalytics) LogAuctionObject(ao *analytics.AuctionObject, ac privacy.ActivityControl) {
	account := ao.Account
	accountModules := account.AnalyticsModules

	for name, module := range ea {
		// check if there is a specific configuration for the account
		if accountModules != nil {
			if config, exists := accountModules[name]; exists {
				// if there is a specific configuration, initialize the module with it
				accountSpecificModule, err := initializeAccountSpecificModule(name, config)
				if err != nil {
					glog.Errorf("Failed to initialize account-specific module %s: %v", name, err)
					continue
				}
				accountSpecificModule.LogAuctionObject(ao)
				continue
			}
		}

		// if there is no specific configuration, use the default module
		if isAllowed, cloneBidderReq := evaluateActivities(ao.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			cloneReq := updateReqWrapperForAnalytics(ao.RequestWrapper, name, cloneBidderReq != nil)
			module.LogAuctionObject(ao)
			if cloneReq != nil {
				ao.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea enabledAnalytics) LogAuctionObjectWithCriteria(ao *analytics.AuctionObject, criteria LogCriteria) {
	combinedAnalytics := combineAnalytics(criteria.HostConfig, criteria.AccountConfig)

	for name, module := range combinedAnalytics {
		if isAllowed, cloneBidderReq := evaluateActivities(ao.RequestWrapper, criteria.Privacy, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			module.LogAuctionObject(ao)
		}
	}
}

func (ea enabledAnalytics) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	for _, module := range ea {
		module.LogCookieSyncObject(cso)
	}
}

func (ea enabledAnalytics) LogSetUIDObject(so *analytics.SetUIDObject) {
	for _, module := range ea {
		module.LogSetUIDObject(so)
	}
}

func (ea enabledAnalytics) LogAmpObject(ao *analytics.AmpObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := evaluateActivities(ao.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				ao.RequestWrapper = cloneBidderReq
			}
			cloneReq := updateReqWrapperForAnalytics(ao.RequestWrapper, name, cloneBidderReq != nil)
			module.LogAmpObject(ao)
			if cloneReq != nil {
				ao.RequestWrapper = cloneReq
			}
		}
	}
}

func (ea enabledAnalytics) LogNotificationEventObject(ne *analytics.NotificationEvent, ac privacy.ActivityControl) {
	for name, module := range ea {
		component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: name}
		if ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
			module.LogNotificationEventObject(ne)
		}
	}
}

// LogVideoObject implements the missing method for analytics.Runner.
func (ea enabledAnalytics) LogVideoObject(vo *analytics.VideoObject, ac privacy.ActivityControl) {
	for name, module := range ea {
		if isAllowed, cloneBidderReq := evaluateActivities(vo.RequestWrapper, ac, name); isAllowed {
			if cloneBidderReq != nil {
				vo.RequestWrapper = cloneBidderReq
			}
			module.LogVideoObject(vo)
		}
	}
}

// Shutdown - correctly shutdown all analytics modules and wait for them to finish
func (ea enabledAnalytics) Shutdown() {
	for _, module := range ea {
		module.Shutdown()
	}
}

func evaluateActivities(rw *openrtb_ext.RequestWrapper, ac privacy.ActivityControl, componentName string) (bool, *openrtb_ext.RequestWrapper) {
	// returned nil request wrapper means that request wrapper was not modified by activities and doesn't have to be changed in analytics object
	// it is needed in order to use one function for all analytics objects with RequestWrapper
	component := privacy.Component{Type: privacy.ComponentTypeAnalytics, Name: componentName}
	if !ac.Allow(privacy.ActivityReportAnalytics, component, privacy.ActivityRequest{}) {
		return false, nil
	}
	blockUserFPD := !ac.Allow(privacy.ActivityTransmitUserFPD, component, privacy.ActivityRequest{})
	blockPreciseGeo := !ac.Allow(privacy.ActivityTransmitPreciseGeo, component, privacy.ActivityRequest{})

	if !blockUserFPD && !blockPreciseGeo {
		return true, nil
	}

	cloneReq := &openrtb_ext.RequestWrapper{
		BidRequest: ortb.CloneBidRequestPartial(rw.BidRequest),
	}

	if blockUserFPD {
		privacy.ScrubUserFPD(cloneReq)
	}
	if blockPreciseGeo {
		ipConf := privacy.IPConf{IPV6: ac.IPv6Config, IPV4: ac.IPv4Config}
		privacy.ScrubGeoAndDeviceIP(cloneReq, ipConf)
	}

	cloneReq.RebuildRequest()
	return true, cloneReq
}

func updateReqWrapperForAnalytics(rw *openrtb_ext.RequestWrapper, adapterName string, isCloned bool) *openrtb_ext.RequestWrapper {
	if rw == nil {
		return nil
	}
	reqExt, _ := rw.GetRequestExt()
	reqExtPrebid := reqExt.GetPrebid()
	if reqExtPrebid == nil {
		return nil
	}

	var cloneReq *openrtb_ext.RequestWrapper
	if !isCloned {
		cloneReq = &openrtb_ext.RequestWrapper{BidRequest: ortb.CloneBidRequestPartial(rw.BidRequest)}
	} else {
		cloneReq = nil
	}

	if len(reqExtPrebid.Analytics) == 0 {
		return cloneReq
	}

	// Remove the entire analytics object if the adapter module is not present
	if _, ok := reqExtPrebid.Analytics[adapterName]; !ok {
		reqExtPrebid.Analytics = nil
	} else {
		reqExtPrebid.Analytics = updatePrebidAnalyticsMap(reqExtPrebid.Analytics, adapterName)
	}
	reqExt.SetPrebid(reqExtPrebid)
	rw.RebuildRequest()

	if cloneReq != nil {
		cloneReq.RebuildRequest()
	}

	return cloneReq
}

func updatePrebidAnalyticsMap(extPrebidAnalytics map[string]json.RawMessage, adapterName string) map[string]json.RawMessage {
	newMap := make(map[string]json.RawMessage)
	if val, ok := extPrebidAnalytics[adapterName]; ok {
		newMap[adapterName] = val
	}
	return newMap
}

func (ea enabledAnalytics) getOrInitializeModule(name string, config json.RawMessage) (analytics.Module, error) {
	// Sprawdź, czy adapter już istnieje
	if module, exists := ea[name]; exists {
		return module, nil
	}

	// Jeśli nie istnieje, zainicjalizuj go dynamicznie
	module, err := initializeAccountSpecificModule(name, config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize module %s: %v", name, err)
	}

	// Dodaj do aktywnych adapterów
	ea[name] = module
	return module, nil
}

func (ea enabledAnalytics) getOrEnableModule(name string, config json.RawMessage) (analytics.Module, error) {
	if module, exists := ea[name]; exists {
		return module, nil
	}

	builder, exists := moduleRegistry[name]
	if !exists {
		return nil, fmt.Errorf("analytics module %s is not registered", name)
	}

	module, err := builder.Build(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize analytics module %s: %v", name, err)
	}

	ea[name] = module
	return module, nil
}

func combineAnalytics(hostConfig enabledAnalytics, accountConfig map[string]json.RawMessage) enabledAnalytics {
	combined := make(enabledAnalytics)

	// Dodaj adaptery z host config
	for name, module := range hostConfig {
		combined[name] = module
	}

	// Nadpisz/uzupełnij adaptery z account config
	for name, config := range accountConfig {
		if _, exists := combined[name]; !exists {
			// Lazy loading adaptera
			if module, err := combined.getOrInitializeModule(name, config); err == nil {
				combined[name] = module
			} else {
				glog.Errorf("Error initializing module %s: %v", name, err)
			}
		}
	}

	return combined
}

func initializeAccountSpecificModule(name string, config json.RawMessage) (analytics.Module, error) {
	switch name {
	case "greenbids":
		var greenbidsConfig struct {
			Enabled           bool    `json:"enabled"`
			PubID             string  `json:"pubid"`
			GreenbidsSampling float64 `json:"greenbidsSampling"`
		}
		if err := json.Unmarshal(config, &greenbidsConfig); err != nil {
			return nil, fmt.Errorf("invalid configuration for greenbids: %v", err)
		}
		if greenbidsConfig.Enabled {
			return greenbids.NewModule(greenbidsConfig.PubID, greenbidsConfig.GreenbidsSampling), nil
		}
	// here we can add more cases for other analytics modules
	default:
		return nil, fmt.Errorf("unknown analytics module: %s", name)
	}
	return nil, nil
}

type LogCriteria struct {
	AccountConfig map[string]json.RawMessage
	HostConfig    enabledAnalytics
	Privacy       privacy.ActivityControl
}
