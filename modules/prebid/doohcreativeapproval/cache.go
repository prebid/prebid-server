package doohcreativeapproval

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/coocood/freecache"
)

type cachedApprovalStatus struct {
	CreativeApprovalID   string         `json:"creative_approval_id"`
	Status               approvalStatus `json:"status"`
	RefreshAfterUnixNano int64          `json:"refresh_after_unix_nano"`
}

type cachedApprovalLookup struct {
	Status     approvalStatus
	RefreshDue bool
}

type approvalCache struct {
	cache     *freecache.Cache
	marshal   func(v any) ([]byte, error)
	unmarshal func(data []byte, v any) error
	now       func() time.Time
}

func newApprovalCache(sizeBytes int) *approvalCache {
	return &approvalCache{
		cache:     freecache.NewCache(sizeBytes),
		marshal:   json.Marshal,
		unmarshal: json.Unmarshal,
		now:       time.Now,
	}
}

func (c *approvalCache) get(creativeApprovalID string) (cachedApprovalLookup, bool) {
	if c == nil || c.cache == nil || creativeApprovalID == "" {
		return cachedApprovalLookup{}, false
	}

	data, err := c.cache.Get([]byte(creativeApprovalID))
	if err != nil {
		return cachedApprovalLookup{}, false
	}

	var entry cachedApprovalStatus
	if err := c.unmarshal(data, &entry); err != nil {
		return cachedApprovalLookup{}, false
	}
	if entry.CreativeApprovalID != creativeApprovalID || !isValidApprovalStatus(entry.Status) || entry.RefreshAfterUnixNano <= 0 {
		return cachedApprovalLookup{}, false
	}

	return cachedApprovalLookup{
		Status:     entry.Status,
		RefreshDue: !c.currentTime().Before(time.Unix(0, entry.RefreshAfterUnixNano)),
	}, true
}

func (c *approvalCache) set(creativeApprovalID string, status approvalStatus, refreshSeconds int) error {
	if c == nil || c.cache == nil || creativeApprovalID == "" || refreshSeconds <= 0 || !isValidApprovalStatus(status) {
		return nil
	}

	entry := cachedApprovalStatus{
		CreativeApprovalID:   creativeApprovalID,
		Status:               status,
		RefreshAfterUnixNano: c.currentTime().Add(time.Duration(refreshSeconds) * time.Second).UnixNano(),
	}
	data, err := c.marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal approval cache entry: %s", err)
	}

	if err := c.cache.Set([]byte(creativeApprovalID), data, 0); err != nil {
		return fmt.Errorf("store approval cache entry: %s", err)
	}
	return nil
}

func (c *approvalCache) currentTime() time.Time {
	if c.now == nil {
		return time.Now()
	}
	return c.now()
}
