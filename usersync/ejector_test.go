package usersync

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChooseElementToEject(t *testing.T) {
	testCases := []struct {
		name          string
		givenUids     map[string]UIDEntry
		givenEjector  Ejector
		expected      string
		expectedError error
	}{
		{
			name: "priority-ejector",
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
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"adnxs"},
					{"oldestElement", "newerElement"},
				},
				SyncerKey: "adnxs",
			},
			expected: "oldestElement",
		},
		{
			name:      "priority-ejector-syncer-not-priority",
			givenUids: map[string]UIDEntry{},
			givenEjector: &PriorityBidderEjector{
				PriorityGroups: [][]string{
					{"syncerKey1"},
				},
				SyncerKey: "adnxs",
			},
			expectedError: errors.New("syncer key adnxs is not in priority groups"),
		},
		{
			name: "oldest-ejector",
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
			givenEjector: &OldestEjector{
				[]string{"newestElement", "oldestElement"},
			},
			expected: "oldestElement",
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

func TestGetOldestElement(t *testing.T) {
	testCases := []struct {
		name              string
		givenUids         map[string]UIDEntry
		givenFilteredKeys []string
		expected          string
	}{
		{
			name: "basic-oldest-element",
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
			givenFilteredKeys: []string{"newestElement", "oldestElement"},
			expected:          "oldestElement",
		},
		{
			name: "no-filtered-keys",
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
			givenFilteredKeys: []string{},
			expected:          "",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			oldestElement := getOldestElement(test.givenFilteredKeys, test.givenUids)
			assert.Equal(t, test.expected, oldestElement)
		})
	}
}

func TestGetNonPriorityKeys(t *testing.T) {
	testCases := []struct {
		name                string
		givenUids           map[string]UIDEntry
		givenPriorityGroups [][]string
		expected            []string
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
			expected: []string{"syncerKey2", "syncerKey3"},
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
			givenPriorityGroups: [][]string{},
			expected:            []string{"syncerKey1", "syncerKey2", "syncerKey3"},
		},
		{
			name:      "no-given-uids",
			givenUids: map[string]UIDEntry{},
			givenPriorityGroups: [][]string{
				{"syncerKey1"},
			},
			expected: []string{},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			keys := getNonPriorityKeys(test.givenUids, test.givenPriorityGroups)
			assert.Equal(t, true, reflect.DeepEqual(test.expected, keys))
		})
	}
}

func TestIsSyncerPriority(t *testing.T) {
	testCases := []struct {
		name                string
		givenSyncerKey      string
		givenPriorityGroups [][]string
		expected            bool
	}{
		{
			name:           "syncer-is-priority",
			givenSyncerKey: "adnxs",
			givenPriorityGroups: [][]string{
				{"adnxs"},
				{"2", "3"},
			},
			expected: true,
		},
		{
			name:           "syncer-is-not-priority",
			givenSyncerKey: "adnxs",
			givenPriorityGroups: [][]string{
				{"1"},
				{"2", "3"},
			},
			expected: false,
		},
		{
			name:           "no-syncer-given",
			givenSyncerKey: "",
			givenPriorityGroups: [][]string{
				{"1"},
				{"2", "3"},
			},
			expected: false,
		},
		{
			name:                "no-priority-groups-given",
			givenSyncerKey:      "adnxs",
			givenPriorityGroups: [][]string{},
			expected:            false,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			isPriority := isSyncerPriority(test.givenSyncerKey, test.givenPriorityGroups)
			assert.Equal(t, test.expected, isPriority)
		})
	}
}
