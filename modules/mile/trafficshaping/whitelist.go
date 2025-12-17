package trafficshaping

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// GeoWhitelist maps siteID to allowed country codes
type GeoWhitelist map[string]map[string]struct{}

// PlatformWhitelist maps siteID to allowed platform strings (e.g., "m-android/chrome")
type PlatformWhitelist map[string]map[string]struct{}

// WhitelistClient fetches and caches the geo and platform whitelists
type WhitelistClient struct {
	httpClient *http.Client
	config     *Config

	geoWhitelist      atomic.Pointer[GeoWhitelist]
	platformWhitelist atomic.Pointer[PlatformWhitelist]

	done     chan struct{}
	stopOnce sync.Once

	// For testing
	newTimer func(time.Duration) schedulerTimer
	jitterFn func(int64) int64
}

// NewWhitelistClient creates a new whitelist client and starts background refresh
func NewWhitelistClient(httpClient *http.Client, config *Config) *WhitelistClient {
	client := &WhitelistClient{
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

	// Initial fetch
	if err := client.fetchAll(); err != nil {
		glog.Warningf("trafficshaping: initial whitelist fetch failed: %v", err)
	}

	// Start background refresh
	go client.refreshLoop()

	return client
}

// IsAllowed checks if a site/geo/platform combination is in the whitelist
// Returns true (allow shaping) if:
// - Whitelist is not loaded (fail-open - allow shaping to proceed)
// - Site ID is empty (fail-open - let URL construction handle it)
// - Site is in whitelist and both geo and platform match
// Returns false (skip shaping) if:
// - Site is not in whitelist (site not enabled for traffic shaping)
// - Site is in whitelist but geo/platform don't match
func (c *WhitelistClient) IsAllowed(siteID, country, platform string) bool {
	geoWL := c.geoWhitelist.Load()
	platformWL := c.platformWhitelist.Load()

	// Fail-open if whitelists not loaded (allow shaping to proceed)
	if geoWL == nil || platformWL == nil {
		return true
	}

	// Empty site ID - fail-open (let URL construction handle the error)
	if siteID == "" {
		return true
	}

	// Check if site is in geo whitelist
	geoAllowed, geoSiteExists := (*geoWL)[siteID]
	if !geoSiteExists {
		// Site not in whitelist - skip shaping (site not enabled)
		return false
	}

	// Check if site is in platform whitelist
	platformAllowed, platformSiteExists := (*platformWL)[siteID]
	if !platformSiteExists {
		// Site not in platform whitelist - skip shaping
		return false
	}

	// Both whitelists have this site - check if geo and platform match
	_, geoMatch := geoAllowed[country]
	_, platformMatch := platformAllowed[platform]

	return geoMatch && platformMatch
}

// Stop stops the background refresh goroutine
func (c *WhitelistClient) Stop() {
	c.stopOnce.Do(func() {
		close(c.done)
	})
}

// refreshLoop periodically fetches the whitelists
func (c *WhitelistClient) refreshLoop() {
	interval := c.config.GetWhitelistRefreshInterval()
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
			if err := c.fetchAll(); err != nil {
				glog.Warningf("trafficshaping: whitelist fetch failed: %v", err)

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

// fetchAll fetches both geo and platform whitelists
func (c *WhitelistClient) fetchAll() error {
	var geoErr, platformErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		geoErr = c.fetchGeoWhitelist()
	}()

	go func() {
		defer wg.Done()
		platformErr = c.fetchPlatformWhitelist()
	}()

	wg.Wait()

	if geoErr != nil && platformErr != nil {
		return fmt.Errorf("both fetches failed: geo=%v, platform=%v", geoErr, platformErr)
	}
	if geoErr != nil {
		return fmt.Errorf("geo whitelist fetch failed: %w", geoErr)
	}
	if platformErr != nil {
		return fmt.Errorf("platform whitelist fetch failed: %w", platformErr)
	}

	return nil
}

// fetchGeoWhitelist fetches and parses the geo whitelist
func (c *WhitelistClient) fetchGeoWhitelist() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.GetRequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.config.GeoWhitelistEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON: {"siteID": ["US", "CA", ...]}
	var raw map[string][]string
	if err := jsonutil.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Convert to map[string]map[string]struct{} for fast lookup
	whitelist := make(GeoWhitelist, len(raw))
	for siteID, geos := range raw {
		geoSet := make(map[string]struct{}, len(geos))
		for _, geo := range geos {
			geoSet[geo] = struct{}{}
		}
		whitelist[siteID] = geoSet
	}

	c.geoWhitelist.Store(&whitelist)
	glog.Infof("trafficshaping: geo whitelist loaded with %d sites", len(whitelist))

	return nil
}

// fetchPlatformWhitelist fetches and parses the platform whitelist
func (c *WhitelistClient) fetchPlatformWhitelist() error {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.GetRequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", c.config.PlatformWhitelistEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON: {"siteID": ["m-android/chrome", "w/safari", ...]}
	var raw map[string][]string
	if err := jsonutil.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	// Convert to map[string]map[string]struct{} for fast lookup
	whitelist := make(PlatformWhitelist, len(raw))
	for siteID, platforms := range raw {
		platformSet := make(map[string]struct{}, len(platforms))
		for _, platform := range platforms {
			platformSet[platform] = struct{}{}
		}
		whitelist[siteID] = platformSet
	}

	c.platformWhitelist.Store(&whitelist)
	glog.Infof("trafficshaping: platform whitelist loaded with %d sites", len(whitelist))

	return nil
}
