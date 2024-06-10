package usersync

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPriorityEjector(t *testing.T) {
	testCases := []struct {
		name          string
		givenUids     map[string]UIDEntry
		givenEjector  Ejector
		expected      string
		expectedError error
	}{
		{
			name: "one-lowest-priority-element",
			givenUids: map[string]UIDEntry{
				"highestPrioritySyncer": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
				"lowestPriority": {
					UID:     "456",
					Expires: time.Now(),
				},
			},
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"highestPriorityBidder"},
					{"lowestPriority"},
				},
				SyncersByBidder: map[string]Syncer{
					"highestPriorityBidder": fakeSyncer{
						key: "highestPrioritySyncer",
					},
					"lowestPriority": fakeSyncer{
						key: "lowestPriority",
					},
				},
				IsSyncerPriority: true,
			},
			expected: "lowestPriority",
		},
		{
			name: "multiple-uids-same-priority",
			givenUids: map[string]UIDEntry{
				"newerButSamePriority": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
				"olderButSamePriority": {
					UID:     "456",
					Expires: time.Now(),
				},
			},
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"newerButSamePriority", "olderButSamePriority"},
				},
				SyncersByBidder: map[string]Syncer{
					"newerButSamePriority": fakeSyncer{
						key: "newerButSamePriority",
					},
					"olderButSamePriority": fakeSyncer{
						key: "olderButSamePriority",
					},
				},
				IsSyncerPriority: true,
				TieEjector:       &OldestEjector{},
			},
			expected: "olderButSamePriority",
		},
		{
			name: "non-priority-uids-present",
			givenUids: map[string]UIDEntry{
				"higherPriority": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
				"lowestPriority": {
					UID:     "456",
					Expires: time.Now(),
				},
				"oldestNonPriority": {
					UID:     "456",
					Expires: time.Now(),
				},
				"newestNonPriority": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
			},
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"higherPriority"},
					{"lowestPriority"},
				},
				SyncersByBidder: map[string]Syncer{
					"higherPriority": fakeSyncer{
						key: "higherPriority",
					},
					"lowestPriority": fakeSyncer{
						key: "lowestPriority",
					},
					"oldestNonPriority": fakeSyncer{
						key: "oldestNonPriority",
					},
					"newestNonPriority": fakeSyncer{
						key: "newestNonPriority",
					},
				},
				IsSyncerPriority: true,
				TieEjector:       &OldestEjector{},
			},
			expected: "oldestNonPriority",
		},
		{
			name: "empty-priority-groups",
			givenUids: map[string]UIDEntry{
				"oldestNonPriority": {
					UID:     "456",
					Expires: time.Now(),
				},
				"newestNonPriority": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
			},
			givenEjector: &PriorityBidderEjector{
				SyncersByBidder: map[string]Syncer{
					"oldestNonPriority": fakeSyncer{
						key: "oldestNonPriority",
					},
					"newestNonPriority": fakeSyncer{
						key: "newestNonPriority",
					},
				},
				IsSyncerPriority: false,
				TieEjector:       &OldestEjector{},
			},
			expected: "oldestNonPriority",
		},
		{
			name: "one-priority-element",
			givenUids: map[string]UIDEntry{
				"onlyPriorityElement": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
			},
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"onlyPriorityElement"},
				},
				SyncersByBidder: map[string]Syncer{
					"onlyPriorityElement": fakeSyncer{
						key: "onlyPriorityElement",
					},
				},
				IsSyncerPriority: true,
			},
			expected: "onlyPriorityElement",
		},
		{
			name: "syncer-is-not-priority",
			givenUids: map[string]UIDEntry{
				"onlyPriorityElement": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
				"syncer": {
					UID:     "456",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
			},
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"onlyPriorityElement"},
				},
				SyncersByBidder: map[string]Syncer{
					"onlyPriorityElement": fakeSyncer{
						key: "onlyPriorityElement",
					},
				},
				IsSyncerPriority: false,
			},
			expectedError: errors.New("syncer key is not a priority, and there are only priority elements left"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			uidToDelete, err := test.givenEjector.Choose(test.givenUids)
			if test.expectedError != nil {
				assert.Equal(t, test.expectedError, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, uidToDelete)
			}
		})
	}
}

func TestOldestEjector(t *testing.T) {
	testCases := []struct {
		name      string
		givenUids map[string]UIDEntry
		expected  string
	}{
		{
			name: "multiple-elements",
			givenUids: map[string]UIDEntry{
				"newestElement": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
				"oldestElement": {
					UID:     "456",
					Expires: time.Now(),
				},
			},
			expected: "oldestElement",
		},
		{
			name: "one-element",
			givenUids: map[string]UIDEntry{
				"onlyElement": {
					UID:     "123",
					Expires: time.Now().Add((90 * 24 * time.Hour)),
				},
			},
			expected: "onlyElement",
		},
		{
			name:      "no-elements",
			givenUids: map[string]UIDEntry{},
			expected:  "",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			ejector := OldestEjector{}
			oldestElement, err := ejector.Choose(test.givenUids)
			assert.NoError(t, err)
			assert.Equal(t, test.expected, oldestElement)
		})
	}
}

func TestGetNonPriorityUids(t *testing.T) {
	syncersByBidder := map[string]Syncer{
		"syncerKey1": fakeSyncer{
			key: "syncerKey1",
		},
		"syncerKey2": fakeSyncer{
			key: "syncerKey2",
		},
		"syncerKey3": fakeSyncer{
			key: "syncerKey3",
		},
	}

	testCases := []struct {
		name                string
		givenUids           map[string]UIDEntry
		givenPriorityGroups [][]string
		expected            map[string]UIDEntry
	}{
		{
			name: "one-priority-group",
			givenUids: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
			givenPriorityGroups: [][]string{
				{"syncerKey1"},
			},
			expected: map[string]UIDEntry{
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
		},
		{
			name: "multiple-priority-groups",
			givenUids: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
			givenPriorityGroups: [][]string{
				{"syncerKey1"},
				{"syncerKey2"},
			},
			expected: map[string]UIDEntry{
				"syncerKey3": {
					UID: "789",
				},
			},
		},
		{
			name: "no-priority-groups",
			givenUids: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
			expected: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			uids := getNonPriorityUids(test.givenUids, test.givenPriorityGroups, syncersByBidder)
			assert.Equal(t, true, reflect.DeepEqual(test.expected, uids))
		})
	}
}

func TestGetPriorityUids(t *testing.T) {
	syncersByBidder := map[string]Syncer{
		"syncerKey1": fakeSyncer{
			key: "syncerKey1",
		},
		"syncerKey2": fakeSyncer{
			key: "syncerKey2",
		},
		"syncerKey3": fakeSyncer{
			key: "syncerKey3",
		},
	}

	testCases := []struct {
		name                     string
		givenUids                map[string]UIDEntry
		givenLowestPriorityGroup []string
		expected                 map[string]UIDEntry
	}{
		{
			name: "one-priority-element",
			givenUids: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
			givenLowestPriorityGroup: []string{"syncerKey1"},
			expected: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
			},
		},
		{
			name: "multiple-priority-elements",
			givenUids: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
			givenLowestPriorityGroup: []string{"syncerKey1", "syncerKey2"},
			expected: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
			},
		},
		{
			name: "no-priority-elements",
			givenUids: map[string]UIDEntry{
				"syncerKey1": {
					UID: "123",
				},
				"syncerKey2": {
					UID: "456",
				},
				"syncerKey3": {
					UID: "789",
				},
			},
			givenLowestPriorityGroup: []string{},
			expected:                 map[string]UIDEntry{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			uids := getPriorityUids(test.givenLowestPriorityGroup, test.givenUids, syncersByBidder)
			assert.Equal(t, true, reflect.DeepEqual(test.expected, uids))
		})
	}
}
