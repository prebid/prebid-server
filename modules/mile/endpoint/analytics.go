package endpoint

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/glog"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/prebid-server/v3/analytics/mile/helpers"
)

// RequestAnalytics handles logging of mile-bidder requests to the data pipeline.
type RequestAnalytics struct {
	eventCh     chan *helpers.MileAnalyticsEvent
	endCh       chan struct{}
	httpClient  *http.Client
	endpoint    string
	batchSize   int
	flushPeriod time.Duration
	buffer      []*helpers.MileAnalyticsEvent
	mu          sync.Mutex
	clock       clock.Clock
}

// NewRequestAnalytics creates a new RequestAnalytics instance.
func NewRequestAnalytics(cfg AnalyticsConfig, httpClient *http.Client) (*RequestAnalytics, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	flushPeriod, err := time.ParseDuration(cfg.FlushTimeout)
	if err != nil {
		return nil, fmt.Errorf("invalid flush_timeout: %w", err)
	}

	// Build the full endpoint URL with path and query params
	endpointURL, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}
	endpointURL.Path = "/bidanalytics-event/json"
	endpointURL.RawQuery = "mode=beacon"

	ra := &RequestAnalytics{
		eventCh:     make(chan *helpers.MileAnalyticsEvent, cfg.BatchSize*2),
		endCh:       make(chan struct{}),
		httpClient:  httpClient,
		endpoint:    endpointURL.String(),
		batchSize:   cfg.BatchSize,
		flushPeriod: flushPeriod,
		buffer:      make([]*helpers.MileAnalyticsEvent, 0, cfg.BatchSize),
		clock:       clock.New(),
	}

	go ra.run()

	return ra, nil
}

// LogRequest logs a mile-bidder request to the analytics pipeline.
func (ra *RequestAnalytics) LogRequest(mileReq MileRequest, r *http.Request) {
	if ra == nil {
		return
	}

	event := ra.buildEvent(mileReq, r)
	select {
	case ra.eventCh <- event:
	default:
		glog.Warning("[mile-analytics] Event channel full, dropping event")
	}
}

// Close gracefully shuts down the analytics, flushing any remaining events.
func (ra *RequestAnalytics) Close() {
	if ra == nil {
		return
	}
	close(ra.endCh)
}

// run is the main loop that buffers and flushes events.
func (ra *RequestAnalytics) run() {
	ticker := ra.clock.Ticker(ra.flushPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ra.endCh:
			ra.flush()
			return
		case event := <-ra.eventCh:
			ra.mu.Lock()
			ra.buffer = append(ra.buffer, event)
			shouldFlush := len(ra.buffer) >= ra.batchSize
			ra.mu.Unlock()
			if shouldFlush {
				ra.flush()
			}
		case <-ticker.C:
			ra.flush()
		}
	}
}

// flush sends the buffered events to the pipeline.
func (ra *RequestAnalytics) flush() {
	ra.mu.Lock()
	if len(ra.buffer) == 0 {
		ra.mu.Unlock()
		return
	}

	// Take ownership of the buffer
	toSend := ra.buffer
	ra.buffer = make([]*helpers.MileAnalyticsEvent, 0, ra.batchSize)
	ra.mu.Unlock()

	// Send asynchronously
	go ra.send(toSend)
}

// send posts the events to the pipeline endpoint.
func (ra *RequestAnalytics) send(events []*helpers.MileAnalyticsEvent) {
	// Convert to maps and remove serverTimestamp field
	var eventMaps []map[string]interface{}
	for _, event := range events {
		eventData, err := json.Marshal(event)
		if err != nil {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal(eventData, &m); err != nil {
			continue
		}
		delete(m, "serverTimestamp")
		eventMaps = append(eventMaps, m)
	}

	data, err := json.Marshal(eventMaps)
	if err != nil {
		glog.Warningf("[mile-analytics] Failed to marshal events: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, ra.endpoint, bytes.NewReader(data))
	if err != nil {
		glog.Warningf("[mile-analytics] Failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ra.httpClient.Do(req)
	if err != nil {
		glog.Warningf("[mile-analytics] Failed to send events: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		glog.Warningf("[mile-analytics] Pipeline returned status %d for %d events", resp.StatusCode, len(events))
	}
}

// buildEvent constructs a MileAnalyticsEvent from the request.
func (ra *RequestAnalytics) buildEvent(mileReq MileRequest, r *http.Request) *helpers.MileAnalyticsEvent {
	event := &helpers.MileAnalyticsEvent{
		EventType:       "mile-bidder-request",
		Timestamp:       time.Now().UnixMilli(),
		YetiSiteID:      mileReq.SiteID,
		YetiPublisherID: mileReq.PublisherID,
		IsPBS:           true,
	}

	ortb := mileReq.BaseORTB
	if ortb == nil {
		return event
	}

	// Extract device info
	if ortb.Device != nil {
		event.Ua = ortb.Device.UA
		event.Ip = ortb.Device.IP
		if event.Ip == "" {
			event.Ip = ortb.Device.IPv6
		}
		event.Device = mapDeviceType(ortb.Device.DeviceType)
		event.Browser = detectBrowser(ortb.Device.UA)

		// Geo data
		if ortb.Device.Geo != nil {
			event.CountryName = ortb.Device.Geo.Country
			event.StateName = ortb.Device.Geo.Region
			event.CityName = ortb.Device.Geo.City
		}
	}

	// Extract site info
	if ortb.Site != nil {
		event.Page = ortb.Site.Page
		event.ReferrerURL = ortb.Site.Ref
		event.Site = ortb.Site.Domain
		if ortb.Site.Publisher != nil {
			event.Publisher = ortb.Site.Publisher.Name
		}
	}

	// Extract user info
	if ortb.User != nil {
		event.UserID = ortb.User.ID
	}

	// Extract auction ID
	event.AuctionID = ortb.ID

	// Store the full request as arbitrary data
	if len(mileReq.Raw) > 0 {
		event.ArbitraryData = string(mileReq.Raw)
	}

	return event
}

// mapDeviceType converts OpenRTB device type to Mile's device code.
func mapDeviceType(dt adcom1.DeviceType) string {
	switch dt {
	case adcom1.DeviceMobile:
		return "m"
	case adcom1.DeviceTablet:
		return "t"
	case adcom1.DevicePC:
		return "w"
	case adcom1.DeviceTV:
		return "tv"
	default:
		return "w" // default to web/desktop
	}
}

// detectBrowser parses the User-Agent to determine the browser.
func detectBrowser(ua string) string {
	ua = strings.ToLower(ua)

	switch {
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/"):
		return "edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		return "opera"
	case strings.Contains(ua, "chrome") && !strings.Contains(ua, "chromium"):
		return "chrome"
	case strings.Contains(ua, "firefox"):
		return "firefox"
	case strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome"):
		return "safari"
	case strings.Contains(ua, "msie") || strings.Contains(ua, "trident"):
		return "ie"
	default:
		return "other"
	}
}
