package pubstack

import (
	"fmt"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/analytics/pubstack/helpers"

	"github.com/prebid/prebid-server/analytics"
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

type PubstackModule struct {
	eventChannels map[string]*eventchannel.EventChannel
	httpClient    *http.Client
	configCh      chan *Configuration
	sigTermCh     chan os.Signal
	scope         string
	cfg           *Configuration
	buffsCfg      *bufferConfig
	muxConfig     sync.RWMutex
}

func NewPubstackModule(client *http.Client, scope, endpoint, configRefreshDelay string, maxEventCount int, maxByteSize, maxTime string) (analytics.PBSAnalyticsModule, error) {
	glog.Infof("[pubstack] Initializing module scope=%s endpoint=%s\n", scope, endpoint)

	// parse args

	refreshDelay, err := time.ParseDuration(configRefreshDelay)
	if err != nil {
		return nil, fmt.Errorf("fail to parse the module args, arg=analytics.pubstack.configuration_refresh_delay, :%v", err)
	}

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

	pb := PubstackModule{
		scope:         scope,
		httpClient:    client,
		cfg:           defaultConfig,
		buffsCfg:      bufferCfg,
		sigTermCh:     make(chan os.Signal),
		configCh:      make(chan *Configuration),
		eventChannels: make(map[string]*eventchannel.EventChannel),
		muxConfig:     sync.RWMutex{},
	}
	signal.Notify(pb.sigTermCh, os.Interrupt, syscall.SIGTERM)

	configUrl, err := url.Parse(pb.cfg.Endpoint + "/bootstrap?scopeId=" + pb.cfg.ScopeID)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	go pb.start(configUrl, refreshDelay)
	go func() {
		err = pb.reloadConfig(configUrl)
		if err != nil {
			glog.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
		}
	}()

	glog.Info("[pubstack] Pubstack analytics configured and ready")
	return &pb, nil
}

func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
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

func (p *PubstackModule) LogNotificationEventObject(ne *analytics.NotificationEvent) {
}

func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
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

func (p *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
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

func (p *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
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

func (p *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
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

func (p *PubstackModule) reloadConfig(configUrl *url.URL) error {
	config, err := fetchConfig(p.httpClient, configUrl)
	if err != nil {
		return err
	}
	p.configCh <- config
	return nil
}

func (p *PubstackModule) start(configUrl *url.URL, refreshDelay time.Duration) {

	tick := time.NewTicker(refreshDelay)

	for {
		select {
		case <-p.sigTermCh:
			p.closeAllEventChannels()
			return
		case config := <-p.configCh:
			p.updateConfig(config)
			glog.Infof("[pubstack] Updating config: %v", p.cfg)
		case <-tick.C:
			go func() {
				err := p.reloadConfig(configUrl)
				if err != nil {
					glog.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
				}
			}()
		}
	}

}

func (p *PubstackModule) updateConfig(config *Configuration) {
	p.muxConfig.Lock()
	defer p.muxConfig.Unlock()

	if p.cfg.isSameAs(config) {
		return
	}

	p.cfg = config
	p.closeAllEventChannels()

	if p.isFeatureEnable(amp) {
		p.eventChannels[amp] = eventchannel.NewEventChannel(eventchannel.BuildEndpointSender(p.httpClient, p.cfg.Endpoint, amp), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if p.isFeatureEnable(auction) {
		p.eventChannels[auction] = eventchannel.NewEventChannel(eventchannel.BuildEndpointSender(p.httpClient, p.cfg.Endpoint, auction), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if p.isFeatureEnable(cookieSync) {
		p.eventChannels[cookieSync] = eventchannel.NewEventChannel(eventchannel.BuildEndpointSender(p.httpClient, p.cfg.Endpoint, cookieSync), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if p.isFeatureEnable(video) {
		p.eventChannels[video] = eventchannel.NewEventChannel(eventchannel.BuildEndpointSender(p.httpClient, p.cfg.Endpoint, video), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if p.isFeatureEnable(setUID) {
		p.eventChannels[setUID] = eventchannel.NewEventChannel(eventchannel.BuildEndpointSender(p.httpClient, p.cfg.Endpoint, setUID), p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
}

func (p *PubstackModule) closeAllEventChannels() {
	for key, ch := range p.eventChannels {
		ch.Close()
		delete(p.eventChannels, key)
	}
}

func (p *PubstackModule) isFeatureEnable(feature string) bool {
	val, ok := p.cfg.Features[feature]
	return ok && val
}
