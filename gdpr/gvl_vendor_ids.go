package gdpr

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prebid/prebid-server/v3/logger"
	"github.com/prebid/prebid-server/v3/util/task"
	"golang.org/x/net/context/ctxhttp"
)

// LiveGVLVendorIDs provides thread-safe access to the set of vendor IDs present in the latest
// Global Vendor List. It uses atomic.Value so reads (on every request) are lock-free, while
// writes (periodic refresh) are safe without mutexes.
type LiveGVLVendorIDs struct {
	ids atomic.Value // holds map[uint16]struct{}
}

// NewLiveGVLVendorIDs creates a LiveGVLVendorIDs initialized with an empty set.
func NewLiveGVLVendorIDs() *LiveGVLVendorIDs {
	l := &LiveGVLVendorIDs{}
	l.ids.Store(make(map[uint16]struct{}))
	return l
}

// Contains returns true if the given vendor ID is in the current valid set.
// If the set is empty (e.g. the GVL fetch failed), all IDs are considered valid
// as a safe fallback.
func (l *LiveGVLVendorIDs) Contains(id uint16) bool {
	m := l.ids.Load().(map[uint16]struct{})
	if len(m) == 0 {
		return true
	}
	_, ok := m[id]
	return ok
}

// Update atomically replaces the valid vendor ID set. If newIDs is empty, the existing
// set is retained.
func (l *LiveGVLVendorIDs) Update(newIDs map[uint16]struct{}) {
	if len(newIDs) > 0 {
		l.ids.Store(newIDs)
	}
}

// NewGVLVendorIDTickerTask creates a TickerTask that fetches the latest GVL vendor IDs and
// updates the LiveGVLVendorIDs set. Calling Start on the returned task performs the initial
// fetch immediately and then schedules periodic refreshes at the given interval.
func NewGVLVendorIDTickerTask(interval time.Duration, client *http.Client, urlMaker func(uint16, uint16) string, live *LiveGVLVendorIDs) *task.TickerTask {
	return task.NewTickerTaskFromFunc(interval, func() error {
		newIDs := FetchLatestGVLVendorIDs(context.Background(), client, urlMaker)
		live.Update(newIDs)
		return nil
	})
}

// gvlVendorListContract is a lightweight contract for parsing only vendor IDs from GVL JSON
type gvlVendorListContract struct {
	Vendors map[string]struct {
		ID uint16 `json:"id"`
	} `json:"vendors"`
}

// FetchLatestGVLVendorIDs fetches the most recent Global Vendor List and returns a set of all
// vendor IDs present in it. The returned map has vendor IDs as keys and empty structs as values.
// If the fetch or parse fails, an empty map is returned.
func FetchLatestGVLVendorIDs(ctx context.Context, client *http.Client, urlMaker func(uint16, uint16) string) map[uint16]struct{} {
	vendorIDs := make(map[uint16]struct{})

	// Fetch latest GVL for the latest spec version (listVersion 0 means latest)
	url := urlMaker(latestSpecVersion, 0)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Errorf("Failed to build GET %s request for GVL vendor ID extraction: %v", url, err)
		return vendorIDs
	}

	resp, err := ctxhttp.Do(ctx, client, req)
	if err != nil {
		logger.Errorf("Error calling GET %s for GVL vendor ID extraction: %v", url, err)
		return vendorIDs
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("Error reading response body from GET %s for GVL vendor ID extraction: %v", url, err)
		return vendorIDs
	}

	if resp.StatusCode != http.StatusOK {
		logger.Errorf("GET %s returned %d for GVL vendor ID extraction", url, resp.StatusCode)
		return vendorIDs
	}

	var contract gvlVendorListContract
	if err := json.Unmarshal(respBody, &contract); err != nil {
		logger.Errorf("GET %s returned malformed JSON for GVL vendor ID extraction: %v", url, err)
		return vendorIDs
	}

	for _, v := range contract.Vendors {
		vendorIDs[v.ID] = struct{}{}
	}

	return vendorIDs
}
