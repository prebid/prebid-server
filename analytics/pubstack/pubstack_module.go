package pubstack

import (
	"fmt"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"net/http"
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
	ScopeId  string          `json:"scopeId"`
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
	scope         string
	cfg           *Configuration
	buffsCfg      *bufferConfig
	mux           sync.Mutex
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
		ScopeId:  scope,
		Endpoint: endpoint,
		Features: defaultFeatures,
	}

	pb := PubstackModule{
		scope:         scope,
		httpClient:    client,
		cfg:           defaultConfig,
		buffsCfg:      bufferCfg,
		configCh:      make(chan *Configuration),
		eventChannels: make(map[string]*eventchannel.EventChannel),
	}
	endCh := make(chan os.Signal)
	signal.Notify(endCh, os.Interrupt, syscall.SIGTERM)

	go pb.setup()
	go pb.start(refreshDelay, endCh)

	glog.Info("[pubstack] Pubstack analytics configured and ready")
	return &pb, nil
}

func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	p.mux.Lock()
	defer p.mux.Unlock()

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

func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	p.mux.Lock()
	defer p.mux.Unlock()

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
	p.mux.Lock()
	defer p.mux.Unlock()

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
	p.mux.Lock()
	defer p.mux.Unlock()

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
	p.mux.Lock()
	defer p.mux.Unlock()

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

func (p *PubstackModule) setup() {
	config, err := fetchConfig(p.httpClient, p.cfg.ScopeId, p.cfg.Endpoint)
	if err != nil {
		glog.Errorf("[pubstack] Fail to fetch remote configuration: %v", err)
		return
	}
	p.configCh <- config
}

func (p *PubstackModule) start(refreshDelay time.Duration, endCh chan os.Signal) {

	// update periodically the config
	go p.fetchAndUpdateConfig(refreshDelay, endCh)

	for {
		select {
		case config := <-p.configCh:
			p.configure(config)
			glog.Infof("[pubstack] Updating config: %v", p.cfg)
		case <-endCh:
			return
		}
	}

}

func (p *PubstackModule) configure(config *Configuration) {
	p.mux.Lock()
	defer p.mux.Unlock()

	p.cfg = config

	// close previous instance
	for key, ch := range p.eventChannels {
		ch.Close()
		delete(p.eventChannels, key)
	}

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

func (p *PubstackModule) isFeatureEnable(feature string) bool {
	val, ok := p.cfg.Features[feature]
	return ok && val
}
