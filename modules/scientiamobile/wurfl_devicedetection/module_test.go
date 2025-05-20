package wurfl_devicedetection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		name        string
		configRaw   json.RawMessage
		expectedErr bool
		validate    func(t *testing.T, module interface{})
	}{
		{
			name: "Valid configuration",
			configRaw: json.RawMessage(`{
				"wurfl_snapshot_url": "http://example.com/wurfl-data",
				"wurfl_file_dir_path": "/tmp/wurfl",
				"allowed_publisher_ids": ["pub1", "pub2"],
				"ext_caps": true
			}`),
			expectedErr: false,
			validate: func(t *testing.T, module interface{}) {
				m, ok := module.(Module)
				assert.True(t, ok, "Module type assertion failed")
				assert.Equal(t, map[string]struct{}{"pub1": {}, "pub2": {}}, m.allowedPublisherIDs)
				assert.True(t, m.extCaps)
				assert.NotNil(t, m.we)
			},
		},
		{
			name:        "Invalid configuration - newConfig fails",
			configRaw:   json.RawMessage(`{ "wurfl_snapshot_url": "http://example.com/wurfl-data" }`), // Missing required fields
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			module, err := Builder(tc.configRaw, moduledeps.ModuleDeps{})

			if tc.expectedErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tc.validate != nil {
				tc.validate(t, module)
			}
		})
	}
}

func TestHandleEntrypointHook(t *testing.T) {
	tests := []struct {
		name              string
		module            Module
		payload           hookstage.EntrypointPayload
		expectedError     bool
		expectedModuleCtx map[string]map[string]string
	}{
		{
			name: "Publisher allowed with headers",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload: hookstage.EntrypointPayload{
				Body: []byte(`{"site":{"publisher":{"id":"pub1"}}}`),
				Request: &http.Request{
					Header: http.Header{
						"User-Agent": {"Mozilla/5.0"},
						"X-Test":     {"HeaderValue"},
					},
				},
			},
			expectedError: false,
			expectedModuleCtx: map[string]map[string]string{
				wurflHeaderCtxKey: {
					"User-Agent": "Mozilla/5.0",
					"X-Test":     "HeaderValue",
				},
			},
		},
		{
			name: "Publisher not allowed",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload: hookstage.EntrypointPayload{
				Body: []byte(`{"site":{"publisher":{"id":"pub2"}}}`),
				Request: &http.Request{
					Header: http.Header{
						"User-Agent": {"Mozilla/5.0"},
					},
				},
			},
			expectedError:     true,
			expectedModuleCtx: nil,
		},
		{
			name: "No publisher ID in payload",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload: hookstage.EntrypointPayload{
				Body: []byte(`{}`),
				Request: &http.Request{
					Header: http.Header{
						"User-Agent": {"Mozilla/5.0"},
					},
				},
			},
			expectedError:     true,
			expectedModuleCtx: nil,
		},
		{
			name: "Nil Request, publisher allowed",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload: hookstage.EntrypointPayload{
				Body:    []byte(`{"site":{"publisher":{"id":"pub1"}}}`),
				Request: nil,
			},
			expectedError: false,
			expectedModuleCtx: map[string]map[string]string{
				wurflHeaderCtxKey: {},
			},
		},
		{
			name: "Nil allowedPublisherIDs (all publishers allowed)",
			module: Module{
				allowedPublisherIDs: nil,
			},
			payload: hookstage.EntrypointPayload{
				Body: []byte(`{"site":{"publisher":{"id":"pub1"}}}`),
				Request: &http.Request{
					Header: http.Header{
						"X-Custom-Header": {"HeaderValue"},
					},
				},
			},
			expectedError: false,
			expectedModuleCtx: map[string]map[string]string{
				wurflHeaderCtxKey: {
					"X-Custom-Header": "HeaderValue",
				},
			},
		},
		{
			name: "Malformed payload",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload: hookstage.EntrypointPayload{
				Body: []byte(`{"site":{"publisher": `),
				Request: &http.Request{
					Header: http.Header{
						"X-Custom-Header": {"HeaderValue"},
					},
				},
			},
			expectedError:     true,
			expectedModuleCtx: nil,
		},
		{
			name: "Empty headers",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload: hookstage.EntrypointPayload{
				Body:    []byte(`{"site":{"publisher":{"id":"pub1"}}}`),
				Request: &http.Request{Header: http.Header{}},
			},
			expectedError: false,
			expectedModuleCtx: map[string]map[string]string{
				wurflHeaderCtxKey: {},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.module.HandleEntrypointHook(context.Background(), hookstage.ModuleInvocationContext{}, tc.payload)

			if tc.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result.ModuleContext)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result.ModuleContext)
				assert.Equal(t, tc.expectedModuleCtx[wurflHeaderCtxKey], result.ModuleContext[wurflHeaderCtxKey])
			}
		})
	}
}

func TestHandleRawAuctionHook(t *testing.T) {
	tests := []struct {
		name            string
		module          Module
		invocationCtx   hookstage.ModuleInvocationContext
		payload         hookstage.RawAuctionRequestPayload
		expectedErr     bool
		mutationErr     bool
		expectedPayload string
	}{
		{
			name: "Successful device enrichment without extCaps",
			module: Module{
				we: &mockWurflDeviceDetection{
					detectDeviceFunc: func(headers map[string]string) (wurflData, error) {
						return wurflData{
							"brand_name": "BrandX",
							"model_name": "ModelY",
							"is_mobile":  "true",
							"is_phone":   "true",
							"is_tablet":  "false",
						}, nil
					},
				},
				extCaps: false,
			},
			invocationCtx: hookstage.ModuleInvocationContext{
				ModuleContext: hookstage.ModuleContext{
					wurflHeaderCtxKey: map[string]string{
						"User-Agent": "Mozilla/5.0",
					},
				},
			},
			payload:     []byte(`{"device":{"ua":"Mozilla/5.0"}}`),
			expectedErr: false,
			expectedPayload: `{
				"device": {
					"ua": "Mozilla/5.0",
					"make": "BrandX",
					"model": "ModelY",
					"hwv": "ModelY",
					"devicetype": 1
				}
			}`,
		},
		{
			name: "Nil module context",
			module: Module{
				we: &mockWurflDeviceDetection{
					detectDeviceFunc: func(headers map[string]string) (wurflData, error) {
						return wurflData{
							"brand_name": "BrandX",
							"model_name": "ModelY",
							"is_mobile":  "true",
							"is_phone":   "true",
							"is_tablet":  "false",
						}, nil
					},
				},
				extCaps: false,
			},
			invocationCtx:   hookstage.ModuleInvocationContext{},
			payload:         []byte(`{"device":{"ua":"Mozilla/5.0"}}`),
			expectedErr:     true,
			expectedPayload: `{"device":{"ua":"Mozilla/5.0"}}`,
		},
		{
			name: "Successful device enrichment with extCaps",
			module: Module{
				we: &mockWurflDeviceDetection{
					detectDeviceFunc: func(headers map[string]string) (wurflData, error) {
						return wurflData{
							"brand_name": "BrandX",
							"model_name": "ModelY",
							"is_mobile":  "true",
							"is_phone":   "true",
							"is_tablet":  "false",
						}, nil
					},
				},
				extCaps: true,
			},
			invocationCtx: hookstage.ModuleInvocationContext{
				ModuleContext: hookstage.ModuleContext{
					wurflHeaderCtxKey: map[string]string{
						"User-Agent": "Mozilla/5.0",
					},
				},
			},
			payload:     []byte(`{"device":{"ua":"Mozilla/5.0"}}`),
			expectedErr: false,
			expectedPayload: `{
				"device": {
					"ua": "Mozilla/5.0",
					"make": "BrandX",
					"model": "ModelY",
					"hwv": "ModelY",
					"devicetype": 1,
					"ext": {
            "wurfl": {
							"brand_name": "BrandX",
							"model_name": "ModelY",
							"is_mobile":  "true",
							"is_phone":   "true",
							"is_tablet":  "false"
            } 
          }
				}
			}`,
		},
		{
			name: "Successful device enrichment with ext data and with extCaps",
			module: Module{
				we: &mockWurflDeviceDetection{
					detectDeviceFunc: func(headers map[string]string) (wurflData, error) {
						return wurflData{
							"brand_name": "BrandX",
							"model_name": "ModelY",
							"is_mobile":  "true",
							"is_phone":   "true",
							"is_tablet":  "false",
						}, nil
					},
				},
				extCaps: true,
			},
			invocationCtx: hookstage.ModuleInvocationContext{
				ModuleContext: hookstage.ModuleContext{
					wurflHeaderCtxKey: map[string]string{
						"User-Agent": "Mozilla/5.0",
					},
				},
			},
			payload:     []byte(`{"device":{"ua":"Mozilla/5.0", "ext": {"test": 1}}}`),
			expectedErr: false,
			expectedPayload: `{
				"device": {
					"ua": "Mozilla/5.0",
					"make": "BrandX",
					"model": "ModelY",
					"hwv": "ModelY",
					"devicetype": 1,
					"ext": {
            "test": 1,
            "wurfl": {
							"brand_name": "BrandX",
							"model_name": "ModelY",
							"is_mobile":  "true",
							"is_phone":   "true",
							"is_tablet":  "false"
            } 
          }
				}
			}`,
		},
		{
			name: "Failed device detection",
			module: Module{
				we: &mockWurflDeviceDetection{
					detectDeviceFunc: func(headers map[string]string) (wurflData, error) {
						return nil, errors.New("device detection error")
					},
				},
				extCaps: false,
			},
			invocationCtx: hookstage.ModuleInvocationContext{
				ModuleContext: hookstage.ModuleContext{
					wurflHeaderCtxKey: map[string]string{
						"User-Agent": "Mozilla/5.0",
					},
				},
			},
			payload:         []byte(`{"device":{"ua":"Mozilla/5.0"}}`),
			expectedErr:     false,
			mutationErr:     true,
			expectedPayload: `{"device":{"ua":"Mozilla/5.0"}}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.module.HandleRawAuctionHook(context.Background(), tc.invocationCtx, tc.payload)
			if tc.expectedErr {
				assert.Error(t, err)
				assert.JSONEq(t, tc.expectedPayload, string(tc.payload))
				return
			}
			assert.NoError(t, err)

			assert.Equal(t, len(result.ChangeSet.Mutations()), 1)

			assert.Equal(t, result.ChangeSet.Mutations()[0].Type(), hookstage.MutationUpdate)

			mutation := result.ChangeSet.Mutations()[0]
			// Apply mutation
			mutatedPayload, err := mutation.Apply(tc.payload)
			if tc.mutationErr {
				assert.Error(t, err)
				assert.JSONEq(t, tc.expectedPayload, string(tc.payload))
				return
			}
			assert.NoError(t, err)

			// Verify the mutated payload
			assert.JSONEq(t, tc.expectedPayload, string(mutatedPayload))
		})
	}
}

// Mock implementation of wurflDeviceDetection
type mockWurflDeviceDetection struct {
	detectDeviceFunc func(headers map[string]string) (wurflData, error)
}

func (m *mockWurflDeviceDetection) DeviceDetection(headers map[string]string) (wurflData, error) {
	return m.detectDeviceFunc(headers)
}

func TestIsPublisherAllowed(t *testing.T) {
	tests := []struct {
		name                string
		module              Module
		payload             []byte
		expected            bool
		allowedPublisherIDs map[string]bool
	}{
		{
			name: "Allowed publisher - site.publisher.id",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload:  []byte(`{"site":{"publisher":{"id":"pub1"}}}`),
			expected: true,
		},
		{
			name: "Disallowed publisher - site.publisher.id",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload:  []byte(`{"site":{"publisher":{"id":"pub2"}}}`),
			expected: false,
		},
		{
			name: "Allowed publisher - app.publisher.id",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub3": {}},
			},
			payload:  []byte(`{"app":{"publisher":{"id":"pub3"}}}`),
			expected: true,
		},
		{
			name: "Disallowed publisher - app.publisher.id",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub3": {}},
			},
			payload:  []byte(`{"app":{"publisher":{"id":"pub4"}}}`),
			expected: false,
		},
		{
			name: "Allowed publisher - dooh.publisher.id",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub5": {}},
			},
			payload:  []byte(`{"dooh":{"publisher":{"id":"pub5"}}}`),
			expected: true,
		},
		{
			name: "Disallowed publisher - dooh.publisher.id",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub5": {}},
			},
			payload:  []byte(`{"dooh":{"publisher":{"id":"pub6"}}}`),
			expected: false,
		},
		{
			name: "Empty payload - no publisher ID",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload:  []byte(`{}`),
			expected: false,
		},
		{
			name: "Nil allowedPublisherIDs - all publishers allowed",
			module: Module{
				allowedPublisherIDs: nil,
			},
			payload:  []byte(`{"site":{"publisher":{"id":"pub1"}}}`),
			expected: true,
		},
		{
			name: "Malformed JSON - no publisher ID",
			module: Module{
				allowedPublisherIDs: map[string]struct{}{"pub1": {}},
			},
			payload:  []byte(`{"site":{"publisher":{}}`), // Missing closing braces
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.module.isPublisherAllowed(tc.payload)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGetOrtb2Device(t *testing.T) {
	tests := []struct {
		name        string
		payload     []byte
		expectError bool
		expected    openrtb2.Device
	}{
		{
			name: "Valid device object",
			payload: []byte(`{
				"device": {
					"ua": "Mozilla/5.0",
					"ip": "192.168.0.1",
					"make": "Apple",
					"model": "iPhone"
				}
			}`),
			expectError: false,
			expected: openrtb2.Device{
				UA:    "Mozilla/5.0",
				IP:    "192.168.0.1",
				Make:  "Apple",
				Model: "iPhone",
			},
		},
		{
			name:        "Missing device field",
			payload:     []byte(`{}`),
			expectError: true,
			expected:    openrtb2.Device{},
		},
		{
			name:        "Invalid device type (non-object)",
			payload:     []byte(`{"device": "string_instead_of_object"}`),
			expectError: true,
			expected:    openrtb2.Device{},
		},
		{
			name:        "Malformed JSON",
			payload:     []byte(`{"device": { "ua": "Mozilla/5.0"`), // Missing closing braces
			expectError: true,
			expected:    openrtb2.Device{},
		},
		{
			name:        "Empty payload",
			payload:     []byte(``),
			expectError: true,
			expected:    openrtb2.Device{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			device, err := getOrtb2Device(tc.payload)

			if tc.expectError {
				assert.Error(t, err)
				assert.Equal(t, tc.expected, device)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, device)
			}
		})
	}
}
