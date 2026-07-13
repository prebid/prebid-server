package googlesdk

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/feature"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestGetSignalData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)

	tests := []struct {
		name     string
		input    string
		setup    func()
		expected *openrtb2.BidRequest
	}{
		{
			name:  "Empty body",
			input: "",
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name:  "Invalid JSON",
			input: "{invalid-json",
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name:  "Missing imp array",
			input: `{"someKey": "someValue"}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name:  "Empty imp array",
			input: `{"imp": []}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name:  "Missing ext in imp",
			input: `{"imp": [{"id": "1"}]}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name: "Wrong adapter ID",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [{
							"source_app": {"id": "wrong.adapter.id"},
							"data": "{\"id\":\"test-id\"}"
						}]
					}
				}]
			}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name: "Valid signal data",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [{
							"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
							"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
						}]
					}
				}]
			}`,
			expected: &openrtb2.BidRequest{
				ID: "test-id",
				App: &openrtb2.App{
					ID: "app-123",
				},
			},
		},
		{
			name: "Invalid signal data JSON",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [{
							"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
							"data": "{invalid-json}"
						}]
					}
				}]
			}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name: "Empty signal data",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [{
							"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
							"data": ""
						}]
					}
				}]
			}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name: "Multiple apps with valid signal",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [
							{
								"source_app": {"id": "other.app"},
								"data": "{\"id\":\"wrong-id\"}"
							},
							{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}
						]
					}
				}]
			}`,
			expected: &openrtb2.BidRequest{
				ID: "test-id",
				App: &openrtb2.App{
					ID: "app-123",
				},
			},
		},
		{
			name: "Invalid buyer_generated_request_data structure",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [
							"Invalid data"
						]
					}
				}]
			}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
		{
			name: "Invalid signal data unmarshalling failure",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [{
							"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
							"data": "eyJpZCI6"
						}]
					}
				}]
			}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.InvalidSignal)
			},
			expected: nil,
		},
		{
			name: "empty buyer generated request data",
			input: `{
				"imp": [{
					"ext": {
						"buyer_generated_request_data": [{],
					},
				}]
			}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("5890", "123", models.MissingSignal)
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result := getSignalData([]byte(tt.input), models.RequestCtx{
				MetricsEngine: mockEngine,
			}, &wrapperData{
				PublisherId: "5890",
				ProfileId:   "123",
			})

			if tt.expected == nil {
				assert.Nil(t, result, "Expected nil result for test: %s", tt.name)
				return
			}

			assert.NotNil(t, result, "Expected non-nil result for test: %s", tt.name)
			expectedJSON, err := json.Marshal(tt.expected)
			assert.NoError(t, err, "Failed to marshal expected value for test: %s", tt.name)
			resultJSON, err := json.Marshal(result)
			assert.NoError(t, err, "Failed to marshal result for test: %s", tt.name)
			assert.JSONEq(t, string(expectedJSON), string(resultJSON), "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestGetWrapperData(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *wrapperData
		expectedErr string
	}{
		{
			name:        "Empty body",
			input:       "",
			expected:    nil,
			expectedErr: "empty request body",
		},
		{
			name:        "Invalid JSON",
			input:       "{invalid-json",
			expected:    nil,
			expectedErr: "failed to get ad unit mapping",
		},
		{
			name:        "Missing imp array",
			input:       `{"someKey": "someValue"}`,
			expected:    nil,
			expectedErr: "failed to get ad unit mapping",
		},
		{
			name:        "Empty imp array",
			input:       `{"imp": []}`,
			expected:    nil,
			expectedErr: "failed to get ad unit mapping",
		},
		{
			name:        "Missing ext in imp",
			input:       `{"imp": [{"id": "1"}]}`,
			expected:    nil,
			expectedErr: "failed to get ad unit mapping",
		},
		{
			name:        "Missing Keyval in ad_unit_mapping",
			input:       `{"imp":[{"ext":{"ad_unit_mapping":[{"format":1}]}}]}`,
			expected:    nil,
			expectedErr: "wrapper data not found in ad unit mapping",
		},
		{
			name: "Valid wrapper data",
			input: `{
				"imp": [{
					"ext": {
						"ad_unit_mapping": [
							{
								"keyvals": [
									{"key": "publisher_id", "value": "12345"},
									{"key": "profile_id", "value": "67890"},
									{"key": "ad_unit_id", "value": "tag-123"}
								]
						}
						]
					}
				}]
			}`,
			expected: &wrapperData{
				PublisherId: "12345",
				ProfileId:   "67890",
				TagId:       "tag-123",
			},
			expectedErr: "",
		},
		{
			name: "Partial wrapper data",
			input: `{
				"imp": [{
					"ext": {
						"ad_unit_mapping": [
							{
								"keyvals": [
									{"key": "publisher_id", "value": "12345"}
								]
							}
						]
					}
				}]
			}`,
			expected: &wrapperData{
				PublisherId: "12345",
			},
			expectedErr: "",
		},
		{
			name: "Invalid Keyval structure",
			input: `{
				"imp": [{
					"ext": {
						"ad_unit_mapping": [{
							"Keyval": [
								{"key": "publisher_id"}
							]
						}]
					}
				}]
			}`,
			expected:    nil,
			expectedErr: "wrapper data not found in ad unit mapping",
		},
		{
			name: "No matching keys in Keyval",
			input: `{
				"imp": [{
					"ext": {
						"ad_unit_mapping": [{
							"keyvals": [
								{"key": "unknown_key", "value": "value"}
							]
						}]
					}
				}]
			}`,
			expected:    nil,
			expectedErr: "wrapper data not found in ad unit mapping",
		},
		{
			name: "Invalid adunit mapping structure",
			input: `{
				"imp": [{
					"ext": {
						"ad_unit_mapping": [{]
					}
				}]
			}`,
			expected:    nil,
			expectedErr: "failed to unmarshal ad unit mapping",
		},
		{
			name: "Not valid key value structure",
			input: `{
				"imp": [{
					"ext": {
						"ad_unit_mapping": [{
							"keyvals": [
								{"key": 123, "value": "12345"},
								{"key": "profile_id", "value": 67890},
								"abc"
							]
						}]
					}
				}]
			}`,
			expected:    nil,
			expectedErr: "wrapper data not found in ad unit mapping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getWrapperData([]byte(tt.input))
			if tt.expectedErr != "" {
				assert.Nil(t, result, "Expected nil result for test: %s", tt.name)
				assert.EqualError(t, err, tt.expectedErr, "Unexpected error for test: %s", tt.name)
				return
			}

			assert.NoError(t, err, "Unexpected error for test: %s", tt.name)
			assert.Equal(t, tt.expected, result, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyImpression(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		signalImps     []openrtb2.Imp
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "No request impressions",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
			},
			signalImps:     []openrtb2.Imp{},
			expectedResult: &openrtb2.BidRequest{Imp: []openrtb2.Imp{}},
		},
		{
			name: "No signal impressions",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1"},
				},
			},
			signalImps:     []openrtb2.Imp{},
			expectedResult: &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp1"}}},
		},
		{
			name: "Update DisplayManager and DisplayManagerVer",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1"},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					DisplayManager:    "dm",
					DisplayManagerVer: "1.0",
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:                "imp1",
						DisplayManager:    "dm",
						DisplayManagerVer: "1.0",
					},
				},
			},
		},
		{
			name: "Update_exp_from_signal_when_positive",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Exp: 0},
				},
			},
			signalImps: []openrtb2.Imp{
				{Exp: 300},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Exp: 300},
				},
			},
		},
		{
			name: "Does_not_overwrite_exp_when_signal_exp_is_zero",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Exp: 120},
				},
			},
			signalImps: []openrtb2.Imp{
				{Exp: 0},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Exp: 120},
				},
			},
		},
		{
			name: "Update ClickBrowser",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1"},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					ClickBrowser: ptrutil.ToPtr(int8(1)),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:           "imp1",
						ClickBrowser: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Update Banner",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", Banner: &openrtb2.Banner{}},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					Banner: &openrtb2.Banner{
						MIMEs: []string{"image/jpeg"},
						API:   []adcom1.APIFramework{adcom1.APIVPAID10},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Banner: &openrtb2.Banner{
							MIMEs: []string{"image/jpeg"},
							API:   []adcom1.APIFramework{adcom1.APIVPAID10},
						},
					},
				},
			},
		},
		{
			name: "Update Video",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Video: &openrtb2.Video{
							BAttr: []adcom1.CreativeAttribute{12233},
							MIMEs: []string{"video/mp4", "video/x-ms-wmv"},
							W:     ptrutil.ToPtr(int64(640)),
							H:     ptrutil.ToPtr(int64(480)),
						},
					},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					Video: &openrtb2.Video{
						MIMEs: []string{"video/mp4"},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Video: &openrtb2.Video{
							MIMEs: []string{"video/mp4"},
							BAttr: []adcom1.CreativeAttribute{12233},
						},
					},
				},
			},
		},
		{
			name: "Update Native and Secure",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1"},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					Native: &openrtb2.Native{
						Request: `{"ver": "1","privacy": 1}`,
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
						Native: &openrtb2.Native{
							Request: `{"ver": "1"}`,
						},
					},
				},
			},
		},
		{
			name: "Update Imp Extension and GPID",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "imp1", TagID: "tag-123"},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					Ext: []byte(`{"owsdk":{"ctaoverlay":1},"skadn": {"version": "2.0", "skoverlay": 1, "productpage": "page1", "versions": ["1.0"]}}`),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						TagID: "tag-123",
						Ext:   []byte(`{"skadn":{"versions":["1.0"],"version":"2.0","skoverlay":1,"productpage":"page1"},"owsdk":{"ctaoverlay":1}}`),
					},
				},
			},
		},
		{
			name: "update skadnetids from signal",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						TagID: "tag-123",
						Ext:   []byte(`{"skadn":{"skadnetids":["old1", "old2"]}}`),
					},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					Ext: []byte(`{"skadn":{"skadnetids": ["net1", "net2"]}}`),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:    "imp1",
						TagID: "tag-123",
						Ext:   []byte(`{"skadn":{"skadnetids":["net1", "net2"]}}`),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyImpression(tt.request, tt.signalImps)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyApp(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		signalApp      *openrtb2.App
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "Signal app is nil",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.com",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "sports,news",
				},
			},
			signalApp: nil,
			expectedResult: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.com",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "sports,news",
				},
			},
		},
		{
			name: "Request app is nil, signal app has data",
			request: &openrtb2.BidRequest{
				App: nil,
			},
			signalApp: &openrtb2.App{
				Domain:   "example.com",
				Paid:     ptrutil.ToPtr(int8(1)),
				Keywords: "sports,news",
			},
			expectedResult: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.com",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "sports,news",
				},
			},
		},
		{
			name: "Update domain and keywords, keep existing paid value",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Paid: ptrutil.ToPtr(int8(0)),
				},
			},
			signalApp: &openrtb2.App{
				Domain:   "example.org",
				Keywords: "tech,science",
			},
			expectedResult: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.org",
					Paid:     ptrutil.ToPtr(int8(0)),
					Keywords: "tech,science",
				},
			},
		},
		{
			name: "Signal app has empty domain and keywords, no changes",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.com",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "sports,news",
				},
			},
			signalApp: &openrtb2.App{
				Domain:   "",
				Keywords: "",
			},
			expectedResult: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.com",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "sports,news",
				},
			},
		},
		{
			name: "Signal app updates all fields",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{},
			},
			signalApp: &openrtb2.App{
				Domain:   "example.net",
				Paid:     ptrutil.ToPtr(int8(1)),
				Keywords: "movies,entertainment",
			},
			expectedResult: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Domain:   "example.net",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "movies,entertainment",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyApp(tt.request, tt.signalApp)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyDevice(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		signalDevice   *openrtb2.Device
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "Signal device is nil",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:    "Mozilla/5.0",
					Make:  "Apple",
					Model: "iPhone",
				},
			},
			signalDevice: nil,
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:    "Mozilla/5.0",
					Make:  "Apple",
					Model: "iPhone",
				},
			},
		},
		{
			name: "Request device is nil, signal device has data",
			request: &openrtb2.BidRequest{
				Device: nil,
			},
			signalDevice: &openrtb2.Device{
				IFA:   "07c387f2-e030-428f-8336-42f682150759",
				UA:    "Mozilla/5.0",
				Make:  "Samsung",
				Model: "Galaxy",
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IFA:   "07c387f2-e030-428f-8336-42f682150759",
					UA:    "Mozilla/5.0",
					Make:  "Samsung",
					Model: "Galaxy",
				},
			},
		},
		{
			name: "Update UA and Model, keep existing Make",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Make: "Google",
					IFA:  "07c387f2-e030-428f-8336-42f682150759",
				},
			},
			signalDevice: &openrtb2.Device{
				UA:    "Mozilla/5.0",
				Model: "Pixel",
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IFA:   "07c387f2-e030-428f-8336-42f682150759",
					UA:    "Mozilla/5.0",
					Make:  "Google",
					Model: "Pixel",
				},
			},
		},
		{
			name: "Signal device has empty UA and Model, no changes",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:    "Mozilla/5.0",
					Make:  "Apple",
					Model: "iPhone",
				},
			},
			signalDevice: &openrtb2.Device{
				UA:    "",
				Model: "",
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:    "Mozilla/5.0",
					Make:  "Apple",
					Model: "iPhone",
				},
			},
		},
		{
			name: "Signal device updates all fields",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{},
			},
			signalDevice: &openrtb2.Device{
				IFA:   "07c387f2-e030-428f-8336-42f682150759",
				UA:    "Mozilla/5.0",
				Make:  "Samsung",
				Model: "Galaxy",
				JS:    ptrutil.ToPtr(int8(1)),
				Geo: &openrtb2.Geo{
					Lat: ptrutil.ToPtr(float64(37.7749)),
					Lon: ptrutil.ToPtr(float64(-122.4194)),
				},
				HWV: "SM-G991B",
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					IFA:   "07c387f2-e030-428f-8336-42f682150759",
					UA:    "Mozilla/5.0",
					Make:  "Samsung",
					Model: "Galaxy",
					JS:    ptrutil.ToPtr(int8(1)),
					Geo: &openrtb2.Geo{
						Lat: ptrutil.ToPtr(float64(37.7749)),
						Lon: ptrutil.ToPtr(float64(-122.4194)),
					},
					HWV: "SM-G991B",
				},
			},
		},
		{
			name: "signal_device_copies_ipv6_devicetype_lmt_os_dims_lang_carrier_mccmnc_connectiontype",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA: "Mozilla/5.0",
				},
			},
			signalDevice: &openrtb2.Device{
				IPv6:           "2001:db8::1",
				DeviceType:     adcom1.DeviceType(4),
				Lmt:            ptrutil.ToPtr(int8(1)),
				OS:             "android",
				OSV:            "13.0.0",
				W:              1080,
				H:              2400,
				PxRatio:        2.75,
				Language:       "en_US",
				Carrier:        "MYTEL",
				MCCMNC:         "200-260",
				ConnectionType: ptrutil.ToPtr(adcom1.ConnectionType(2)),
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:             "Mozilla/5.0",
					IPv6:           "2001:db8::1",
					DeviceType:     adcom1.DeviceType(4),
					Lmt:            ptrutil.ToPtr(int8(1)),
					OS:             "android",
					OSV:            "13.0.0",
					W:              1080,
					H:              2400,
					PxRatio:        2.75,
					Language:       "en_US",
					Carrier:        "MYTEL",
					MCCMNC:         "200-260",
					ConnectionType: ptrutil.ToPtr(adcom1.ConnectionType(2)),
				},
			},
		},
		{
			name: "signal_device_geo_does_not_override_existing_lat_lon_but_merges_other_fields",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Geo: &openrtb2.Geo{
						Lat: ptrutil.ToPtr(float64(1.23)),
						Lon: ptrutil.ToPtr(float64(4.56)),
					},
				},
			},
			signalDevice: &openrtb2.Device{
				Geo: &openrtb2.Geo{
					Lat:       ptrutil.ToPtr(float64(9.87)),
					Lon:       ptrutil.ToPtr(float64(6.54)),
					Country:   "IN",
					Region:    "DL",
					Metro:     "NCR",
					City:      "New Delhi",
					ZIP:       "110001",
					UTCOffset: 330,
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					Geo: &openrtb2.Geo{
						Lat:       ptrutil.ToPtr(float64(1.23)),
						Lon:       ptrutil.ToPtr(float64(4.56)),
						Country:   "IN",
						Region:    "DL",
						Metro:     "NCR",
						City:      "New Delhi",
						ZIP:       "110001",
						UTCOffset: 330,
					},
				},
			},
		},
		{
			name: "signal_device_has_ifv",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA: "Mozilla/5.0",
				},
			},
			signalDevice: &openrtb2.Device{
				Ext: json.RawMessage(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:  "Mozilla/5.0",
					Ext: json.RawMessage(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
				},
			},
		},
		{
			name: "request_has_ifv_signal_does_not",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:  "Mozilla/5.0",
					Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
				},
			},
			signalDevice: &openrtb2.Device{
				Make: "Samsung",
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:   "Mozilla/5.0",
					Make: "Samsung",
					Ext:  json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
				},
			},
		},
		{
			name: "both_request_and_signal_have_ifv_signal_wins",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:  "Mozilla/5.0",
					Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
				},
			},
			signalDevice: &openrtb2.Device{
				Ext: json.RawMessage(`{"ifv":"SIGNAL-IFV-VALUE"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:  "Mozilla/5.0",
					Ext: json.RawMessage(`{"ifv":"SIGNAL-IFV-VALUE"}`),
				},
			},
		},
		{
			name: "signal_has_empty_ifv_overwrites_request_ifv",
			request: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:  "Mozilla/5.0",
					Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
				},
			},
			signalDevice: &openrtb2.Device{
				Ext: json.RawMessage(`{"ifv":""}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Device: &openrtb2.Device{
					UA:  "Mozilla/5.0",
					Ext: json.RawMessage(`{"ifv":""}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyDevice(tt.request, tt.signalDevice)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyRegs(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		signalRegs     *openrtb2.Regs
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "Signal regs is nil",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true}}`),
				},
			},
			signalRegs: nil,
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true}}`),
				},
			},
		},
		{
			name: "Request regs is nil, signal regs has data",
			request: &openrtb2.BidRequest{
				Regs: nil,
			},
			signalRegs: &openrtb2.Regs{
				Ext: []byte(`{"dsa":{"dsarequired":true,"pubrender":false},"gpp":"some-gpp-data","gpp_sid":["sid1","sid2"]}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true,"pubrender":false},"gpp":"some-gpp-data","gpp_sid":["sid1","sid2"]}`),
				},
			},
		},
		{
			name: "Update regs ext with signal regs data",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"datatopub":true}}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				Ext: []byte(`{"dsa":{"dsarequired":true,"pubrender":false},"gpp":"some-gpp-data","gpp_sid":["sid1","sid2"]}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"datatopub":true,"dsarequired":true,"pubrender":false},"gpp":"some-gpp-data","gpp_sid":["sid1","sid2"]}`),
				},
			},
		},
		{
			name: "Signal regs has empty ext, no changes",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true}}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				Ext: []byte(`{}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true}}`),
				},
			},
		},
		{
			name: "Signal regs updates all fields in ext",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				Ext: []byte(`{"dsa":{"dsarequired":true,"pubrender":false,"datatopub":true},"gpp":"some-gpp-data","gpp_sid":["sid1","sid2"]}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true,"pubrender":false,"datatopub":true},"gpp":"some-gpp-data","gpp_sid":["sid1","sid2"]}`),
				},
			},
		},
		{
			name: "copy_coppa_gdpr_us_privacy_from_signal",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"existing":1}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				COPPA: 1,
				Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"existing":1,"gdpr":1,"us_privacy":"1YNN"}`),
				},
			},
		},
		{
			name: "signal_overrides_request_coppa_gdpr_us_privacy_when_both_present",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"gdpr":0,"us_privacy":"1---","existing":1}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				COPPA: 1,
				Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN","existing":1}`),
				},
			},
		},
		{
			name: "signal_coppa_0_does_not_override_request_coppa",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				COPPA: 0,
				Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
				},
			},
		},
		{
			name: "signal_missing_gdpr_us_privacy_keeps_request_values",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN","existing":1}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				COPPA: 1,
				Ext:   []byte(`{"dsa":{"dsarequired":true}}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					COPPA: 1,
					Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN","existing":1,"dsa":{"dsarequired":true}}`),
				},
			},
		},
		{
			name: "signal_empty_us_privacy_does_not_override_request_us_privacy",
			request: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
				},
			},
			signalRegs: &openrtb2.Regs{
				Ext: []byte(`{"us_privacy":""}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyRegs(tt.request, tt.signalRegs)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}

func TestModifyRequestWithGoogleSDKParams_Privacy(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)

	signal := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{
			COPPA: 1,
			Ext:   []byte(`{"gdpr":1,"us_privacy":"1YNN"}`),
		},
	}

	signalBytes, err := json.Marshal(signal)
	assert.NoError(t, err)
	encodedSignal := base64.StdEncoding.EncodeToString(signalBytes)

	requestBody := `{
		"id": "123",
		"imp": [{
			"id": "imp1",
			"ext": {
				"ad_unit_mapping": [
					{
						"keyvals": [
							{"key": "publisher_id", "value": "12345"},
							{"key": "profile_id", "value": "67890"},
							{"key": "ad_unit_id", "value": "tag-123"}
						]
					}
				],
				"buyer_generated_request_data": [{
					"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
					"data": "` + encodedSignal + `"
				}]
			}
		}]
	}`

	result := ModifyRequestWithGoogleSDKParams([]byte(requestBody), models.RequestCtx{MetricsEngine: mockEngine}, nil)

	modified := &openrtb2.BidRequest{}
	err = json.Unmarshal(result, modified)
	assert.NoError(t, err)

	if assert.NotNil(t, modified.Regs) {
		assert.Equal(t, int8(1), modified.Regs.COPPA)
		gdpr, err := jsonparser.GetInt(modified.Regs.Ext, "gdpr")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), gdpr)

		usPrivacy, err := jsonparser.GetString(modified.Regs.Ext, "us_privacy")
		assert.NoError(t, err)
		assert.Equal(t, "1YNN", usPrivacy)
	}
}
func TestModifySource(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		signalSource   *openrtb2.Source
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "Signal source is nil",
			request: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic"}`),
				},
			},
			signalSource: nil,
			expectedResult: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic"}`),
				},
			},
		},
		{
			name: "Request source is nil, signal source has data",
			request: &openrtb2.BidRequest{
				Source: nil,
			},
			signalSource: &openrtb2.Source{
				Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
				},
			},
		},
		{
			name: "Update source ext with signal source data",
			request: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"existing"}`),
				},
			},
			signalSource: &openrtb2.Source{
				Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
				},
			},
		},
		{
			name: "Signal source has empty ext, no changes",
			request: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic"}`),
				},
			},
			signalSource: &openrtb2.Source{
				Ext: []byte(`{}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic"}`),
				},
			},
		},
		{
			name: "Signal source updates all fields in ext",
			request: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{}`),
				},
			},
			signalSource: &openrtb2.Source{
				Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifySource(tt.request, tt.signalSource)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyUser(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		signalUser     *openrtb2.User
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "Signal user is nil",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue"}`),
				},
			},
			signalUser: nil,
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue"}`),
				},
			},
		},
		{
			name: "Request user is nil, signal user has data",
			request: &openrtb2.BidRequest{
				User: nil,
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
				},
			},
		},
		{
			name: "Update user ext with signal user data",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue"}`),
				},
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue","sessionduration":3600,"impdepth":5}`),
				},
			},
		},
		{
			name: "copy_consent_from_signal_user_ext",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue"}`),
				},
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{"consent":"BOEFEAyOEFEAyAHABDENAI4AAAB9vABAASA"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue","consent":"BOEFEAyOEFEAyAHABDENAI4AAAB9vABAASA"}`),
				},
			},
		},
		{
			name: "signal_consent_overrides_request_consent_when_both_present",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"consent":"OLD"}`),
				},
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{"consent":"NEW"}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"consent":"NEW"}`),
				},
			},
		},
		{
			name: "empty_signal_consent_does_not_override_request_consent",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"consent":"OLD"}`),
				},
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{"consent":""}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"consent":"OLD"}`),
				},
			},
		},
		{
			name: "Signal user has empty ext, no changes",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue"}`),
				},
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"existingKey":"existingValue"}`),
				},
			},
		},
		{
			name: "Signal user updates all fields in ext",
			request: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{}`),
				},
			},
			signalUser: &openrtb2.User{
				Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
			},
			expectedResult: &openrtb2.BidRequest{
				User: &openrtb2.User{
					Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyUser(tt.request, tt.signalUser)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyRequestWithGoogleSDKParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockEngine := mock_metrics.NewMockMetricsEngine(ctrl)

	tests := []struct {
		name           string
		requestBody    string
		setup          func()
		features       feature.Features
		expectedResult string
		expectError    bool
	}{
		{
			name:           "empty request",
			requestBody:    "",
			expectedResult: "",
		},
		{
			name:           "Invalid sdk request",
			requestBody:    `{`,
			expectedResult: `{`,
			expectError:    true,
		},
		{
			name: "Valid request with wrapper and signal data",
			requestBody: `{
					"id": "123",
					"imp": [{
					    "id": "imp1",
						"ext": {
							"ad_unit_mapping": [
								{
									"keyvals": [
										{"key": "publisher_id", "value": "12345"},
										{"key": "profile_id", "value": "67890"},
										{"key": "ad_unit_id", "value": "tag-123"}
									]
								}
							],
							"buyer_generated_request_data": [{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}]
						}
					}]
				}`,
			setup: nil,
			expectedResult: `{
					"id": "123",
					"imp": [{
						"id": "imp1",
						"tagid": "tag-123",
						"secure": 1,
						"ext": {
							"ad_unit_mapping": [
								{
									"keyvals": [
										{"key": "publisher_id", "value": "12345"},
										{"key": "profile_id", "value": "67890"},
										{"key": "ad_unit_id", "value": "tag-123"}
									]
								}
							],
							"buyer_generated_request_data": [{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}]
						}
					}],
					"app": {
						"publisher": {
							"id": "12345"
						}
					},
					"ext": {
						"prebid": {
							"bidderparams": {
								"pubmatic": {
									"wrapper": {
										"profileid": 67890
									}
								}
							}
						}
					}
				}`,
		},
		{
			name: "Missing wrapper data",
			requestBody: `{
					"imp": [{
						"ext": {
							"buyer_generated_request_data": [{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}]
						}
					}]
				}`,
			expectedResult: `{
					"imp": [{
						"ext": {
							"buyer_generated_request_data": [{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}]
						}
					}]
				}`,
		},
		{
			name: "Missing signal data",
			requestBody: `{
					"id": "123",
					"imp": [{
						"id": "imp1",
						"tagid": "tag-gp-123",
						"metric": [
							{
								"type": "ow"
							}
						],
						"ext": {
							"ad_unit_mapping": [
								{
									"keyvals": [
										{"key": "publisher_id", "value": "12345"},
										{"key": "profile_id", "value": "67890"},
										{"key": "ad_unit_id", "value": "tag-123"}
									]
								}
							]
						}
					}]
				}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("12345", "67890", models.MissingSignal)
			},
			expectedResult: `{
					"id": "123",
					"imp": [{
						"id": "imp1",
						"tagid": "tag-123",
						"secure": 1,
						"ext": {
							"ad_unit_mapping": [
								{
									"keyvals": [
										{"key": "publisher_id", "value": "12345"},
										{"key": "profile_id", "value": "67890"},
										{"key": "ad_unit_id", "value": "tag-123"}
									]
								}
							],
							"gpid": "tag-gp-123"
						}
					}],
					"app": {
						"publisher": {
							"id": "12345"
						}
					},
					"ext": {
						"prebid": {
							"bidderparams": {
								"pubmatic": {
									"wrapper": {
										"profileid": 67890
									}
								}
							}
						}
					}
				}`,
		},
		{
			name: "Request with native and empty signal data",
			requestBody: `{
							"id": "123",
							"imp": [
								{
								"id": "imp1",
								"tagid": "tag-gp-123",
								"native": {
									"request": "{\"ver\": \"1\",\"privacy\": 1}"
								},
								"ext": {
									"ad_unit_mapping": [
									{
										"keyvals": [
										{
											"key": "publisher_id",
											"value": "12345"
										},
										{
											"key": "profile_id",
											"value": "67890"
										},
										{
											"key": "ad_unit_id",
											"value": "tag-123"
										}
										]
									}
									]
								}
								}
							]
						}`,
			setup: func() {
				mockEngine.EXPECT().RecordSignalDataStatus("12345", "67890", models.MissingSignal)
			},
			expectedResult: `{
				"id": "123",
				"imp": [{
					"id": "imp1",
					"tagid": "tag-123",
					"secure": 1,
					"ext": {
						"ad_unit_mapping": [
							{
								"keyvals": [
									{"key": "publisher_id", "value": "12345"},
									{"key": "profile_id", "value": "67890"},
									{"key": "ad_unit_id", "value": "tag-123"}
								]
							}
						],
						"gpid": "tag-gp-123"
					}
				}],
				"app": {
					"publisher": {
						"id": "12345"
					}
				},
				"ext": {
					"prebid": {
						"bidderparams": {
							"pubmatic": {
								"wrapper": {
									"profileid": 67890
								}
							}
						}
					}
				}
			}`,
		},
		{
			name: "Valid request with flexslot for banner",
			requestBody: `{
					"id": "123",
					"imp": [{
					    "id": "imp1",
						"banner": {
							"w": 300,
							"h": 250,
							"ext": {
								"flexslot": {
                        			"wmin": 10,
                        			"wmax": 1000,
                        			"hmin": 10,
                        			"hmax": 1000
                    			}
							}
						},
						"ext": {
							"ad_unit_mapping": [
								{
									"keyvals": [
										{"key": "publisher_id", "value": "12345"},
										{"key": "profile_id", "value": "67890"},
										{"key": "ad_unit_id", "value": "tag-123"}
									]
								}
							],
							"buyer_generated_request_data": [{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}]
						}
					}]
				}`,
			setup: nil,
			features: feature.Features{
				feature.FeatureNameGoogleSDK: []feature.Feature{
					{
						Name: feature.FeatureFlexSlot,
						Data: []string{"320x90"},
					},
				},
			},
			expectedResult: `{
					"id": "123",
					"imp": [{
						"id": "imp1",
						"tagid": "tag-123",
						"banner": {
							"w": 300,
							"h": 250,
							"format": [{"w": 320, "h": 90}],
							"ext": {
								"flexslot": {
                        			"wmin": 10,
                        			"wmax": 1000,
                        			"hmin": 10,
                        			"hmax": 1000
                    			}
							}
						},
						"secure": 1,
						"ext": {
							"ad_unit_mapping": [
								{
									"keyvals": [
										{"key": "publisher_id", "value": "12345"},
										{"key": "profile_id", "value": "67890"},
										{"key": "ad_unit_id", "value": "tag-123"}
									]
								}
							],
							"buyer_generated_request_data": [{
								"source_app": {"id": "com.google.ads.mediation.pubmatic.PubMaticMediationAdapter"},
								"data": "eyJpZCI6InRlc3QtaWQiLCJhcHAiOnsiaWQiOiJhcHAtMTIzIn19"
							}]
						}
					}],
					"app": {
						"publisher": {
							"id": "12345"
						}
					},
					"ext": {
						"prebid": {
							"bidderparams": {
								"pubmatic": {
									"wrapper": {
										"profileid": 67890
									}
								}
							}
						}
					}
				}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			result := ModifyRequestWithGoogleSDKParams([]byte(tt.requestBody), models.RequestCtx{
				MetricsEngine: mockEngine,
			}, tt.features)

			if tt.expectError {
				assert.Equal(t, tt.expectedResult, string(result), "Actual and expected result does not match: %s", tt.name)
				return
			}

			if tt.expectedResult == "" {
				assert.Empty(t, result, "Expected nil result for test: %s", tt.name)
				return
			}
			assert.JSONEq(t, tt.expectedResult, string(result), "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyRequestWithStaticData(t *testing.T) {
	tests := []struct {
		name           string
		request        *openrtb2.BidRequest
		expectedResult *openrtb2.BidRequest
	}{
		{
			name: "No impressions in request",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
			},
		},
		{
			name: "Set secure to 1 and add gpid",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						TagID: "tag-123",
						Ext:   []byte(`{}`),
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						TagID:  "tag-123",
						Secure: ptrutil.ToPtr(int8(1)),
						Ext:    []byte(`{"gpid":"tag-123"}`),
					},
				},
			},
		},
		{
			name: "Remove metric from impression",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Metric: []openrtb2.Metric{
							{Type: "viewability"},
						},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Metric: nil,
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Remove banner if impression is rewarded and both banner and video are present",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Rwdd:   1,
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Rwdd:   1,
						Banner: nil,
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Do not remove banner if impression is not rewarded",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Rwdd:   0,
						Banner: &openrtb2.Banner{},
						Video:  &openrtb2.Video{},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Rwdd:   0,
						Banner: &openrtb2.Banner{},
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "remove_unsupported_fields_from_banner",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{

						Banner: &openrtb2.Banner{
							WMin: 100,
							WMax: 200,
							HMax: 300,
							HMin: 400,
						},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Banner: &openrtb2.Banner{
							WMin: 0,
							WMax: 0,
							HMax: 0,
							HMin: 0,
						},
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Convert consented_providers from string array to int array",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"consented_providers_settings":{"consented_providers":["1","2","3"]}}`),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"consented_providers_settings":{"consented_providers":[1,2,3]}}`),
				},
			},
		},
		{
			name: "Handle mixed valid and invalid string values in consented_providers",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"consented_providers_settings":{"consented_providers":["1","invalid","3"]},"other_field":"value"}`),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"consented_providers_settings":{"consented_providers":[1,3]},"other_field":"value"}`),
				},
			},
		},
		{
			name: "No change when consented_providers_settings is missing",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"other_field":"value"}`),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"other_field":"value"}`),
				},
			},
		},
		{
			name: "No change when consented_providers is not an array",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"consented_providers_settings":{"consented_providers":"not_an_array"}}`),
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
				User: &openrtb2.User{
					Ext: []byte(`{"consented_providers_settings":{"consented_providers":"not_an_array"}}`),
				},
			},
		},
		{
			name: "Remove native and video from request",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Native: &openrtb2.Native{
							Request: `{"native":{"layout":1}}`,
						},
						Video: &openrtb2.Video{
							MIMEs: []string{"video/mp4"},
						},
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Native: nil,
						Video:  nil,
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Set gpid from dfp_ad_unit_code",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Ext: []byte(`{"dfp_ad_unit_code":"abc"}`),
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Ext:    []byte(`{"dfp_ad_unit_code":"abc","gpid":"abc"}`),
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Set gpid from tag id",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						TagID: "def",
						Ext:   []byte(`{}`),
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						Ext:    []byte(`{"gpid":"def"}`),
						TagID:  "def",
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "Do not overwrite gpid from dfp_ad_unit_code if gpid is already present",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						TagID: "def",
						Ext:   []byte(`{"dfp_ad_unit_code":"abc","gpid":"original_value"}`),
					},
				},
			},
			expectedResult: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						TagID:  "def",
						Ext:    []byte(`{"dfp_ad_unit_code":"abc","gpid":"original_value"}`),
						Secure: ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyRequestWithStaticData(tt.request)
			assert.Equal(t, tt.expectedResult, tt.request, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestModifyRequestWithSignalData(t *testing.T) {
	type args struct {
		request    *openrtb2.BidRequest
		signalData *openrtb2.BidRequest
	}
	tests := []struct {
		name     string
		args     args
		expected *openrtb2.BidRequest
	}{
		{
			name: "Nil request does nothing",
			args: args{
				request:    nil,
				signalData: &openrtb2.BidRequest{},
			},
			expected: nil,
		},
		{
			name: "All fields updated from signalData",
			args: args{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{
							ID:     "imp1",
							TagID:  "tag-1",
							Ext:    []byte(`{}`),
							Banner: &openrtb2.Banner{},
							Video: &openrtb2.Video{
								BAttr: []adcom1.CreativeAttribute{adcom1.CreativeAttribute(1)},
							},
							Native: &openrtb2.Native{
								Request: `{"ver":"1","privacy":1}`,
							},
						},
					},
					App:    &openrtb2.App{},
					Device: &openrtb2.Device{},
					Regs:   &openrtb2.Regs{Ext: []byte(`{}`)},
					Source: &openrtb2.Source{Ext: []byte(`{}`)},
					User:   &openrtb2.User{Ext: []byte(`{}`)},
					Ext:    []byte(`{}`),
				},
				signalData: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{
							DisplayManager:    "dm",
							DisplayManagerVer: "1.0",
							ClickBrowser:      ptrutil.ToPtr(int8(1)),
							Banner: &openrtb2.Banner{
								MIMEs: []string{"image/png"},
								API:   []adcom1.APIFramework{adcom1.APIVPAID10},
							},
							Video: &openrtb2.Video{
								MIMEs: []string{"video/mp4"},
							},
							Native: &openrtb2.Native{
								Request: `{"ver":"2","privacy":1}`,
							},
							Ext: []byte(`{"skadn":{"version":"2.0"}}`),
						},
					},
					App: &openrtb2.App{
						Domain:   "example.com",
						Paid:     ptrutil.ToPtr(int8(1)),
						Keywords: "sports,news",
					},
					Device: &openrtb2.Device{
						IFA:   "07c387f2-e030-428f-8336-42f682150759",
						IP:    "127.0.0.1",
						UA:    "Mozilla/5.0",
						Make:  "Samsung",
						Model: "Galaxy",
						JS:    ptrutil.ToPtr(int8(1)),
						Geo: &openrtb2.Geo{
							Lat: ptrutil.ToPtr(float64(1.23)),
							Lon: ptrutil.ToPtr(float64(4.56)),
						},
						HWV: "SM-G991B",
					},
					Regs: &openrtb2.Regs{
						Ext: []byte(`{"dsa":{"dsarequired":true},"gpp":"gppdata"}`),
					},
					Source: &openrtb2.Source{
						Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
					},
					User: &openrtb2.User{
						Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
					},
					Ext: []byte(`{"wrapper":{"clientconfig":{"foo":"bar"}}}`),
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:                "imp1",
						TagID:             "tag-1",
						DisplayManager:    "dm",
						DisplayManagerVer: "1.0",
						ClickBrowser:      ptrutil.ToPtr(int8(1)),
						Banner: &openrtb2.Banner{
							MIMEs: []string{"image/png"},
							API:   []adcom1.APIFramework{adcom1.APIVPAID10},
						},
						Video: &openrtb2.Video{
							MIMEs: []string{"video/mp4"},
							BAttr: []adcom1.CreativeAttribute{adcom1.CreativeAttribute(1)},
						},
						Native: &openrtb2.Native{
							Request: `{"ver":"2"}`,
						},
						Ext: []byte(`{"skadn":{"version":"2.0"}}`),
					},
				},
				App: &openrtb2.App{
					Domain:   "example.com",
					Paid:     ptrutil.ToPtr(int8(1)),
					Keywords: "sports,news",
				},
				Device: &openrtb2.Device{
					IFA:   "07c387f2-e030-428f-8336-42f682150759",
					IP:    "127.0.0.1",
					UA:    "Mozilla/5.0",
					Make:  "Samsung",
					Model: "Galaxy",
					JS:    ptrutil.ToPtr(int8(1)),
					Geo: &openrtb2.Geo{
						Lat: ptrutil.ToPtr(float64(1.23)),
						Lon: ptrutil.ToPtr(float64(4.56)),
					},
					HWV: "SM-G991B",
				},
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"dsa":{"dsarequired":true},"gpp":"gppdata"}`),
				},
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"pubmatic","omidpv":"1.3.15"}`),
				},
				User: &openrtb2.User{
					Ext: []byte(`{"sessionduration":3600,"impdepth":5}`),
				},
				Ext: []byte(`{"wrapper":{"clientconfig":{"foo":"bar"}}}`),
			},
		},
		{
			name: "SignalData with nil subfields does not overwrite request fields",
			args: args{
				request: &openrtb2.BidRequest{
					Imp: []openrtb2.Imp{
						{ID: "imp1"},
					},
					App:    &openrtb2.App{Domain: "keep.com"},
					Device: &openrtb2.Device{UA: "keep-ua"},
					Regs:   &openrtb2.Regs{Ext: []byte(`{"dsa":{"dsarequired":false}}`)},
					Source: &openrtb2.Source{Ext: []byte(`{"omidpn":"keep"}`)},
					User:   &openrtb2.User{Ext: []byte(`{"existing":1}`)},
					Ext:    []byte(`{"existing":"yes"}`),
				},
				signalData: &openrtb2.BidRequest{
					Imp:    nil,
					App:    nil,
					Device: nil,
					Regs:   nil,
					Source: nil,
					User:   nil,
					Ext:    nil,
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "imp1",
					},
				},
				App:    &openrtb2.App{Domain: "keep.com"},
				Device: &openrtb2.Device{UA: "keep-ua"},
				Regs:   &openrtb2.Regs{Ext: []byte(`{"dsa":{"dsarequired":false}}`)},
				Source: &openrtb2.Source{Ext: []byte(`{"omidpn":"keep"}`)},
				User:   &openrtb2.User{Ext: []byte(`{"existing":1}`)},
				Ext:    []byte(`{"existing":"yes"}`),
			},
		},
		{
			name: "SignalData with only Ext updates request Ext",
			args: args{
				request: &openrtb2.BidRequest{
					Ext: []byte(`{"existing":"yes"}`),
				},
				signalData: &openrtb2.BidRequest{
					Ext: []byte(`{"wrapper":{"clientconfig":{"foo":"bar"}}}`),
				},
			},
			expected: &openrtb2.BidRequest{
				Ext: []byte(`{"existing":"yes","wrapper":{"clientconfig":{"foo":"bar"}}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy request to avoid mutation between tests
			var reqCopy *openrtb2.BidRequest
			if tt.args.request != nil {
				b, _ := json.Marshal(tt.args.request)
				_ = json.Unmarshal(b, &reqCopy)
			}
			modifyRequestWithSignalData(reqCopy, tt.args.signalData)
			if tt.expected == nil {
				assert.Nil(t, reqCopy)
				return
			}
			// Compare JSON for Ext fields to avoid ordering issues
			if reqCopy != nil && tt.expected != nil {
				if len(reqCopy.Ext) > 0 || len(tt.expected.Ext) > 0 {
					assert.JSONEq(t, string(tt.expected.Ext), string(reqCopy.Ext), "Ext mismatch for test: %s", tt.name)
					reqCopy.Ext = nil
					tt.expected.Ext = nil
				}
				if reqCopy.Regs != nil && tt.expected.Regs != nil && (len(reqCopy.Regs.Ext) > 0 || len(tt.expected.Regs.Ext) > 0) {
					assert.JSONEq(t, string(tt.expected.Regs.Ext), string(reqCopy.Regs.Ext), "Regs.Ext mismatch for test: %s", tt.name)
					reqCopy.Regs.Ext = nil
					tt.expected.Regs.Ext = nil
				}
				if reqCopy.Source != nil && tt.expected.Source != nil && (len(reqCopy.Source.Ext) > 0 || len(tt.expected.Source.Ext) > 0) {
					assert.JSONEq(t, string(tt.expected.Source.Ext), string(reqCopy.Source.Ext), "Source.Ext mismatch for test: %s", tt.name)
					reqCopy.Source.Ext = nil
					tt.expected.Source.Ext = nil
				}
				if reqCopy.User != nil && tt.expected.User != nil && (len(reqCopy.User.Ext) > 0 || len(tt.expected.User.Ext) > 0) {
					assert.JSONEq(t, string(tt.expected.User.Ext), string(reqCopy.User.Ext), "User.Ext mismatch for test: %s", tt.name)
					reqCopy.User.Ext = nil
					tt.expected.User.Ext = nil
				}
				if len(reqCopy.Imp) > 0 && len(tt.expected.Imp) > 0 && (len(reqCopy.Imp[0].Ext) > 0 || len(tt.expected.Imp[0].Ext) > 0) {
					assert.JSONEq(t, string(tt.expected.Imp[0].Ext), string(reqCopy.Imp[0].Ext), "Imp[0].Ext mismatch for test: %s", tt.name)
					reqCopy.Imp[0].Ext = nil
					tt.expected.Imp[0].Ext = nil
				}
			}
			assert.Equal(t, tt.expected, reqCopy, "Unexpected result for test: %s", tt.name)
		})
	}
}
func TestWrapperData_setProfileID(t *testing.T) {
	tests := []struct {
		name         string
		wrapper      *wrapperData
		request      *openrtb2.BidRequest
		expectedJSON string
	}{
		{
			name: "ProfileId is empty, does nothing",
			wrapper: &wrapperData{
				ProfileId: "",
			},
			request:      &openrtb2.BidRequest{Ext: []byte(`{}`)},
			expectedJSON: `{}`,
		},
		{
			name: "Request.Ext is nil, sets profileid",
			wrapper: &wrapperData{
				ProfileId: "67890",
			},
			request:      &openrtb2.BidRequest{Ext: nil},
			expectedJSON: `{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":67890}}}}}`,
		},
		{
			name: "Request.Ext is empty, sets profileid",
			wrapper: &wrapperData{
				ProfileId: "12345",
			},
			request:      &openrtb2.BidRequest{Ext: []byte(`{}`)},
			expectedJSON: `{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":12345}}}}}`,
		},
		{
			name: "Request.Ext has existing prebid, merges profileid",
			wrapper: &wrapperData{
				ProfileId: "555",
			},
			request: &openrtb2.BidRequest{
				Ext: []byte(`{"prebid":{"other":"value"}}`),
			},
			expectedJSON: `{"prebid":{"other":"value","bidderparams":{"pubmatic":{"wrapper":{"profileid":555}}}}}`,
		},
		{
			name: "Request.Ext has existing profileid, overwrites it",
			wrapper: &wrapperData{
				ProfileId: "999",
			},
			request: &openrtb2.BidRequest{
				Ext: []byte(`{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":123}}}}}`),
			},
			expectedJSON: `{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":999}}}}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wrapper.setProfileID(tt.request)
			assert.JSONEq(t, tt.expectedJSON, string(tt.request.Ext), "Unexpected Ext for test: %s", tt.name)
		})
	}
}
func TestWrapperData_setPublisherId(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *wrapperData
		request     *openrtb2.BidRequest
		expectedApp *openrtb2.App
	}{
		{
			name: "PublisherId is empty, does nothing",
			wrapper: &wrapperData{
				PublisherId: "",
			},
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{ID: "old-id"},
				},
			},
			expectedApp: &openrtb2.App{
				Publisher: &openrtb2.Publisher{ID: "old-id"},
			},
		},
		{
			name: "App is nil, sets PublisherId",
			wrapper: &wrapperData{
				PublisherId: "pub-123",
			},
			request:     &openrtb2.BidRequest{App: nil},
			expectedApp: &openrtb2.App{Publisher: &openrtb2.Publisher{ID: "pub-123"}},
		},
		{
			name: "App.Publisher is nil, sets PublisherId",
			wrapper: &wrapperData{
				PublisherId: "pub-456",
			},
			request:     &openrtb2.BidRequest{App: &openrtb2.App{Publisher: nil}},
			expectedApp: &openrtb2.App{Publisher: &openrtb2.Publisher{ID: "pub-456"}},
		},
		{
			name: "App.Publisher exists, overwrites PublisherId",
			wrapper: &wrapperData{
				PublisherId: "pub-789",
			},
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Publisher: &openrtb2.Publisher{ID: "old-id"},
				},
			},
			expectedApp: &openrtb2.App{
				Publisher: &openrtb2.Publisher{ID: "pub-789"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wrapper.setPublisherId(tt.request)
			if tt.expectedApp == nil {
				assert.Nil(t, tt.request.App)
			} else {
				assert.NotNil(t, tt.request.App)
				assert.Equal(t, tt.expectedApp.Publisher, tt.request.App.Publisher)
			}
		})
	}
}
func TestWrapperData_setTagId(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *wrapperData
		request     *openrtb2.BidRequest
		expectedTag string
	}{
		{
			name: "TagId is empty, does nothing",
			wrapper: &wrapperData{
				TagId: "",
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{TagID: "existing-tag"},
				},
			},
			expectedTag: "existing-tag",
		},
		{
			name: "No impressions, does nothing",
			wrapper: &wrapperData{
				TagId: "new-tag",
			},
			request:     &openrtb2.BidRequest{Imp: []openrtb2.Imp{}},
			expectedTag: "",
		},
		{
			name: "Sets TagId on first impression",
			wrapper: &wrapperData{
				TagId: "new-tag",
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{TagID: "old-tag"},
					{TagID: "second-tag"},
				},
			},
			expectedTag: "new-tag",
		},
		{
			name: "Sets TagId when first impression TagID is empty",
			wrapper: &wrapperData{
				TagId: "filled-tag",
			},
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{TagID: ""},
				},
			},
			expectedTag: "filled-tag",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wrapper.setTagId(tt.request)
			if len(tt.request.Imp) > 0 {
				assert.Equal(t, tt.expectedTag, tt.request.Imp[0].TagID, "Unexpected TagID for test: %s", tt.name)
			} else {
				assert.Empty(t, tt.expectedTag, "Expected empty TagID for test: %s", tt.name)
			}
		})
	}
}
