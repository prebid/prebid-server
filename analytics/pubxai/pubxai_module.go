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

	"github.com/prebid/prebid-server/v3/analytics"
	config "github.com/prebid/prebid-server/v3/analytics/pubxai/config"
	processor "github.com/prebid/prebid-server/v3/analytics/pubxai/processor"
	queue "github.com/prebid/prebid-server/v3/analytics/pubxai/queue"
	"github.com/prebid/prebid-server/v3/analytics/pubxai/utils"
)

type PubxaiModule struct {
	endpoint         string
	winBidsQueue     *queue.WinningBidQueue
	auctionBidsQueue *queue.AuctionBidsQueue
	httpClient       *http.Client
	muxConfig        sync.RWMutex
	clock            clock.Clock
	cfg              *config.Configuration
	sigTermCh        chan os.Signal
	stopCh           chan struct{}
	configService    config.ConfigService
	processorService processor.ProcessorService
}

func InitializePubxAIModule(client *http.Client, publisherId string, endpoint string, bufferInterval string, bufferSize string, samplingPercentage int, configRefresh string, clock clock.Clock) (analytics.Module, error) {
	glog.Infof("[pubxai] NewPubxAIModule: %v, %v, %v, %v", publisherId, endpoint, bufferInterval, bufferSize)
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
	configService, err := config.NewConfigService(
		client,
		publisherId,
		endpoint,
		configRefresh,
	)

	if err != nil {
		glog.Error("[pubxai] Error While creating config Service: %v", err)
		return nil, err
	}

	processorService := processor.NewProcessorService(publisherId, samplingPercentage)

	defaultConfig := &config.Configuration{
		PublisherId:        publisherId,
		BufferInterval:     bufferInterval,
		BufferSize:         bufferSize,
		SamplingPercentage: samplingPercentage,
	}
	winBidsQueue := queue.NewBidQueue("win", endpoint+"/win", client, clock, bufferInterval, bufferSize)
	auctionBidsQueue := queue.NewBidQueue("auction", endpoint+"/auction", client, clock, bufferInterval, bufferSize)

	pb := PubxaiModule{
		endpoint:         endpoint,
		cfg:              defaultConfig,
		winBidsQueue:     winBidsQueue.(*queue.WinningBidQueue),
		auctionBidsQueue: auctionBidsQueue.(*queue.AuctionBidsQueue),
		httpClient:       client,
		clock:            clock,
		muxConfig:        sync.RWMutex{},
		sigTermCh:        make(chan os.Signal),
		stopCh:           make(chan struct{}),
		configService:    configService,
		processorService: processorService,
	}

	signal.Notify(pb.sigTermCh, os.Interrupt, syscall.SIGTERM)

	configChannel := configService.Start(pb.stopCh)
	go pb.start(configChannel)

	glog.Info("[pubxai] pubxai analytics configured and ready")
	return &pb, nil
}

func (p *PubxaiModule) start(c <-chan *config.Configuration) {
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

func (p *PubxaiModule) updateConfig(config *config.Configuration) {
	p.muxConfig.Lock()
	defer p.muxConfig.Unlock()

	if p.configService.IsSameAs(config, p.cfg) {
		return
	}
	p.cfg = config
	p.auctionBidsQueue.UpdateConfig(config.BufferInterval, config.BufferSize)
	p.winBidsQueue.UpdateConfig(config.BufferInterval, config.BufferSize)
}

func (p *PubxaiModule) pushToQueue(auctionBids *utils.AuctionBids, winningBids []utils.WinningBid) {
	if len(winningBids) > 0 {
		for _, winningBid := range winningBids {
			p.winBidsQueue.Enqueue(winningBid)
		}
	}

	if auctionBids != nil {
		p.auctionBidsQueue.Enqueue(*auctionBids)
	}
}
func (p *PubxaiModule) LogAuctionObject(ao *analytics.AuctionObject) {
	if ao == nil {
		glog.Warning("[pubxai] Auction Object is nil")
		return
	}
	// Generate a random integer between 1 and 100
	randomNumber := rand.Intn(100) + 1
	if p.cfg.SamplingPercentage < randomNumber {
		return
	}
	// convert ao to LogObject
	lo := &utils.LogObject{
		Status:         ao.Status,
		Errors:         ao.Errors,
		Response:       ao.Response,
		StartTime:      ao.StartTime,
		SeatNonBid:     ao.SeatNonBid,
		RequestWrapper: ao.RequestWrapper,
	}
	p.pushToQueue(p.processorService.ProcessLogData(lo))
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

func (p *PubxaiModule) Shutdown() {
	close(p.sigTermCh)
}
