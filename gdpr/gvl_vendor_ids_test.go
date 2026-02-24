package gdpr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFetchLatestGVLVendorIDs(t *testing.T) {
	tests := []struct {
		description string
		settings    serverSettings
		expectedIDs map[uint16]struct{}
	}{
		{
			description: "Successful fetch with multiple vendors",
			settings: serverSettings{
				vendorListLatestVersion: 1,
				vendorLists: map[int]map[int]string{
					3: {
						1: MarshalVendorList(vendorList{
							GVLSpecificationVersion: 3,
							VendorListVersion:       1,
							Vendors: map[string]*vendor{
								"10":  {ID: 10},
								"20":  {ID: 20},
								"100": {ID: 100},
							},
						}),
					},
				},
			},
			expectedIDs: map[uint16]struct{}{
				10:  {},
				20:  {},
				100: {},
			},
		},
		{
			description: "Successful fetch with no vendors",
			settings: serverSettings{
				vendorListLatestVersion: 1,
				vendorLists: map[int]map[int]string{
					3: {
						1: MarshalVendorList(vendorList{
							GVLSpecificationVersion: 3,
							VendorListVersion:       1,
							Vendors:                 map[string]*vendor{},
						}),
					},
				},
			},
			expectedIDs: map[uint16]struct{}{},
		},
		{
			description: "Spec version 3 not available returns empty map",
			settings: serverSettings{
				vendorListLatestVersion: 1,
				vendorLists: map[int]map[int]string{
					2: {
						1: MarshalVendorList(vendorList{
							GVLSpecificationVersion: 2,
							VendorListVersion:       1,
							Vendors:                 map[string]*vendor{"5": {ID: 5}},
						}),
					},
				},
			},
			expectedIDs: map[uint16]struct{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(mockServer(tt.settings)))
			defer server.Close()

			result := FetchLatestGVLVendorIDs(context.Background(), server.Client(), testURLMaker(server))
			assert.Equal(t, tt.expectedIDs, result)
		})
	}
}

func TestFetchLatestGVLVendorIDsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result := FetchLatestGVLVendorIDs(context.Background(), server.Client(), testURLMaker(server))
	assert.Empty(t, result)
}

func TestFetchLatestGVLVendorIDsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	result := FetchLatestGVLVendorIDs(context.Background(), server.Client(), testURLMaker(server))
	assert.Empty(t, result)
}

func TestLiveGVLVendorIDsContains(t *testing.T) {
	tests := []struct {
		description string
		ids         map[uint16]struct{}
		checkID     uint16
		expected    bool
	}{
		{
			description: "Empty set returns true for any ID (safe fallback)",
			ids:         map[uint16]struct{}{},
			checkID:     42,
			expected:    true,
		},
		{
			description: "ID present in set",
			ids:         map[uint16]struct{}{10: {}, 20: {}},
			checkID:     10,
			expected:    true,
		},
		{
			description: "ID not present in set",
			ids:         map[uint16]struct{}{10: {}, 20: {}},
			checkID:     30,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			live := NewLiveGVLVendorIDs()
			live.Update(tt.ids)
			assert.Equal(t, tt.expected, live.Contains(tt.checkID))
		})
	}
}

func TestLiveGVLVendorIDsUpdateSkipsEmpty(t *testing.T) {
	live := NewLiveGVLVendorIDs()
	live.Update(map[uint16]struct{}{10: {}, 20: {}})

	// Updating with empty should retain previous set
	live.Update(map[uint16]struct{}{})
	assert.True(t, live.Contains(10))
	assert.True(t, live.Contains(20))
	assert.False(t, live.Contains(30))
}
