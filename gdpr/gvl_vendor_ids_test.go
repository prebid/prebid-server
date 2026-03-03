package gdpr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/stretchr/testify/assert"
)

func TestFetchLatestGVLVendorIDs(t *testing.T) {
	tests := []struct {
		name        string
		settings    serverSettings
		expectedIDs map[uint16]struct{}
		expectError bool
	}{
		{
			name: "fetch-with-multiple-vendors",
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
			name: "fetch-with-no-vendors",
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
			name: "error-spec-v3-not-available",
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
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(mockServer(tt.settings)))
			defer server.Close()

			m := &metrics.MetricsEngineMock{}
			if tt.expectError {
				m.On("RecordLiveGVLFetch", false).Once()
			} else {
				m.On("RecordLiveGVLFetch", true).Once()
			}
			result := FetchLatestGVLVendorIDs(context.Background(), server.Client(), testURLMaker(server), m)
			assert.Equal(t, tt.expectedIDs, result)
			m.AssertExpectations(t)
		})
	}
}

func TestFetchLatestGVLVendorIDsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	m := &metrics.MetricsEngineMock{}
	m.On("RecordLiveGVLFetch", false).Once()
	result := FetchLatestGVLVendorIDs(context.Background(), server.Client(), testURLMaker(server), m)
	assert.Empty(t, result)
	m.AssertExpectations(t)
}

func TestFetchLatestGVLVendorIDsMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	m := &metrics.MetricsEngineMock{}
	m.On("RecordLiveGVLFetch", false).Once()
	result := FetchLatestGVLVendorIDs(context.Background(), server.Client(), testURLMaker(server), m)
	assert.Empty(t, result)
	m.AssertExpectations(t)
}

func TestLiveGVLVendorIDsContains(t *testing.T) {
	tests := []struct {
		name     string
		ids      map[uint16]struct{}
		checkID  uint16
		expected bool
	}{
		{
			name:     "empty-set-returns-true-for-any-id",
			ids:      map[uint16]struct{}{},
			checkID:  42,
			expected: true,
		},
		{
			name:     "id-present-in-set",
			ids:      map[uint16]struct{}{10: {}, 20: {}},
			checkID:  10,
			expected: true,
		},
		{
			name:     "id-not-present-in-set",
			ids:      map[uint16]struct{}{10: {}, 20: {}},
			checkID:  30,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
