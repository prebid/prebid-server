package http

import (
	"bytes"
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
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/util/randomutil"
)

type httpSender = func(payload []byte) error

type HttpLogger struct {
	sender httpSender
	clock  clock.Clock

	eventCount              int64
	maxEventCount           int64
	maxBufferByteSize       int64
	maxDuration             time.Duration
	shouldTrackAuction      filterObjectFunc[analytics.AuctionObject]
	shouldTrackAmp          filterObjectFunc[analytics.AmpObject]
	shouldTrackCookieSync   filterObjectFunc[analytics.CookieSyncObject]
	shouldTrackNotification filterObjectFunc[analytics.NotificationEvent]
	shouldTrackSetUID       filterObjectFunc[analytics.SetUIDObject]
	shouldTrackVideo        filterObjectFunc[analytics.VideoObject]
	mux                     sync.RWMutex
	sigTermCh               chan os.Signal
	buffer                  bytes.Buffer
	bufferCh                chan []byte
}

func newHttpLogger(cfg config.AnalyticsHttp, sender httpSender, clock clock.Clock) (*HttpLogger, error) {
	pSize, err := units.FromHumanSize(cfg.Buffers.BufferSize)
	if err != nil {
		return nil, err
	}
	pDuration, err := time.ParseDuration(cfg.Buffers.Timeout)
	if err != nil {
		return nil, err
	}

	randomGenerator := randomutil.RandomNumberGenerator{}
	shouldTrackAuction, err := createFilter[analytics.AuctionObject](cfg.Auction, randomGenerator)
	if err != nil {
		return nil, err
	}
	shouldTrackAmp, err := createFilter[analytics.AmpObject](cfg.Auction, randomGenerator)
	if err != nil {
		return nil, err
	}
	shouldTrackCookieSync, err := createFilter[analytics.CookieSyncObject](cfg.CookieSync, randomGenerator)
	if err != nil {
		return nil, err
	}
	shouldTrackNotification, err := createFilter[analytics.NotificationEvent](cfg.Notification, randomGenerator)
	if err != nil {
		return nil, err
	}
	shouldTrackSetUID, err := createFilter[analytics.SetUIDObject](cfg.SetUID, randomGenerator)
	if err != nil {
		return nil, err
	}
	shouldTrackVideo, err := createFilter[analytics.VideoObject](cfg.Video, randomGenerator)
	if err != nil {
		return nil, err
	}

	buffer := bytes.Buffer{}
	buffer.Write([]byte("["))

	return &HttpLogger{
		sender:                  sender,
		clock:                   clock,
		maxBufferByteSize:       pSize,
		eventCount:              0,
		maxEventCount:           int64(cfg.Buffers.EventCount),
		maxDuration:             pDuration,
		shouldTrackAuction:      shouldTrackAuction,
		shouldTrackAmp:          shouldTrackAmp,
		shouldTrackCookieSync:   shouldTrackCookieSync,
		shouldTrackNotification: shouldTrackNotification,
		shouldTrackSetUID:       shouldTrackSetUID,
		shouldTrackVideo:        shouldTrackVideo,
		buffer:                  buffer,
		bufferCh:                make(chan []byte),
		sigTermCh:               make(chan os.Signal),
	}, nil
}

func NewModule(httpClient *http.Client, cfg config.AnalyticsHttp, clock clock.Clock) (analytics.Module, error) {
	sender, err := createHttpSender(httpClient, cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	m, err := newHttpLogger(cfg, sender, clock)
	if err != nil {
		return nil, err
	}

	signal.Notify(m.sigTermCh, os.Interrupt, syscall.SIGTERM)

	go m.start()

	return m, nil
}

func (l *HttpLogger) start() {
	ticker := l.clock.Ticker(l.maxDuration)
	for {
		select {
		case <-l.sigTermCh:
			glog.Infof("[HttpAnalytics] Received Close, trying to flush buffer")
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

func (l *HttpLogger) bufferEvent(data []byte) {
	l.mux.Lock()
	defer l.mux.Unlock()

	l.buffer.Write(data)
	l.buffer.WriteByte(',')
	l.eventCount++
}

func (l *HttpLogger) isFull() bool {
	l.mux.RLock()
	defer l.mux.RUnlock()
	return l.eventCount >= l.maxEventCount || int64(l.buffer.Len()) >= l.maxBufferByteSize
}

func (l *HttpLogger) flush() {
	l.mux.Lock()

	if l.eventCount == 0 || l.buffer.Len() == 0 {
		l.mux.Unlock()
		return
	}

	// Close the json array, remove last ,
	l.buffer.Truncate(l.buffer.Len() - 1)
	_, err := l.buffer.Write([]byte("]"))
	if err != nil {
		l.reset()
		l.mux.Unlock()
		glog.Warning("[HttpAnalytics] fail to close the json array")
		return
	}

	payload := make([]byte, l.buffer.Len())
	_, err = l.buffer.Read(payload)
	if err != nil {
		l.reset()
		l.mux.Unlock()
		glog.Warning("[HttpAnalytics] fail to copy the buffer")
		return
	}

	go l.sender(payload)

	l.reset()
	l.mux.Unlock()
}

func (l *HttpLogger) reset() {
	l.buffer.Reset()
	l.buffer.Write([]byte("["))
	l.eventCount = 0
}

func (l *HttpLogger) LogAuctionObject(event *analytics.AuctionObject) {
	shouldTrack := l.shouldTrackAuction(event)
	if !shouldTrack {
		return
	}
	data, err := serializeAuctionObject(event, l.clock.Now())
	if err != nil {
		glog.Errorf("[HttpAnalytics] Error serializing auction object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *HttpLogger) LogAmpObject(event *analytics.AmpObject) {
	shouldTrack := l.shouldTrackAmp(event)
	if !shouldTrack {
		return
	}
	data, err := serializeAmpObject(event, l.clock.Now())
	if err != nil {
		glog.Errorf("[HttpAnalytics] Error serializing amp object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *HttpLogger) LogCookieSyncObject(event *analytics.CookieSyncObject) {
	shouldTrack := l.shouldTrackCookieSync(event)
	if !shouldTrack {
		return
	}
	data, err := serializeCookieSyncObject(event, l.clock.Now())
	if err != nil {
		glog.Errorf("[HttpAnalytics] Error serializing cookie sync object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *HttpLogger) LogNotificationEventObject(event *analytics.NotificationEvent) {
	shouldTrack := l.shouldTrackNotification(event)
	if !shouldTrack {
		return
	}
	data, err := serializeNotificationEvent(event, l.clock.Now())
	if err != nil {
		glog.Errorf("[HttpAnalytics] Error serializing notification event object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *HttpLogger) LogSetUIDObject(event *analytics.SetUIDObject) {
	shouldTrack := l.shouldTrackSetUID(event)
	if !shouldTrack {
		return
	}
	data, err := serializeSetUIDObject(event, l.clock.Now())
	if err != nil {
		glog.Errorf("[HttpAnalytics] Error serializing setuid object: %v", err)
		return
	}
	l.bufferCh <- data
}

func (l *HttpLogger) LogVideoObject(event *analytics.VideoObject) {
	shouldTrack := l.shouldTrackVideo(event)
	if !shouldTrack {
		return
	}
	data, err := serializeVideoObject(event, l.clock.Now())
	if err != nil {
		glog.Errorf("[HttpAnalytics] Error serializing video object: %v", err)
		return
	}
	l.bufferCh <- data
}
