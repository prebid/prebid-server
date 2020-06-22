package pubstack

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	units "github.com/docker/go-units"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics/pubstack/eventchannel"
	"github.com/prebid/prebid-server/analytics/pubstack/helpers"

	"github.com/prebid/prebid-server/analytics"
)

type payload struct {
	request  openrtb.BidRequest
	response openrtb.BidResponse
}

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
	setUiud    = "setuid"
	VIDEO      = "video"
)

type bufferConfig struct {
	timeout time.Duration
	count   int64
	size    int64
}

func newBufferConfig(count int, size, duration string) (*bufferConfig, error) {
	pDuration, err := time.ParseDuration(duration)
	if err != nil {
		return nil, err
	}
	pSize, err := units.FromHumanSize(size)
	if err != nil {
		return nil, err
	}
	return &bufferConfig{
		pDuration,
		int64(count),
		pSize,
	}, nil
}

type PubstackModule struct {
	eventChannels map[string]*eventchannel.EventChannel
	scope         string
	cfg           *Configuration
	buffsCfg      *bufferConfig
}

func (p *PubstackModule) applyConfiguration(cfg *Configuration) {
	newEventChannelMap := make(map[string]*eventchannel.EventChannel)

	if cfg.Features[amp] {
		newEventChannelMap[amp] = eventchannel.NewEventChannel(cfg.Endpoint, amp, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[auction] {
		newEventChannelMap[auction] = eventchannel.NewEventChannel(cfg.Endpoint, auction, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[cookieSync] {
		newEventChannelMap[cookieSync] = eventchannel.NewEventChannel(cfg.Endpoint, cookieSync, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[VIDEO] {
		newEventChannelMap[VIDEO] = eventchannel.NewEventChannel(cfg.Endpoint, VIDEO, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[setUiud] {
		newEventChannelMap[setUiud] = eventchannel.NewEventChannel(cfg.Endpoint, setUiud, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}

	p.eventChannels = newEventChannelMap
	p.cfg = cfg
}

func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	// check if we have to send auctions events
	ch, ok := p.eventChannels[auction]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize auction")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	// check if we have to send auctions events
	ch, ok := p.eventChannels[VIDEO]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyVideoObject(vo, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	// check if we have to send auctions events
	ch, ok := p.eventChannels[setUiud]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifySetUIDObject(so, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	// check if we have to send auctions events
	ch, ok := p.eventChannels[cookieSync]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyCookieSync(cso, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
	// check if we have to send auctions events
	ch, ok := p.eventChannels[amp]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAmpObject(ao, p.scope)
	if err != nil {
		glog.Warning("[pubstack] Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) refreshConfiguration(waitPeriod time.Duration, end chan os.Signal) {
	tick := time.NewTicker(waitPeriod)

	for {
		select {
		case <-tick.C:
			config, err := getConfiguration(p.cfg.ScopeId, p.cfg.Endpoint)
			if err != nil {
				glog.Error("[pubstack] Fail to update configuration")
				continue
			}
			p.applyConfiguration(config)
		case <-end:
			return
		}
	}
}

func getConfiguration(scope string, intake string) (*Configuration, error) {
	u, err := url.Parse(intake)
	if err != nil {
		return nil, err
	}

	u.Path = path.Join(u.Path, "bootstrap")
	q, _ := url.ParseQuery(u.RawQuery)

	q.Add("scopeId", scope)
	u.RawQuery = q.Encode()

	res, err := http.DefaultClient.Get(u.String())
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	c := Configuration{}
	err = json.NewDecoder(res.Body).Decode(c)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func parseBuffersConfiguration(size, duration string) (int64, *time.Duration, error) {
	u, err := units.FromHumanSize(size)
	if err != nil {
		return 0, nil, err
	}

	pdur, err := time.ParseDuration(duration)
	if err != nil {
		return 0, nil, err
	}

	return u, &pdur, nil
}

func NewPubstackModule(scope, intake, refreshConf string, evtCount int, size, duration string) (analytics.PBSAnalyticsModule, error) {
	glog.Infof("[pubstack] Initializing pubstack module with scope: %s intake %s\n", scope, intake)

	refreshDelay, err := time.ParseDuration(refreshConf)
	if err != nil {
		glog.Error("[pubstack] Fail to read configuration refresh duration")
		return nil, err
	}

	config, err := getConfiguration(scope, intake)
	if err != nil {
		glog.Errorf("[pubstack] Fail to initialize pubstack module, fail to acquire configuration\n")
		return nil, err
	}

	bufferCfg, err := newBufferConfig(evtCount, size, duration)

	pb := PubstackModule{
		scope:    scope,
		cfg:      config,
		buffsCfg: bufferCfg,
	}

	pb.applyConfiguration(config)

	// handle termination in goroutine
	endCh := make(chan os.Signal)
	signal.Notify(endCh, os.Interrupt, syscall.SIGTERM)
	glog.Info("[pubstack] Pubstack analytics configured and ready")
	go pb.refreshConfiguration(refreshDelay, endCh)

	return &pb, nil
}
