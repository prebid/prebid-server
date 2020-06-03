package pubstack

import (
	"encoding/json"
	"errors"
	"io/ioutil"
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
	AUCTION    = "auction"
	COOKIESYNC = "cookiesync"
	AMP        = "amp"
	SETUID     = "setuid"
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
	chans    map[string]*eventchannel.Channel
	scope    string
	cfg      *Configuration
	buffsCfg *bufferConfig
}

func (p *PubstackModule) applyConfiguration(cfg *Configuration) {
	newChanMap := make(map[string]*eventchannel.Channel)

	if cfg.Features[AMP] {
		newChanMap[AMP] = eventchannel.NewChannel(cfg.Endpoint, AMP, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[AUCTION] {
		newChanMap[AUCTION] = eventchannel.NewChannel(cfg.Endpoint, AUCTION, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[COOKIESYNC] {
		newChanMap[COOKIESYNC] = eventchannel.NewChannel(cfg.Endpoint, COOKIESYNC, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[VIDEO] {
		newChanMap[VIDEO] = eventchannel.NewChannel(cfg.Endpoint, VIDEO, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}
	if cfg.Features[SETUID] {
		newChanMap[SETUID] = eventchannel.NewChannel(cfg.Endpoint, SETUID, p.buffsCfg.size, p.buffsCfg.count, p.buffsCfg.timeout)
	}

	p.chans = newChanMap
	p.cfg = cfg
}

func (p *PubstackModule) LogAuctionObject(ao *analytics.AuctionObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[AUCTION]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAuctionObject(ao, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize auction")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogVideoObject(vo *analytics.VideoObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[VIDEO]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyVideoObject(vo, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogSetUIDObject(so *analytics.SetUIDObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[SETUID]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifySetUIDObject(so, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[COOKIESYNC]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyCookieSync(cso, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
		return
	}

	ch.Add(payload)
}

func (p *PubstackModule) LogAmpObject(ao *analytics.AmpObject) {
	// check if we have to send auctions events
	ch, ok := p.chans[AMP]
	if !ok {
		return
	}

	// serialize event
	payload, err := helpers.JsonifyAmpObject(ao, p.scope)
	if err != nil {
		glog.Warning("Cannot serialize video")
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
				glog.Error("fail to update configuration")
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
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New("fail to read payload body")
	}
	c := Configuration{}

	err = json.Unmarshal(data, &c)
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
	glog.Infof("Initializing pubstack module with scope: %s intake %s\n", scope, intake)

	refreshDelay, err := time.ParseDuration(refreshConf)
	if err != nil {
		glog.Error("Fail to read configuration refresh duration")
		return nil, err
	}

	config, err := getConfiguration(scope, intake)
	if err != nil {
		glog.Errorf("Fail to initialize pubstack module, fail to acquire configuration\n")
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
	glog.Info("Pubstack analytics configured and ready")
	go pb.refreshConfiguration(refreshDelay, endCh)

	return &pb, nil
}
