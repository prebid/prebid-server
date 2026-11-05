package rulesengine

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	innerStorage := map[accountID]*cacheEntry{
		"account-id-one": {
			hashedConfig: "hash1",
		},
	}
	cache := NewCache(0)
	cache.m.Store(innerStorage)

	testCases := []struct {
		desc        string
		inAccountID string
		expectedObj *cacheEntry
	}{
		{
			desc:        "key not found",
			inAccountID: "foo-account",
			expectedObj: nil,
		},
		{
			desc:        "success",
			inAccountID: "account-id-one",
			expectedObj: &cacheEntry{
				hashedConfig: "hash1",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			assert.Equal(t, tc.expectedObj, cache.Get(tc.inAccountID))
		})
	}
}

func TestSet(t *testing.T) {
	innerStorage := map[accountID]*cacheEntry{
		"account-id-one": {
			hashedConfig: "hash1",
		},
	}
	cache := NewCache(0)
	cache.m.Store(innerStorage)

	testCases := []struct {
		desc              string
		inAccountID       string
		inCacheEntry      *cacheEntry
		expectedStoredObj *cacheEntry
		expectedStorage   map[accountID]*cacheEntry
	}{
		{
			desc:        "success. Insert object under key that wasn't found in our cache already",
			inAccountID: "account-id-two",
			inCacheEntry: &cacheEntry{
				hashedConfig: "hash2",
			},
			expectedStoredObj: &cacheEntry{
				hashedConfig: "hash2",
			},
			expectedStorage: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
				"account-id-two": {
					hashedConfig: "hash2",
				},
			},
		},
		{
			desc:        "success. Updated object under key that was found in our cache already",
			inAccountID: "account-id-one",
			inCacheEntry: &cacheEntry{
				hashedConfig: "updatedHash",
			},
			expectedStoredObj: &cacheEntry{
				hashedConfig: "updatedHash",
			},
			expectedStorage: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "updatedHash",
				},
				"account-id-two": {
					hashedConfig: "hash2",
				},
			},
		},
		{
			desc:              "success. Insert nil object under new key",
			inAccountID:       "foo-account-id",
			inCacheEntry:      nil,
			expectedStoredObj: nil,
			expectedStorage: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "updatedHash",
				},
				"account-id-two": {
					hashedConfig: "hash2",
				},
				"foo-account-id": nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cache.Set(tc.inAccountID, tc.inCacheEntry)
			storedObject := cache.Get(tc.inAccountID)

			// Assert object was stored
			assert.Equal(t, tc.expectedStoredObj, storedObject)

			// Assert inner cache stored values
			actualStorage := cache.m.Load().(map[accountID]*cacheEntry)
			assert.Equal(t, tc.expectedStorage, actualStorage)
		})
	}
}

func TestDelete(t *testing.T) {
	originalInnerStorage := map[accountID]*cacheEntry{
		"account-id-one": {
			hashedConfig: "hash1",
		},
		"account-id-two": {
			hashedConfig: "hash2",
		},
	}

	testCases := []struct {
		desc            string
		inKeyToDelete   string
		expectedStorage map[accountID]*cacheEntry
	}{
		{
			desc:            "Empty key, expect same inner storage",
			inKeyToDelete:   "",
			expectedStorage: originalInnerStorage,
		},
		{
			desc:            "Key not found, expect equal inner storage",
			inKeyToDelete:   "foo-id",
			expectedStorage: originalInnerStorage,
		},
		{
			desc:          "Remove a key that exists inside storage",
			inKeyToDelete: "account-id-two",
			expectedStorage: map[accountID]*cacheEntry{
				"account-id-one": {
					hashedConfig: "hash1",
				},
			},
		},
	}

	cache := NewCache(0)
	cache.m.Store(originalInnerStorage)

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			cache.Delete(tc.inKeyToDelete)

			// Assert inner cache stored values
			actualStorage := cache.m.Load().(map[accountID]*cacheEntry)
			assert.Equal(t, tc.expectedStorage, actualStorage)
		})
	}
}

func TestExpired(t *testing.T) {
	testCases := []struct {
		name               string
		inTimestamp        time.Time
		refreshRateSeconds int
		expectedResult     bool
	}{
		{
			name:               "expired",
			inTimestamp:        mockTimeUtil{}.Now().Add(-6 * time.Second),
			refreshRateSeconds: 5,
			expectedResult:     true,
		},
		{
			name:               "not_expired",
			inTimestamp:        mockTimeUtil{}.Now().Add(-4 * time.Second),
			refreshRateSeconds: 5,
			expectedResult:     false,
		},
		{
			name:               "no_refresh_rate",
			inTimestamp:        mockTimeUtil{}.Now().Add(-10 * time.Second),
			refreshRateSeconds: 0,
			expectedResult:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewCache(tc.refreshRateSeconds)
			c.t = mockTimeUtil{}
			res := c.Expired(tc.inTimestamp)
			assert.Equal(t, tc.expectedResult, res)
		})
	}
}

type mockTimeUtil struct{}

func (mt mockTimeUtil) Now() time.Time {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestSetEvictsLeastRecentlyUsedWhenFull(t *testing.T) {
	cache := NewCache(0)
	cache.maxEntries = 3

	cache.Set("a", &cacheEntry{})
	cache.Set("b", &cacheEntry{}) // never read again: least recently used
	cache.Set("c", &cacheEntry{})
	assert.NotNil(t, cache.Get("a"))
	assert.NotNil(t, cache.Get("c"))

	// Rewriting an existing key doesn't evict anything.
	cache.Set("a", &cacheEntry{})
	assert.Len(t, cache.m.Load().(map[accountID]*cacheEntry), 3)
	assert.NotNil(t, cache.Get("b"))
	assert.NotNil(t, cache.Get("a"))
	assert.NotNil(t, cache.Get("c"))

	// A new key at the limit replaces the least recently used entry.
	cache.Set("e", &cacheEntry{}) // "b" was read before "a" and "c"
	assert.Len(t, cache.m.Load().(map[accountID]*cacheEntry), 3)
	assert.Nil(t, cache.Get("b"))
	assert.NotNil(t, cache.Get("a"))
	assert.NotNil(t, cache.Get("c"))
	assert.NotNil(t, cache.Get("e"))
}

// A long-lived account that keeps receiving auctions must survive a flood of new IDs,
// even though its entry was built long before any of the new ones.
func TestSetKeepsSteadyAccountThroughFlood(t *testing.T) {
	cache := NewCache(0)
	cache.maxEntries = 100

	steady := &cacheEntry{hashedConfig: "steady", timestamp: time.Unix(0, 0)}
	cache.Set("steady-account", steady)
	for i := 0; i < 99; i++ {
		cache.Set(fmt.Sprintf("old-idle-%d", i), &cacheEntry{timestamp: time.Unix(int64(i+1), 0)})
	}

	for i := 0; i < 10000; i++ {
		// One auction for the steady account between new IDs.
		assert.Same(t, steady, cache.Get("steady-account"), "evicted after %d new IDs", i)
		cache.Set(fmt.Sprintf("flood-%d", i), &cacheEntry{timestamp: time.Unix(int64(1000+i), 0)})
	}

	assert.Same(t, steady, cache.Get("steady-account"))
	assert.Len(t, cache.m.Load().(map[accountID]*cacheEntry), 100)
	assert.Nil(t, cache.Get("old-idle-0"))
}

func TestGetConcurrentWithSet(t *testing.T) {
	cache := NewCache(0)
	cache.maxEntries = 10
	cache.Set("steady", &cacheEntry{})

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				cache.Get("steady")
			}
		}()
	}
	for i := 0; i < 1000; i++ {
		cache.Set(fmt.Sprintf("id-%d", i), &cacheEntry{})
		if i%100 == 0 {
			cache.Delete("steady")
			cache.Set("steady", &cacheEntry{})
		}
	}
	wg.Wait()
}

func TestSetStaysBoundedForManyAccountIDs(t *testing.T) {
	cache := NewCache(0)
	cache.maxEntries = 100

	for i := 0; i < 5000; i++ {
		cache.Set(fmt.Sprintf("random-account-%d", i), &cacheEntry{timestamp: time.Unix(int64(i), 0)})
	}
	assert.Len(t, cache.m.Load().(map[accountID]*cacheEntry), 100)
}
