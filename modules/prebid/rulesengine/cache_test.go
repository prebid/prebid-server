package rulesengine

import (
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
