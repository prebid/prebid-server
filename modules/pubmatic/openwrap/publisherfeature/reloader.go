package publisherfeature

import (
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/cache"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
)

type Config struct {
	Cache                 cache.Cache
	DefaultExpiry         int
	AnalyticsThrottleList string
}

type feature struct {
	cache       cache.Cache
	serviceStop chan struct{}
	sync.RWMutex
	defaultExpiry           int
	publisherFeature        map[int]map[int]models.FeatureData
	fsc                     fsc
	tbf                     tbf
	ant                     analyticsThrottle
	ampMultiformat          ampMultiformat
	maxFloors               maxFloors
	bidRecovery             bidRecovery
	appLovinMultiFloors     appLovinMultiFloors
	impCountingMethod       impCountingMethod
	gdprCountryCodes        gdprCountryCodes
	mbmf                    *mbmf
	dynamicFloor            dynamicFloor
	performanceDSPs         performanceDSPs
	inViewEnabledPublishers inViewEnabledPublishers
	act                     act
	edsBlockedCountries     edsBlockedCountries
}

var fe *feature
var fOnce sync.Once

func New(config Config) *feature {
	fOnce.Do(func() {
		fe = &feature{
			cache:            config.Cache,
			serviceStop:      make(chan struct{}),
			defaultExpiry:    config.DefaultExpiry,
			publisherFeature: make(map[int]map[int]models.FeatureData),
			fsc: fsc{
				disabledPublishers: make(map[int]struct{}),
				thresholdsPerDsp:   make(map[int]int),
			},
			tbf: tbf{
				pubProfileTraffic: make(map[int]map[int]int),
			},
			ampMultiformat: ampMultiformat{
				enabledPublishers: make(map[int]struct{}),
			},
			maxFloors: maxFloors{
				enabledPublishers: make(map[int]struct{}),
			},
			ant: analyticsThrottle{
				vault: newPubThrottling(config.AnalyticsThrottleList),
				db:    newPubThrottling(config.AnalyticsThrottleList),
			},
			appLovinMultiFloors: appLovinMultiFloors{
				enabledPublisherProfile: make(map[int]map[string]models.ApplovinAdUnitFloors),
			},
			impCountingMethod:       newImpCountingMethod(),
			gdprCountryCodes:        newGDPRCountryCodes(),
			mbmf:                    newMBMF(),
			dynamicFloor:            newDynamicFloor(),
			performanceDSPs:         newPerformanceDSPs(),
			inViewEnabledPublishers: newInViewEnabledPublishers(),
			edsBlockedCountries:     newEDSBlockedCountries(),
		}
	})
	return fe
}

func (fe *feature) Start() {
	go initReloader(fe)
	glog.Info("Initialized feature reloader")
}

func (fe *feature) Stop() {
	//updating serviceStop flag to true
	close(fe.serviceStop)
}

// Initializing reloader with cache-refresh default-expiry + 30 mins (to avoid DB load post cache refresh)
var initReloader = func(fe *feature) {
	if fe.defaultExpiry <= 0 {
		return
	}
	glog.Info("Feature reloader start")
	ticker := time.NewTicker(time.Duration(fe.defaultExpiry+1800) * time.Second)
	for {
		//Populating feature config maps from cache
		fe.updateFeatureConfigMaps()
		//update gdprCountryCodes
		fe.updateGDPRCountryCodes()
		select {
		case t := <-ticker.C:
			glog.Info("Feature Reloader loads cache @", t)
		case <-fe.serviceStop:
			return
		}
	}
}

func (fe *feature) updateFeatureConfigMaps() {
	var err error

	publisherFeatureMap, errPubFeature := fe.cache.GetPublisherFeatureMap()
	if errPubFeature != nil {
		err = models.ErrorWrap(err, errPubFeature)
	}

	if publisherFeatureMap != nil {
		fe.publisherFeature = publisherFeatureMap
	}

	// Single cache/DB call for both FSC and ACT when possible (same logic: publisher enabled/disabled + DSP percentage threshold).
	if errFscActUpdate := fe.updateFscAndActConfigMapsFromCache(); errFscActUpdate != nil {
		err = models.ErrorWrap(err, errFscActUpdate)
	}

	fe.updateTBFConfigMap()
	fe.updateAmpMutiformatEnabledPublishers()
	fe.updateMaxFloorsEnabledPublishers()
	fe.updateAnalyticsThrottling()
	fe.updateBidRecoveryEnabledPublishers()
	fe.updateApplovinMultiFloorsFeature()
	fe.updateImpCountingMethodEnabledBidders()
	fe.updateMBMF()
	fe.updateDynamicFloorEnabledPublishers()
	fe.updatePerformanceDSPs()
	fe.updateInViewEnabledPublishers()
	fe.updateEDSBlockedCountries()

	if err != nil {
		glog.Error(err.Error())
	}
}
