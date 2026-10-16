package doohcreativeapproval

import (
	"context"
	"fmt"
	"sync"

	"github.com/prebid/prebid-server/v4/logger"
)

type approvalRefresh struct {
	Creative       creativeApproval
	FallbackStatus approvalStatus
}

type approvalRefreshCoordinator struct {
	mu       sync.Mutex
	inFlight map[string]struct{}
	slots    chan struct{}
	wg       sync.WaitGroup
}

func newApprovalRefreshCoordinator(maxConcurrent int) *approvalRefreshCoordinator {
	return &approvalRefreshCoordinator{
		inFlight: make(map[string]struct{}),
		slots:    make(chan struct{}, maxConcurrent),
	}
}

// claim returns capacityAvailable=false only when every background lookup slot is busy.
func (c *approvalRefreshCoordinator) claim(refreshes []approvalRefresh) (claimed []approvalRefresh, capacityAvailable bool) {
	if c == nil || len(refreshes) == 0 {
		return nil, true
	}

	c.mu.Lock()
	for _, refresh := range refreshes {
		id := refresh.Creative.CreativeApprovalID
		if _, ok := c.inFlight[id]; ok {
			continue
		}
		claimed = append(claimed, refresh)
	}
	if len(claimed) == 0 {
		c.mu.Unlock()
		return nil, true
	}

	select {
	case c.slots <- struct{}{}:
	default:
		c.mu.Unlock()
		return nil, false
	}
	for _, refresh := range claimed {
		c.inFlight[refresh.Creative.CreativeApprovalID] = struct{}{}
	}
	c.wg.Add(1)
	c.mu.Unlock()
	return claimed, true
}

func (c *approvalRefreshCoordinator) finish(refreshes []approvalRefresh) {
	c.mu.Lock()
	for _, refresh := range refreshes {
		delete(c.inFlight, refresh.Creative.CreativeApprovalID)
	}
	<-c.slots
	c.mu.Unlock()
	c.wg.Done()
}

func (c *approvalRefreshCoordinator) wait() {
	if c != nil {
		c.wg.Wait()
	}
}

func (m *Module) scheduleApprovalRefresh(cfg moduleConfig, accountID string, refreshes []approvalRefresh) []string {
	if len(refreshes) == 0 {
		return nil
	}
	if m.refreshes == nil {
		return []string{"DOOH creative approval refresh coordinator is not configured"}
	}

	claimed, capacityAvailable := m.refreshes.claim(refreshes)
	if !capacityAvailable {
		return nil
	}
	if len(claimed) == 0 {
		return nil
	}

	go m.runApprovalRefresh(cfg, accountID, claimed)
	return nil
}

func (m *Module) runApprovalRefresh(cfg moduleConfig, accountID string, refreshes []approvalRefresh) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Errorf("DOOH creative approval refresh panicked: %v", recovered)
			m.storeApprovalRefreshFallbacks(cfg, refreshes)
		}
		m.refreshes.finish(refreshes)
	}()

	creatives := make([]creativeApproval, 0, len(refreshes))
	for _, refresh := range refreshes {
		creatives = append(creatives, refresh.Creative)
	}

	statuses, warnings, err := m.provider.Lookup(context.Background(), cfg, accountID, creatives)
	for _, warning := range warnings {
		logger.Warnf("DOOH creative approval lookup warning: %s", warning)
	}
	if err != nil {
		logger.Warnf("DOOH creative approval lookup failed: %s", err)
		m.storeApprovalRefreshFallbacks(cfg, refreshes)
		return
	}

	for _, refresh := range refreshes {
		id := refresh.Creative.CreativeApprovalID
		status, ok := statuses[id]
		if !ok || !isValidApprovalStatus(status) {
			logger.Warnf("DOOH creative approval response did not contain a usable status for creative_approval_id %s", id)
			if err := m.cache.set(id, refresh.FallbackStatus, cfg.PendingTTLSeconds); err != nil {
				logger.Warnf("%s", cacheWriteWarning(id, err))
			}
			continue
		}
		if err := m.cache.set(id, status, ttlForStatus(cfg, status)); err != nil {
			logger.Warnf("%s", cacheWriteWarning(id, err))
		}
	}
}

func (m *Module) storeApprovalRefreshFallbacks(cfg moduleConfig, refreshes []approvalRefresh) {
	for _, refresh := range refreshes {
		id := refresh.Creative.CreativeApprovalID
		if err := m.cache.set(id, refresh.FallbackStatus, cfg.PendingTTLSeconds); err != nil {
			logger.Warnf("%s", cacheWriteWarning(id, err))
		}
	}
}

func cacheWriteWarning(creativeApprovalID string, err error) string {
	return fmt.Sprintf("DOOH creative approval cache write failed for creative_approval_id %s: %s", creativeApprovalID, err)
}
