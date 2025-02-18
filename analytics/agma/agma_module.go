package agma

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/docker/go-units"
	"github.com/golang/glog"
	"github.com/prebid/go-gdpr/vendorconsent"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

type httpSender = func(payload []byte) error

const (
	agmaGVLID = 1122
	p9        = 9
)

type AgmaLogger struct {
	sender            httpSender
	clock             clock.Clock
	accounts          []config.AgmaAnalyticsAccount
	eventCount        int64
	maxEventCount     int64
	maxBufferByteSize int64
	maxDuration       time.Duration
	mux               sync.RWMutex
	sigTermCh         chan os.Signal
	buffer            bytes.Buffer
	bufferCh          chan []byte
}

func newAgmaLogger(cfg config.AgmaAnalytics, sender httpSender, clock clock.Clock) (*AgmaLogger, error) {
	pSize, err := units.FromHumanSize(cfg.Buffers.BufferSize)
	if err != nil {
		return nil, err
	}
	pDuration, err := time.ParseDuration(cfg.Buffers.Timeout)
	if err != nil {
		return nil, err
	}
	if len(cfg.Accounts) == 0 {
		return nil, errors.New("Please configure at least one account for Agma Analytics")
	}

	buffer := bytes.Buffer{}
	buffer.Write([]byte("["))

	return &AgmaLogger{
		sender:            sender,
		clock:             clock,
		accounts:          cfg.Accounts,
		maxBufferByteSize: pSize,
		eventCount:        0,
		maxEventCount:     int64(cfg.Buffers.EventCount),
		maxDuration:       pDuration,
		buffer:            buffer,
		bufferCh:          make(chan []byte),
		sigTermCh:         make(chan os.Signal, 1),
	}, nil
}

func NewModule(httpClient *http.Client, cfg config.AgmaAnalytics, clock clock.Clock) (analytics.Module, error) {
	sender, err := createHttpSender(httpClient, cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	m, err := newAgmaLogger(cfg, sender, clock)
	if err != nil {
		return nil, err
	}

	signal.Notify(m.sigTermCh, os.Interrupt, syscall.SIGTERM)

	go m.start()

	return m, nil
}

func (l *AgmaLogger) start() {
	ticker := l.clock.Ticker(l.maxDuration)
	for {
		select {
		case <-l.sigTermCh:
			glog.Infof("[AgmaAnalytics] Received Close, trying to flush buffer")
			l.flush()
			return
		case event := <-l.bufferCh:
			l.bufferEvent(event)
			if l.isFull() {
				l.flush()
			}
		case <-ticker.C:
			l.flush()
		}
	}
}

func (l *AgmaLogger) bufferEvent(data []byte) {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.buffer.Write(data)
	l.buffer.WriteByte(',')
	l.eventCount++
}

func (l *AgmaLogger) isFull() bool {
	l.mux.RLock()
	defer l.mux.RUnlock()
	return l.eventCount >= l.maxEventCount || int64(l.buffer.Len()) >= l.maxBufferByteSize
}

func (l *AgmaLogger) flush() {
	l.mux.Lock()

	if l.eventCount == 0 || l.buffer.Len() == 0 {
		l.mux.Unlock()
		return
	}

	// Close the json array, remove last ,
	l.buffer.Truncate(l.buffer.Len() - 1)
	l.buffer.Write([]byte("]"))

	payload := make([]byte, l.buffer.Len())
	_, err := l.buffer.Read(payload)
	if err != nil {
		l.reset()
		l.mux.Unlock()
		glog.Warning("[AgmaAnalytics] fail to copy the buffer")
		return
	}

	go l.sender(payload)

	l.reset()
	l.mux.Unlock()
}

func (l *AgmaLogger) reset() {
	l.buffer.Reset()
	l.buffer.Write([]byte("["))
	l.eventCount = 0
}

func (l *AgmaLogger) extractPublisherAndSite(requestWrapper *openrtb_ext.RequestWrapper) (string, string) {
	publisherId := ""
	appSiteId := ""
	if requestWrapper.Site != nil {
		if requestWrapper.Site.Publisher != nil {
			publisherId = requestWrapper.Site.Publisher.ID
		}
		appSiteId = requestWrapper.Site.ID
	}
	if requestWrapper.App != nil {
		if requestWrapper.App.Publisher != nil {
			publisherId = requestWrapper.App.Publisher.ID
		}
		appSiteId = requestWrapper.App.ID
		if appSiteId == "" {
			appSiteId = requestWrapper.App.Bundle
		}

	}
	return publisherId, appSiteId
}

func (l *AgmaLogger) shouldTrackEvent(requestWrapper *openrtb_ext.RequestWrapper) (bool, string) {
	if requestWrapper.User == nil {
		return false, ""
	}
	consentStr := requestWrapper.User.Consent

	parsedConsent, err := vendorconsent.ParseString(consentStr)
	if err != nil {
		return false, ""
	}

	p9Allowed := parsedConsent.PurposeAllowed(p9)
	agmaAllowed := parsedConsent.VendorConsent(agmaGVLID)
	if !p9Allowed || !agmaAllowed {
		return false, ""
	}

	publisherId, appSiteId := l.extractPublisherAndSite(requestWrapper)
	if publisherId == "" && appSiteId == "" {
		return false, ""
	}

	for _, account := range l.accounts {
		if account.PublisherId == publisherId {
			if account.SiteAppId == "" {
				return true, account.Code
			}
			if account.SiteAppId == appSiteId {
				return true, account.Code
			}
		}
	}

	return false, ""
}

func (l *AgmaLogger) LogAuctionObject(event *analytics.AuctionObject) {
	if event == nil || event.Status != http.StatusOK || event.RequestWrapper == nil {
		return
	}
	shouldTrack, code := l.shouldTrackEvent(event.RequestWrapper)
	if !shouldTrack {
		return
	}
	data, err := serializeAnayltics(event.RequestWrapper, EventTypeAuction, code, event.StartTime)
	if err != nil {
		glog.Errorf("[AgmaAnalytics] Error serializing auction object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *AgmaLogger) LogAmpObject(event *analytics.AmpObject) {
	if event == nil || event.Status != http.StatusOK || event.RequestWrapper == nil {
		return
	}
	shouldTrack, code := l.shouldTrackEvent(event.RequestWrapper)
	if !shouldTrack {
		return
	}
	data, err := serializeAnayltics(event.RequestWrapper, EventTypeAmp, code, event.StartTime)
	if err != nil {
		glog.Errorf("[AgmaAnalytics] Error serializing amp object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *AgmaLogger) LogVideoObject(event *analytics.VideoObject) {
	if event == nil || event.Status != http.StatusOK || event.RequestWrapper == nil {
		return
	}
	shouldTrack, code := l.shouldTrackEvent(event.RequestWrapper)
	if !shouldTrack {
		return
	}
	data, err := serializeAnayltics(event.RequestWrapper, EventTypeVideo, code, event.StartTime)
	if err != nil {
		glog.Errorf("[AgmaAnalytics] Error serializing video object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *AgmaLogger) Shutdown() {
	glog.Info("[AgmaAnalytics] Shutdown, trying to flush buffer")
	l.flush() // mutex safe
}

func (l *AgmaLogger) LogCookieSyncObject(event *analytics.CookieSyncObject)         {}
func (l *AgmaLogger) LogNotificationEventObject(event *analytics.NotificationEvent) {}
func (l *AgmaLogger) LogSetUIDObject(event *analytics.SetUIDObject)                 {}
