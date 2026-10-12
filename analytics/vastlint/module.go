// Package vastlint samples returned inline video markup for host monitoring.
package vastlint

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/analytics"
	"github.com/prebid/prebid-server/v4/analytics/vastlint/api"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

const maxScan = 64
const maxExtensionBytes = 4096

var counterNames = [...]string{"auctions", "sampled", "scan_limited", "non_video", "missing_markup", "url_markup", "oversized_markup", "oversized_extension", "capacity_dropped", "rate_dropped", "expired", "submitted", "checked", "with_findings", "failed", "shutdown_dropped"}

const (
	auctions = iota
	sampled
	scanLimited
	nonVideo
	missingMarkup
	urlMarkup
	oversizedMarkup
	oversizedExtension
	capacityDropped
	rateDropped
	expired
	submitted
	checked
	withFindings
	failed
	shutdownDropped
)

type validator interface {
	Validate(context.Context, *api.ValidateRequest, ...grpc.CallOption) (*api.ValidateResponse, error)
}

type document struct {
	xml     string
	created time.Time
}

// Module owns copied documents only. Queued and in-flight work share one cap.
type Module struct {
	cfg                     config.VastlintAnalytics
	accounts                map[string]struct{}
	client                  validator
	conn                    *grpc.ClientConn
	jobs                    chan document
	slots                   chan struct{}
	ctx                     context.Context
	cancel                  context.CancelFunc
	done                    chan struct{}
	shutdown                sync.Once
	mu                      sync.Mutex
	stopped                 atomic.Bool
	eventTokens, byteTokens float64
	refill                  time.Time
	counters                [len(counterNames)]atomic.Uint64
	now                     func() time.Time
	sample                  func() int
}

func normalize(cfg config.VastlintAnalytics) (config.VastlintAnalytics, error) {
	if cfg.SamplePercent == 0 {
		cfg.SamplePercent = 1
	}
	if cfg.MaxDocumentBytes == 0 {
		cfg.MaxDocumentBytes = 64 * 1024
	}
	if cfg.QueueSize == 0 {
		cfg.QueueSize = 32
	}
	if cfg.TimeoutMS == 0 {
		cfg.TimeoutMS = 250
	}
	if cfg.MaxAgeMS == 0 {
		cfg.MaxAgeMS = 1000
	}
	if cfg.MaxEventsPerSecond == 0 {
		cfg.MaxEventsPerSecond = 100
	}
	if cfg.MaxBytesPerSecond == 0 {
		cfg.MaxBytesPerSecond = 1024 * 1024
	}
	if cfg.SamplePercent < 1 || cfg.SamplePercent > 100 || cfg.MaxDocumentBytes < 1 || cfg.MaxDocumentBytes > 128*1024 ||
		cfg.QueueSize < 1 || cfg.QueueSize > 128 || cfg.QueueSize*cfg.MaxDocumentBytes > 8*1024*1024 ||
		cfg.TimeoutMS < 1 || cfg.TimeoutMS > 5000 || cfg.MaxAgeMS < 1 || cfg.MaxAgeMS > 10000 ||
		cfg.MaxEventsPerSecond < 1 || cfg.MaxEventsPerSecond > 10000 || cfg.MaxBytesPerSecond < cfg.MaxDocumentBytes || cfg.MaxBytesPerSecond > 16*1024*1024 {
		return cfg, errors.New("vastlint analytics resource limits are out of range")
	}
	if !cfg.Enabled {
		return cfg, nil
	}
	host, port, err := net.SplitHostPort(cfg.Endpoint)
	number, portErr := strconv.Atoi(port)
	if err != nil || host == "" || strings.ContainsAny(host, "/?#@ \t\n") || portErr != nil || number < 1 || number > 65535 {
		return cfg, errors.New("vastlint analytics endpoint must contain a host and numeric port")
	}
	if len(cfg.Accounts) == 0 || len(cfg.Accounts) > 256 {
		return cfg, errors.New("vastlint analytics requires 1 to 256 authorized account IDs")
	}
	for _, account := range cfg.Accounts {
		if len(account) == 0 || len(account) > 128 {
			return cfg, errors.New("vastlint analytics account ID length is out of range")
		}
	}
	return cfg, nil
}

// NewModule is disabled unless the host enables it and names authorized accounts.
func NewModule(cfg config.VastlintAnalytics) (*Module, error) {
	cfg, err := normalize(cfg)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	m := &Module{cfg: cfg, accounts: make(map[string]struct{}), jobs: make(chan document, cfg.QueueSize), slots: make(chan struct{}, cfg.QueueSize),
		ctx: ctx, cancel: cancel, done: make(chan struct{}), now: time.Now, sample: func() int { return rand.IntN(100) },
		eventTokens: float64(cfg.MaxEventsPerSecond), byteTokens: float64(cfg.MaxBytesPerSecond), refill: time.Now()}
	for _, account := range cfg.Accounts {
		m.accounts[account] = struct{}{}
	}
	if !cfg.Enabled {
		close(m.done)
		return m, nil
	}
	transport := credentials.TransportCredentials(insecure.NewCredentials())
	if !cfg.AllowInsecure {
		host, _, _ := net.SplitHostPort(cfg.Endpoint)
		transport = credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12, ServerName: host})
	}
	conn, err := grpc.NewClient("dns:///"+cfg.Endpoint, grpc.WithTransportCredentials(transport), grpc.WithDisableRetry(),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(64*1024), grpc.MaxCallSendMsgSize(cfg.MaxDocumentBytes+1024)))
	if err != nil {
		cancel()
		return nil, errors.New("could not initialize vastlint analytics client")
	}
	m.conn, m.client = conn, api.NewVastlintServiceClient(conn)
	go m.run()
	return m, nil
}

// RequiresCreativeDataPermissions asks analytics dispatch to gate raw creative
// export on all transmission activities, as well as reportAnalytics.
func (*Module) RequiresCreativeDataPermissions() bool { return true }

func (m *Module) Stats() map[string]uint64 {
	result := make(map[string]uint64, len(counterNames))
	for i, name := range counterNames {
		result[name] = m.counters[i].Load()
	}
	return result
}

func (m *Module) LogAuctionObject(event *analytics.AuctionObject) {
	if !m.cfg.Enabled || m.stopped.Load() || event == nil || event.Status != 200 || event.Account == nil || event.FinalResponse == nil {
		return
	}
	if _, authorized := m.accounts[event.Account.ID]; !authorized {
		return
	}
	m.counters[auctions].Add(1)
	if m.sample() >= m.cfg.SamplePercent {
		return
	}
	m.counters[sampled].Add(1)
	bids := 0
	for seatIndex := range event.FinalResponse.SeatBid {
		if seatIndex >= maxScan {
			m.counters[scanLimited].Add(1)
			return
		}
		for bidIndex := range event.FinalResponse.SeatBid[seatIndex].Bid {
			if bids >= maxScan {
				m.counters[scanLimited].Add(1)
				return
			}
			bids++
			bid := &event.FinalResponse.SeatBid[seatIndex].Bid[bidIndex]
			if !m.video(bid) {
				continue
			}
			if len(bid.AdM) > m.cfg.MaxDocumentBytes {
				m.counters[oversizedMarkup].Add(1)
				continue
			}
			trimmed := strings.TrimSpace(bid.AdM)
			if trimmed == "" {
				m.counters[missingMarkup].Add(1)
				continue
			}
			if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
				m.counters[urlMarkup].Add(1)
				continue
			}
			m.capture(bid.AdM)
			return
		}
	}
}

func (m *Module) video(bid *openrtb2.Bid) bool {
	if bid.MType == 2 {
		return true
	}
	if bid.MType != 0 {
		m.counters[nonVideo].Add(1)
		return false
	}
	if len(bid.Ext) > maxExtensionBytes {
		m.counters[oversizedExtension].Add(1)
		return false
	}
	var ext struct {
		Prebid struct {
			Type string `json:"type"`
		} `json:"prebid"`
	}
	if json.Unmarshal(bid.Ext, &ext) != nil || ext.Prebid.Type != "video" {
		m.counters[nonVideo].Add(1)
		return false
	}
	return true
}

func (m *Module) capture(xml string) {
	select {
	case m.slots <- struct{}{}:
	default:
		m.counters[capacityDropped].Add(1)
		return
	}
	created := m.now()
	m.mu.Lock()
	if m.stopped.Load() {
		m.mu.Unlock()
		<-m.slots
		m.counters[shutdownDropped].Add(1)
		return
	}
	elapsed := created.Sub(m.refill).Seconds()
	if elapsed > 0 {
		m.eventTokens = min(float64(m.cfg.MaxEventsPerSecond), m.eventTokens+elapsed*float64(m.cfg.MaxEventsPerSecond))
		m.byteTokens = min(float64(m.cfg.MaxBytesPerSecond), m.byteTokens+elapsed*float64(m.cfg.MaxBytesPerSecond))
		m.refill = created
	}
	if m.eventTokens < 1 || m.byteTokens < float64(len(xml)) {
		m.mu.Unlock()
		<-m.slots
		m.counters[rateDropped].Add(1)
		return
	}
	m.eventTokens--
	m.byteTokens -= float64(len(xml))
	m.mu.Unlock()
	// Capacity and rate admission precede allocation. Capacity remains reserved
	// through copying, queueing, and the actual client call.
	owned := document{xml: strings.Clone(xml), created: created}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped.Load() {
		<-m.slots
		m.counters[shutdownDropped].Add(1)
		return
	}
	select {
	case m.jobs <- owned:
	default:
		<-m.slots
		m.counters[capacityDropped].Add(1)
	}
}

func (m *Module) run() {
	defer close(m.done)
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.ctx.Done():
			for {
				select {
				case <-m.jobs:
					<-m.slots
					m.counters[shutdownDropped].Add(1)
				default:
					return
				}
			}
		case job := <-m.jobs:
			m.process(job)
		case <-ticker.C:
			m.logCounters()
		}
	}
}

func (m *Module) process(job document) {
	defer func() {
		<-m.slots
		if recover() != nil {
			m.counters[failed].Add(1)
		}
	}()
	if m.ctx.Err() != nil {
		m.counters[shutdownDropped].Add(1)
		return
	}
	remaining := time.Duration(m.cfg.MaxAgeMS)*time.Millisecond - m.now().Sub(job.created)
	if remaining <= 0 {
		m.counters[expired].Add(1)
		return
	}
	ctx, cancel := context.WithTimeout(m.ctx, min(time.Duration(m.cfg.TimeoutMS)*time.Millisecond, remaining))
	defer cancel()
	m.counters[submitted].Add(1)
	ctx = metadata.AppendToOutgoingContext(ctx, "x-vastlint-caller", "prebid-server")
	response, err := m.client.Validate(ctx, &api.ValidateRequest{Document: job.xml, SummaryOnly: true})
	if err != nil || ctx.Err() != nil {
		m.counters[failed].Add(1)
		return
	}
	verdict := response.GetVerdict()
	if !response.GetSummaryOnly() || verdict == nil || verdict.Summary == nil || verdict.Provenance == nil || verdict.Provenance.EngineVersion == "" || len(verdict.Provenance.EngineVersion) > 128 || !validDigest(verdict.Provenance.CatalogDigest) {
		m.counters[failed].Add(1)
		return
	}
	m.counters[checked].Add(1)
	if verdict.Summary.Errors != 0 || verdict.Summary.Warnings != 0 || verdict.Summary.Infos != 0 {
		m.counters[withFindings].Add(1)
	}
}

func (m *Module) logCounters() { logger.Infof("Vastlint analytics collection counters: %v", m.Stats()) }

func (m *Module) Shutdown() {
	m.shutdown.Do(func() {
		m.mu.Lock()
		m.stopped.Store(true)
		m.cancel()
		if m.conn != nil {
			_ = m.conn.Close()
		}
		m.mu.Unlock()
		select {
		case <-m.done:
		case <-time.After(time.Duration(m.cfg.TimeoutMS+100) * time.Millisecond):
		}
		m.logCounters()
	})
}

// The first version monitors only /openrtb2/auction final responses.
func (*Module) LogVideoObject(*analytics.VideoObject)                   {}
func (*Module) LogAmpObject(*analytics.AmpObject)                       {}
func (*Module) LogCookieSyncObject(*analytics.CookieSyncObject)         {}
func (*Module) LogSetUIDObject(*analytics.SetUIDObject)                 {}
func (*Module) LogNotificationEventObject(*analytics.NotificationEvent) {}

func validDigest(digest string) bool {
	if len(digest) != 71 || !strings.HasPrefix(digest, "sha256:") {
		return false
	}
	for i := 7; i < len(digest); i++ {
		if !(digest[i] >= '0' && digest[i] <= '9' || digest[i] >= 'a' && digest[i] <= 'f') {
			return false
		}
	}
	return true
}
