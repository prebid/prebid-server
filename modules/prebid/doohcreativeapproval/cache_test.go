package doohcreativeapproval

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApprovalCacheGetSet(t *testing.T) {
	cache := newApprovalCache(1024 * 1024)

	cache.set("v1:approved", approvalStatusApproved, 60)
	cache.set("v1:rejected", approvalStatusRejected, 60)
	cache.set("v1:pending", approvalStatusPending, 60)

	lookup, ok := cache.get("v1:approved")
	assert.True(t, ok)
	assert.Equal(t, approvalStatusApproved, lookup.Status)
	assert.False(t, lookup.RefreshDue)

	lookup, ok = cache.get("v1:rejected")
	assert.True(t, ok)
	assert.Equal(t, approvalStatusRejected, lookup.Status)
	assert.False(t, lookup.RefreshDue)

	lookup, ok = cache.get("v1:pending")
	assert.True(t, ok)
	assert.Equal(t, approvalStatusPending, lookup.Status)
	assert.False(t, lookup.RefreshDue)
}

func TestApprovalCacheMisses(t *testing.T) {
	cache := newApprovalCache(1024 * 1024)

	cache.set("v1:zero-ttl", approvalStatusApproved, 0)
	cache.set("v1:bad-status", "unknown", 60)

	_, ok := cache.get("v1:missing")
	assert.False(t, ok)

	_, ok = cache.get("v1:zero-ttl")
	assert.False(t, ok)

	_, ok = cache.get("v1:bad-status")
	assert.False(t, ok)
}

func TestApprovalCacheRefreshDueKeepsLastStatus(t *testing.T) {
	cache := newApprovalCache(1024 * 1024)
	now := time.Unix(1000, 0)
	cache.now = func() time.Time {
		return now
	}

	cache.set("v1:refresh", approvalStatusApproved, 1)
	now = now.Add(2 * time.Second)

	lookup, ok := cache.get("v1:refresh")
	assert.True(t, ok)
	assert.Equal(t, approvalStatusApproved, lookup.Status)
	assert.True(t, lookup.RefreshDue)
}

func TestApprovalCacheSetReturnsWriteError(t *testing.T) {
	cache := newApprovalCache(1024 * 1024)
	cache.marshal = func(v any) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	err := cache.set("v1:write-error", approvalStatusApproved, 60)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal approval cache entry")
}
