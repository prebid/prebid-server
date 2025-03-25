package mile

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/glog"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/analytics/mile/eventchannel"
	"github.com/prebid/prebid-server/v3/analytics/mile/helpers"
)

type Configuration struct {
	ScopeID  string          `json:"scopeId"`
	Endpoint string          `json:"endpoint"`
	Features map[string]bool `json:"features"`
}

// routes for events
const (
	auction    = "auction"
	cookieSync = "cookiesync"
	amp        = "amp"
	setUID     = "setuid"
	video      = "video"
)

type bufferConfig struct {
	timeout time.Duration
	count   int64
	size    int64
}

type MileModule struct {
	eventChannels map[string]*eventchannel.EventChannel
	httpClient    *http.Client
	sigTermCh     chan os.Signal
	stopCh        chan struct{}
	scope         string
	cfg           *Configuration
	buffsCfg      *bufferConfig
	muxConfig     sync.RWMutex
	clock         clock.Clock
}

func NewModule(client *http.Client, scope, endpoint, configRefreshDelay string, maxEventCount int, maxByteSize, maxTime string, clock clock.Clock) (analytics.Module, error) {
	configUpdateTask, err := NewConfigUpdateHttpTask(
		client,
		scope,
		endpoint,
		configRefreshDelay)
	if err != nil {
		return nil, err
	}

	return NewModuleWithConfigTask(client, scope, endpoint, maxEventCount, maxByteSize, maxTime, configUpdateTask, clock)
}

func loadSentry() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://52ba2fbdf01d4f7c8921ee775f05cb06@sentry.mile.so/13",
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		TracesSampleRate: 0.01,
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}

	defer sentry.Flush(2 * time.Second)

}

func NewModuleWithConfig(client *http.Client, scope, endpoint string, config *Configuration, maxEventCount int, maxByteSize, maxTime string, clock clock.Clock) (analytics.Module, error) {

	bufferCfg, err := newBufferConfig(maxEventCount, maxByteSize, maxTime)

	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubstack.buffers, :%v", err)
	}
	mm := MileModule{
		scope:         scope,
		httpClient:    client,
		cfg:           config,
		buffsCfg:      bufferCfg,
		sigTermCh:     make(chan os.Signal),
		stopCh:        make(chan struct{}),
		eventChannels: make(map[string]*eventchannel.EventChannel),
		muxConfig:     sync.RWMutex{},
		clock:         clock,
	}

	mm.updateConfig(config)
	loadSentry()

	return &mm, nil
}

func NewModuleWithConfigTask(client *http.Client, scope, endpoint string, maxEventCount int, maxByteSize, maxTime string, configTask ConfigUpdateTask, clock clock.Clock) (analytics.Module, error) {
	glog.Infof("[mile] Initializing module scope=%s endpoint=%s\n", scope, endpoint)

	// parse args
	bufferCfg, err := newBufferConfig(maxEventCount, maxByteSize, maxTime)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubstack.buffers, :%v", err)
	}

	defaultFeatures := map[string]bool{
		auction:    false,
		video:      false,
		amp:        false,
		cookieSync: false,
		setUID:     false,
	}

	defaultConfig := &Configuration{
		ScopeID:  scope,
		Endpoint: endpoint,
		Features: defaultFeatures,
	}

	mm := MileModule{
		scope:         scope,
		httpClient:    client,
		cfg:           defaultConfig,
		buffsCfg:      bufferCfg,
		sigTermCh:     make(chan os.Signal),
		stopCh:        make(chan struct{}),
		eventChannels: make(map[string]*eventchannel.EventChannel),
		muxConfig:     sync.RWMutex{},
		clock:         clock,
	}

	signal.Notify(mm.sigTermCh, os.Interrupt, syscall.SIGTERM)

	configChannel := configTask.Start(mm.stopCh)
	go mm.start(configChannel)

	glog.Info("[mile] Mile analytics configured and ready")
	return &mm, nil
}

func (m *MileModule) LogAuctionObject(ao *analytics.AuctionObject) {
	m.muxConfig.RLock()
	defer m.muxConfig.RUnlock()

	if !m.isFeatureEnable(auction) {
		return
	}

	// serialize event
	events, err := helpers.JsonifyAuctionObject(ao, m.scope)

	if err != nil {
		glog.Warning("[mile] Cannot serialize auction")
		sentry.CaptureException(err)
		return
	}
	for _, event := range events {
		m.eventChannels[auction].Push(&event)
	}
}

func (m *MileModule) LogNotificationEventObject(ne *analytics.NotificationEvent) {
}

func (m *MileModule) LogVideoObject(vo *analytics.VideoObject) {
	m.muxConfig.RLock()
	defer m.muxConfig.RUnlock()

	if !m.isFeatureEnable(video) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyVideoObject(vo, m.scope)
	if err != nil {
		glog.Warning("[mile] Cannot serialize video")
		return
	}

	m.eventChannels[video].Push(payload)
}

func (m *MileModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	m.muxConfig.RLock()
	defer m.muxConfig.RUnlock()

	if !m.isFeatureEnable(setUID) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifySetUIDObject(so, m.scope)
	if err != nil {
		glog.Warning("[mile] Cannot serialize video")
		return
	}

	m.eventChannels[setUID].Push(payload)
}

func (m *MileModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	m.muxConfig.RLock()
	defer m.muxConfig.RUnlock()

	if !m.isFeatureEnable(cookieSync) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyCookieSync(cso, m.scope)
	if err != nil {
		glog.Warning("[mile] Cannot serialize video")
		return
	}

	m.eventChannels[cookieSync].Push(payload)
}

func (m *MileModule) LogAmpObject(ao *analytics.AmpObject) {
	defer sentry.Recover()
	m.muxConfig.RLock()
	defer m.muxConfig.RUnlock()

	if !m.isFeatureEnable(amp) {
		return
	}

	// serialize event
	events, err := helpers.JsonifyAmpObject(ao, m.scope)
	if err != nil {
		sentry.CaptureException(err)
	}

	for _, event := range events {
		m.eventChannels[amp].Push(&event)
	}
}

func (m *MileModule) start(c <-chan *Configuration) {
	for {
		select {
		case <-m.sigTermCh:
			close(m.stopCh)
			cfg := m.cfg.clone().disableAllFeatures()
			m.updateConfig(cfg)
			return
		case config := <-c:
			m.updateConfig(config)
			glog.Infof("[mile] Updating config: %v", m.cfg)
		}
	}
}

func (m *MileModule) updateConfig(config *Configuration) {
	m.muxConfig.Lock()
	defer m.muxConfig.Unlock()

	//if m.cfg.isSameAs(config) {
	//	return
	//}

	m.cfg = config
	m.closeAllEventChannels()

	m.registerChannel(amp)
	m.registerChannel(auction)
	m.registerChannel(cookieSync)
	m.registerChannel(video)
	m.registerChannel(setUID)

}

func (m *MileModule) isFeatureEnable(feature string) bool {
	val, ok := m.cfg.Features[feature]
	return ok && val
}

func (m *MileModule) registerChannel(feature string) {
	if m.isFeatureEnable(feature) {
		sender := eventchannel.BuildEndpointSender(m.httpClient, m.cfg.Endpoint, feature)
		m.eventChannels[feature] = eventchannel.NewEventChannel(sender, m.clock, m.buffsCfg.size, m.buffsCfg.count, m.buffsCfg.timeout)
	}
}

func (m *MileModule) closeAllEventChannels() {
	for key, ch := range m.eventChannels {
		ch.Close()
		delete(m.eventChannels, key)
	}
}

func (p *MileModule) Shutdown() {
	glog.Info("[MileModule] Shutdown")
}
