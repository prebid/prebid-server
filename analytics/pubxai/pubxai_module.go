package pubxai

import (
	"errors"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/docker/go-units"
	"github.com/golang/glog"

	"github.com/prebid/prebid-server/v2/analytics"
)

func InitializePubxAIModule(client *http.Client, publisherId string, endpoint string, bufferInterval string, bufferSize string, SamplingPercentage int, configRefresh string, clock clock.Clock) (analytics.Module, error) {
	// print client, publisherId, endpoint, maxEventCount, clock
	glog.Infof("NewPubxAIModule: %v, %v, %v, %v", publisherId, endpoint, bufferInterval, bufferSize)
	if client == nil {
		return nil, errors.New("client cannot be nil")
	}

	if publisherId == "" {
		glog.Error("[pubxai] pubx publisherId cannot be empty when pubxai analytics is enabled")
		return nil, errors.New("pubx publisherId cannot be empty when pubxai analytics is enabled")
	}

	if endpoint == "" {
		glog.Error("[pubxai] pubx endpoint cannot be empty when pubxai analytics is enabled")
		return nil, errors.New("pubx endpoint cannot be empty when pubxai analytics is enabled")
	}

	_, err := time.ParseDuration(bufferInterval)
	if err != nil {
		glog.Error("[pubxai] Error parsing bufferInterval: %v", err)
		return nil, err
	}

	_, err = units.FromHumanSize(bufferSize)
	if err != nil {
		glog.Error("[pubxai] Error parsing bufferSize: %v", err)
		return nil, err
	}
	configUpdateTask, err := NewConfigUpdateHttpTask(
		client,
		publisherId,
		endpoint,
		configRefresh,
	)

	if err != nil {
		glog.Error("[pubxai] Error creating config update task: %v", err)
		return nil, err
	}

	defaultConfig := &Configuration{
		PublisherId:        publisherId,
		BufferInterval:     bufferInterval,
		BufferSize:         bufferSize,
		SamplingPercentage: SamplingPercentage,
	}

	pb := PubxaiModule{
		publisherId:      publisherId,
		endpoint:         endpoint,
		cfg:              defaultConfig,
		winBidsQueue:     NewBidQueue("win", endpoint+"/win", client, clock, bufferInterval, bufferSize),
		auctionBidsQueue: NewBidQueue("auction", endpoint+"/auction", client, clock, bufferInterval, bufferSize),
		httpClient:       client,
		clock:            clock,
		muxConfig:        sync.RWMutex{},
		sigTermCh:        make(chan os.Signal),
		stopCh:           make(chan struct{}),
	}

	signal.Notify(pb.sigTermCh, os.Interrupt, syscall.SIGTERM)

	configChannel := configUpdateTask.Start(pb.stopCh)
	go pb.start(configChannel)

	glog.Info("[pubxai] pubxai analytics configured and ready")
	return &pb, nil
}

func (p *PubxaiModule) start(c <-chan *Configuration) {
	for {
		select {
		case <-p.sigTermCh:
			close(p.stopCh)
			return
		case config := <-c:
			p.updateConfig(config)
			glog.Infof("[pubxai] Updating config: %v", p.cfg)
		}
	}
}

func (p *PubxaiModule) updateConfig(config *Configuration) {
	p.muxConfig.Lock()
	defer p.muxConfig.Unlock()

	if p.cfg.isSameAs(config) {
		return
	}
	p.cfg = config
	p.auctionBidsQueue.bufferInterval = config.BufferInterval
	p.auctionBidsQueue.bufferSize = config.BufferSize
	p.winBidsQueue.bufferInterval = config.BufferInterval
	p.winBidsQueue.bufferSize = config.BufferSize
}

func (p *PubxaiModule) LogAuctionObject(ao *analytics.AuctionObject) {
	if ao == nil {
		glog.Warning("Auction Object is nil")
		return
	}
	// Generate a random integer between 1 and 100
	randomNumber := rand.Intn(100) + 1
	if p.cfg.SamplingPercentage < randomNumber {
		return
	}
	p.processAuctionData(ao)
}

func (p *PubxaiModule) LogNotificationEventObject(ne *analytics.NotificationEvent) {
}

func (p *PubxaiModule) LogVideoObject(vo *analytics.VideoObject) {
}

func (p *PubxaiModule) LogSetUIDObject(so *analytics.SetUIDObject) {
}

func (p *PubxaiModule) LogCookieSyncObject(cso *analytics.CookieSyncObject) {
}

func (p *PubxaiModule) LogAmpObject(ao *analytics.AmpObject) {
}
