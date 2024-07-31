package mile

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/glog"

	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/analytics/mile/eventchannel"
	"github.com/prebid/prebid-server/v2/analytics/mile/helpers"
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

func NewModuleWithConfigTask(client *http.Client, scope, endpoint string, maxEventCount int, maxByteSize, maxTime string, configTask ConfigUpdateTask, clock clock.Clock) (analytics.Module, error) {
	glog.Infof("[pubstack] Initializing module scope=%s endpoint=%s\n", scope, endpoint)

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

	pb := MileModule{
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

	signal.Notify(pb.sigTermCh, os.Interrupt, syscall.SIGTERM)

	configChannel := configTask.Start(pb.stopCh)
	go pb.start(configChannel)

	glog.Info("[pubstack] Pubstack analytics configured and ready")
	return &pb, nil
}

func (p *MileModule) LogAuctionObject(ao *analytics.AuctionObject) {
	p.muxConfig.RLock()
	defer p.muxConfig.RUnlock()

	if !p.isFeatureEnable(auction) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize auction")
		return
	}

	p.eventChannels[auction].Push(payload)
}

func (p *MileModule) LogNotificationEventObject(ne *analytics.NotificationEvent) {
}

func (p *MileModule) LogVideoObject(vo *analytics.VideoObject) {
	p.muxConfig.RLock()
	defer p.muxConfig.RUnlock()

	if !p.isFeatureEnable(video) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyVideoObject(vo, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.eventChannels[video].Push(payload)
}

func (p *MileModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	p.muxConfig.RLock()
	defer p.muxConfig.RUnlock()

	if !p.isFeatureEnable(setUID) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifySetUIDObject(so, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.eventChannels[setUID].Push(payload)
}

func (p *MileModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	p.muxConfig.RLock()
	defer p.muxConfig.RUnlock()

	if !p.isFeatureEnable(cookieSync) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyCookieSync(cso, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.eventChannels[cookieSync].Push(payload)
}

func (p *MileModule) LogAmpObject(ao *analytics.AmpObject) {
	p.muxConfig.RLock()
	defer p.muxConfig.RUnlock()

	if !p.isFeatureEnable(amp) {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAmpObject(ao, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	p.eventChannels[amp].Push(payload)
}

func (p *MileModule) start(c <-chan *Configuration) {
	for {
		select {
		case <-p.sigTermCh:
			close(p.stopCh)
			cfg := p.cfg.clone().disableAllFeatures()
			p.updateConfig(cfg)
			return
		case config := <-c:
			p.updateConfig(config)
			glog.Infof("[pubstack] Updating config: %v", p.cfg)
		}
	}
}

func (p *MileModule) updateConfig(config *Configuration) {
	p.muxConfig.Lock()
	defer p.muxConfig.Unlock()

	if p.cfg.isSameAs(config) {
		return
	}

	p.cfg = config
	p.closeAllEventChannels()

	p.registerChannel(amp)
	p.registerChannel(auction)
	p.registerChannel(cookieSync)
	p.registerChannel(video)
	p.registerChannel(setUID)
}

func (p *MileModule) isFeatureEnable(feature string) bool {
	val, ok := p.cfg.Features[feature]
	return ok && val
}

func (p *MileModule) registerChannel(feature string) {
	if p.isFeatureEnable(feature) {
		sender := eventchannel.BuildEndpointSender(p.httpClient, p.cfg.Endpoint, feature)
		p.eventChannels[feature] = eventchannel.NewEventChannel(sender, p.clock, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
}

func (p *MileModule) closeAllEventChannels() {
	for key, ch := range p.eventChannels {
		ch.Close()
		delete(p.eventChannels, key)
	}
}
