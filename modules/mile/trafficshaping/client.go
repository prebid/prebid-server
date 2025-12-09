package trafficshaping

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"golang.org/x/sync/singleflight"
)

// cachedConfig holds a config with expiry time
type cachedConfig struct {
	config    *ShapingConfig
	expiresAt time.Time
}

// ConfigClient fetches and caches the traffic shaping configuration
type ConfigClient struct {
	httpClient *http.Client
	config     *Config
	// Static mode (legacy): single config with background refresh
	cache    atomic.Pointer[ShapingConfig]
	done     chan struct{}
	stopOnce sync.Once
	// Dynamic mode: multiple configs keyed by URL
	dynamicCache       sync.Map // map[string]*cachedConfig
	lastDynamicCleanup int64
	fetchGroup         singleflight.Group
	newTimer           func(time.Duration) schedulerTimer
	jitterFn           func(int64) int64
}

// NewConfigClient creates a new config client and starts the background refresh
func NewConfigClient(httpClient *http.Client, config *Config) *ConfigClient {
	client := &ConfigClient{
		httpClient: httpClient,
		config:     config,
		done:       make(chan struct{}),
		newTimer: func(d time.Duration) schedulerTimer {
			return &realTimer{time.NewTimer(d)}
		},
		jitterFn: func(n int64) int64 {
			if n <= 0 {
				return 0
			}
			return rand.Int63n(n)
		},
	}

	// Only use background refresh for static mode (legacy endpoint config)
	if !config.IsDynamicMode() {
		// Initial fetch for static mode
		if err := client.fetch(); err != nil {
			glog.Warningf("trafficshaping: initial fetch failed: %v", err)
		}

		// Start background refresh for static mode
		go client.refreshLoop()
	}

	return client
}

// GetConfig returns the current shaping config (may be nil if fetch failed)
// This is for static mode (legacy) - use GetConfigForURL for dynamic mode
func (c *ConfigClient) GetConfig() *ShapingConfig {
	return c.cache.Load()
}

// GetConfigForURL returns the config for a specific URL (dynamic mode)
// Fetches and caches on-demand with TTL-based expiry
func (c *ConfigClient) GetConfigForURL(url string) *ShapingConfig {
	now := time.Now()
	c.maybeCleanupDynamicCache(now)

	// Check cache first
	if cached, ok := c.dynamicCache.Load(url); ok {
		cachedCfg := cached.(*cachedConfig)
		// Return if not expired
		if now.Before(cachedCfg.expiresAt) {
			return cachedCfg.config
		}
		// Expired, remove from cache
		c.dynamicCache.Delete(url)
	}

	// Fetch new config (coalesced per URL)
	value, err, _ := c.fetchGroup.Do(url, func() (interface{}, error) {
		config, fetchErr := c.fetchForURL(url)
		if fetchErr != nil {
			return nil, fetchErr
		}

		expiresAt := time.Now().Add(c.config.GetRefreshInterval())
		c.dynamicCache.Store(url, &cachedConfig{
			config:    config,
			expiresAt: expiresAt,
		})

		return config, nil
	})
	if err != nil {
		glog.Warningf("trafficshaping: fetch failed for %s: %v", url, err)
		return nil
	}

	return value.(*ShapingConfig)
}

// Stop stops the background refresh goroutine
func (c *ConfigClient) Stop() {
	c.stopOnce.Do(func() {
		close(c.done)
	})
}

// refreshLoop periodically fetches the configuration
func (c *ConfigClient) refreshLoop() {
	interval := c.config.GetRefreshInterval()
	timer := c.newTimer(interval)
	defer timer.Stop()

	backoff := time.Duration(0)
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-c.done:
			if !timer.Stop() {
				select {
				case <-timer.C():
				default:
				}
			}
			return
		case <-timer.C():
			if err := c.fetch(); err != nil {
				glog.Warningf("trafficshaping: fetch failed: %v", err)

				// Exponential backoff on errors
				if backoff == 0 {
					backoff = 10 * time.Second
				} else {
					backoff = min(backoff*2, maxBackoff)
				}

				// Add jitter
				jitter := time.Duration(c.jitterFn(int64(backoff / 10)))
				timer.Reset(backoff + jitter)
			} else {
				// Reset backoff on success
				backoff = 0
				timer.Reset(interval)
			}
		}
	}
}

func (c *ConfigClient) maybeCleanupDynamicCache(now time.Time) {
	interval := c.config.GetRefreshInterval()
	if interval <= 0 {
		interval = time.Minute
	}
	last := atomic.LoadInt64(&c.lastDynamicCleanup)
	if now.UnixNano()-last < interval.Nanoseconds() {
		return
	}
	if !atomic.CompareAndSwapInt64(&c.lastDynamicCleanup, last, now.UnixNano()) {
		return
	}
	c.dynamicCache.Range(func(key, value any) bool {
		cached, ok := value.(*cachedConfig)
		if ok && now.After(cached.expiresAt) {
			c.dynamicCache.Delete(key)
		}
		return true
	})
}

type schedulerTimer interface {
	C() <-chan time.Time
	Reset(time.Duration) bool
	Stop() bool
}

type realTimer struct {
	t *time.Timer
}

func (rt *realTimer) C() <-chan time.Time {
	return rt.t.C
}

func (rt *realTimer) Reset(d time.Duration) bool {
	return rt.t.Reset(d)
}

func (rt *realTimer) Stop() bool {
	return rt.t.Stop()
}

// fetch retrieves and parses the configuration from the remote endpoint (static mode)
func (c *ConfigClient) fetch() error {
	return c.fetchAndStore(c.config.Endpoint, true)
}

// fetchForURL fetches config for a specific URL (dynamic mode)
func (c *ConfigClient) fetchForURL(url string) (*ShapingConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.GetRequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var data TrafficShapingData
	if err := jsonutil.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Preprocess the config for fast lookup
	shapingConfig := preprocessConfig(&data.Response)

	glog.Infof("trafficshaping: config fetched for %s, skipRate=%d, gpids=%d",
		url, shapingConfig.SkipRate, len(shapingConfig.GPIDRules))

	return shapingConfig, nil
}

// fetchAndStore fetches and stores config (for static mode)
func (c *ConfigClient) fetchAndStore(url string, storeInCache bool) error {
	config, err := c.fetchForURL(url)
	if err != nil {
		return err
	}

	if storeInCache {
		c.cache.Store(config)
	}

	return nil
}

// preprocessConfig converts the raw config into an optimized lookup structure
func preprocessConfig(response *Response) *ShapingConfig {
	config := &ShapingConfig{
		SkipRate:      response.SkipRate,
		UserIdVendors: make(map[string]struct{}, len(response.UserIdVendors)),
		GPIDRules:     make(map[string]*GPIDRule, len(response.Values)),
	}

	// Build user ID vendors map
	for _, vendor := range response.UserIdVendors {
		config.UserIdVendors[vendor] = struct{}{}
	}

	// Build GPID rules
	for gpid, bidders := range response.Values {
		rule := &GPIDRule{
			AllowedBidders: make(map[string]struct{}, len(bidders)),
			AllowedSizes:   make(map[BannerSize]struct{}),
		}

		// Collect allowed bidders and sizes
		for bidder, sizes := range bidders {
			hasAllowedSize := false

			// Parse sizes (e.g., "320x50") and check if any are allowed
			for sizeStr, flag := range sizes {
				if flag == 1 { // Only if this size is allowed
					hasAllowedSize = true
					if size := parseSize(sizeStr); size != nil {
						rule.AllowedSizes[*size] = struct{}{}
					}
				}
			}

			// Only add bidder if at least one size is allowed
			if hasAllowedSize {
				rule.AllowedBidders[bidder] = struct{}{}
			}
		}

		config.GPIDRules[gpid] = rule
	}

	return config
}

// parseSize parses a size string like "320x50" into a BannerSize
func parseSize(sizeStr string) *BannerSize {
	parts := strings.Split(sizeStr, "x")
	if len(parts) != 2 {
		return nil
	}

	w, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil
	}

	h, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil
	}

	return &BannerSize{W: w, H: h}
}
