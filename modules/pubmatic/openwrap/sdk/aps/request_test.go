package aps

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	adcom1 "github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPS(t *testing.T) {
	a := NewAPS(nil)
	require.NotNil(t, a)
	assert.Nil(t, a.metricsEngine)
}

func TestModifyRequestWithAPSParams(t *testing.T) {
	rctx := models.RequestCtx{}

	signalBR := &openrtb2.BidRequest{
		Imp: []openrtb2.Imp{{
			ID:                "si1",
			Instl:             1,
			DisplayManager:    "dm",
			DisplayManagerVer: "2.0.0",
			Ext:               json.RawMessage(`{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}`),
		}},
		Device: &openrtb2.Device{UA: "Mozilla"},
		App:    &openrtb2.App{Name: "SignalApp"},
	}
	validSig := mustMarshalSignalBidRequest(t, signalBR)
	badSignal := "not-json"

	tests := []struct {
		name             string
		requestBody      []byte
		expectedResponse []byte
		expectedError    bool
		expectNilBody    bool
		metricsSetup     func(*mock_metrics.MockMetricsEngine)
		signalBR         *openrtb2.BidRequest
	}{
		{
			name:             "empty_request_body",
			requestBody:      nil,
			expectedResponse: nil,
		},
		{
			name:             "invalid_json_returns_original_bytes",
			requestBody:      []byte(`{broken`),
			expectedResponse: []byte(`{broken`),
			expectedError:    true,
		},
		{
			name:             "static_data_sets_secure_clears_native_on_imp[0]",
			requestBody:      []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","secure":0,"banner":{"w":300,"h":250},"video":{"mimes":["video/mp4"]},"native":{"request":"n"}}],"app":{"publisher":{"id":"pub-9"}}}`),
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","banner":{"w":300,"h":250},"video":{"mimes":["video/mp4"]},"tagid":"t1","secure":1}],"app":{"publisher":{"id":"pub-9"}}}`),
		},
		{
			name:             "reward_video_sets_rwdd_drops_banner_when_video.ext.videotype_is_rewarded",
			requestBody:      []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","banner":{"w":1,"h":1},"video":{"ext":{"videotype":"rewarded"}}}],"app":{"publisher":{"id":"pub"}}}`),
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","video":{"ext":{"videotype":"rewarded"},"mimes":null},"secure":1,"rwdd":1}],"app":{"publisher":{"id":"pub"}}}`),
		},
		{
			name:        "missing_signal_records_metric",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t"}],"app":{"publisher":{"id":"pub-1"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":42}}}}},"user":{"buyeruid":""}}`),
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("pub-1", "42", models.MissingSignal)
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t","secure":1}],"app":{"publisher":{"id":"pub-1"}},"user":{},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":42}}}}}}`),
		},
		{
			name:        "invalid_json_inside_signal_records_metric",
			requestBody: []byte(fmt.Sprintf(`{"id":"r1","imp":[{"id":"i1","tagid":"t"}],"app":{"publisher":{"id":"p"}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1}}}}},"user":{"buyeruid":%q}}`, badSignal)),
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("p", "1", models.InvalidSignal)
			},
			expectedResponse: []byte(fmt.Sprintf(`{"id":"r1","imp":[{"id":"i1","tagid":"t","secure":1}],"app":{"publisher":{"id":"p"}},"user":{"buyeruid":%q},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":1}}}}}}`, badSignal)),
		},
		{
			name:             "valid_signal_merges_impression_app_and_device_from_signal",
			requestBody:      []byte(fmt.Sprintf(`{"id":"base","imp":[{"id":"i1","tagid":"t1","ext":{}}],"app":{"publisher":{"id":"pubx"}},"device":{"ua":"orig"},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}},"user":{"buyeruid":%q}}`, validSig)),
			expectedResponse: []byte(`{"id":"base","imp":[{"id":"i1","displaymanager":"dm","displaymanagerver":"2.0.0","instl":1,"tagid":"t1","secure":1,"ext":{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}}],"app":{"name":"SignalApp","publisher":{"id":"pubx"}},"device":{"ua":"Mozilla"},"user":{},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}}}`),
		},
		{
			name:        "video_battr_preserved_when_signal_has_video",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","video":{"battr":[1,2],"mimes":["video/mp4"]}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`),
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4", "video/webm"}},
					Instl: 1,
				}},
				Device: &openrtb2.Device{UA: "Mozilla"},
				App:    &openrtb2.App{Name: "SignalApp"},
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","instl":1,"tagid":"t1","video":{"battr":[1,2],"mimes":["video/mp4","video/webm"]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockMetrics := mock_metrics.NewMockMetricsEngine(ctrl)
			if tt.metricsSetup != nil {
				tt.metricsSetup(mockMetrics)
			}

			sig := validSig
			if tt.signalBR != nil {
				sig = mustMarshalSignalBidRequest(t, tt.signalBR)
			}

			// Inject signal into request body
			requestBody := tt.requestBody
			if requestBody != nil && strings.Contains(string(requestBody), "%q") {
				requestBody = []byte(fmt.Sprintf(string(requestBody), sig))
			}

			a := NewAPS(mockMetrics)
			response := a.ModifyRequestWithAPSParams(requestBody, rctx)
			if tt.expectedError {
				assert.Equal(t, tt.expectedResponse, response)
				return
			}

			if tt.expectedResponse == nil {
				assert.Empty(t, response)
			} else {
				assert.JSONEq(t, string(tt.expectedResponse), string(response))
			}
		})
	}
}

func TestModifyBanner(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.Banner
		signal   *openrtb2.Banner
		expected *openrtb2.Banner
	}{
		{
			name:     "nil_request_is_a_no_op",
			request:  nil,
			signal:   &openrtb2.Banner{API: []adcom1.APIFramework{5}},
			expected: nil,
		},
		{
			name:     "nil_signal_is_a_no_op",
			request:  &openrtb2.Banner{W: ptrutil.ToPtr[int64](1)},
			signal:   nil,
			expected: &openrtb2.Banner{W: ptrutil.ToPtr[int64](1)},
		},
		{
			name:    "copies_api_frameworks_from_signal",
			request: &openrtb2.Banner{},
			signal:  &openrtb2.Banner{API: []adcom1.APIFramework{7}},
			expected: &openrtb2.Banner{
				API: []adcom1.APIFramework{7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyBanner(tt.request, tt.signal)
			assert.Equal(t, tt.expected, tt.request)
		})
	}
}

func TestUpdateImpExtension(t *testing.T) {
	tests := []struct {
		name             string
		reqExt           []byte
		sigExt           []byte
		expectedResponse string
	}{
		{
			name:             "nil_signal_returns_request_ext_unchanged",
			reqExt:           []byte(`{"prebid":1}`),
			sigExt:           nil,
			expectedResponse: `{"prebid":1}`,
		},
		{
			name:             "empty_request_ext_receives_skadn_and_owsdk_from_signal",
			reqExt:           nil,
			sigExt:           []byte(`{"skadn":{"version":"2"},"owsdk":{"a":1}}`),
			expectedResponse: `{"skadn":{"version":"2"},"owsdk":{"a":1}}`,
		},
		{
			name:             "merges_skadn_paths_and_owsdk_into_existing_ext",
			reqExt:           []byte(`{"foo":1}`),
			sigExt:           []byte(`{"skadn":{"skoverlay":true,"productpage":7},"owsdk":{"k":2}}`),
			expectedResponse: `{"foo":1,"owsdk":{"k":2},"skadn":{"productpage":7,"skoverlay":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := updateImpExtension(tt.reqExt, tt.sigExt)
			assert.JSONEq(t, tt.expectedResponse, string(out))
		})
	}
}

func TestUpdateRegs(t *testing.T) {
	tests := []struct {
		name     string
		req      *openrtb2.BidRequest
		sig      *openrtb2.Regs
		expected string
	}{
		{
			name:     "copies_coppa_and_reg_ext_paths_from_signal",
			req:      &openrtb2.BidRequest{},
			sig:      &openrtb2.Regs{COPPA: 1, Ext: json.RawMessage(`{"gdpr":1,"gpp":"x"}`)},
			expected: `{"coppa":1,"ext":{"gdpr":1,"gpp":"x"}}`,
		},
		{
			name: "nil_signal_leaves_request_unchanged",
			req: &openrtb2.BidRequest{
				Regs: &openrtb2.Regs{Ext: json.RawMessage(`{"keep":true}`)},
			},
			sig:      nil,
			expected: `{"ext":{"keep":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateRegs(tt.req, tt.sig)
			require.NotNil(t, tt.req.Regs)
			b, err := json.Marshal(tt.req.Regs)
			require.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(b))
		})
	}
}

func TestUpdateApp(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.App
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil_signal_app",
			request:  &openrtb2.BidRequest{App: &openrtb2.App{Name: "test"}},
			signal:   nil,
			expected: &openrtb2.BidRequest{App: &openrtb2.App{Name: "test"}},
		},
		{
			name:     "nil_request_app",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.App{Name: "test"},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{Name: "test"}},
		},
		{
			name:    "copy_app_fields_from_signal",
			request: &openrtb2.BidRequest{App: &openrtb2.App{}},
			signal: &openrtb2.App{
				Domain:   "example.com",
				Cat:      []string{"IAB1"},
				Paid:     ptrutil.ToPtr(int8(1)),
				Keywords: "test,app",
				Name:     "test-app",
			},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain:   "example.com",
				Cat:      []string{"IAB1"},
				Paid:     ptrutil.ToPtr(int8(1)),
				Keywords: "test,app",
				Name:     "test-app",
			}},
		},
		{
			name: "empty_signal_fields_not_copied",
			request: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain:   "example.com",
				Cat:      []string{"IAB1"},
				Keywords: "test,app",
				Name:     "test-app",
			}},
			signal: &openrtb2.App{
				Domain:   "",
				Cat:      []string{},
				Keywords: "",
				Name:     "",
			},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain:   "example.com",
				Cat:      []string{"IAB1"},
				Keywords: "test,app",
				Name:     "test-app",
			}},
		},
		{
			name: "partial_signal_fields_copied",
			request: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain: "example.com",
				Cat:    []string{"IAB1"},
			}},
			signal: &openrtb2.App{
				Keywords: "test,app",
				Name:     "test-app",
			},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain:   "example.com",
				Cat:      []string{"IAB1"},
				Keywords: "test,app",
				Name:     "test-app",
			}},
		},
		{
			name: "ver_copied_from_signal_when_non_empty",
			request: &openrtb2.BidRequest{App: &openrtb2.App{
				Name: "my-app",
			}},
			signal: &openrtb2.App{
				Ver: "3.2.1",
			},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{
				Name: "my-app",
				Ver:  "3.2.1",
			}},
		},
		{
			name: "signal_domain_ignored_when_request_already_has_domain",
			request: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain: "keep.example",
				Bundle: "com.keep",
			}},
			signal: &openrtb2.App{
				Domain: "other.example",
				Name:   "from-signal",
			},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{
				Domain: "keep.example",
				Bundle: "com.keep",
				Name:   "from-signal",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateApp(tt.request, tt.signal)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(tt.request)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestUpdateDevice(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.Device
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil_signal_device",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
		},
		{
			name:     "nil_request_device",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.Device{UA: "test-ua"},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
		},
		{
			name:     "signal_has_device_ip",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
			signal:   &openrtb2.Device{IP: "127.0.0.1"},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IP: "127.0.0.1"}},
		},
		{
			name:     "request_has_device_ip",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IP: "127.0.0.1"}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IP: "127.0.0.1"}},
		},
		{
			name:     "both_request_and_signal_has_device_ip",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IP: "127.0.0.1"}},
			signal:   &openrtb2.Device{IP: "127.0.0.2"},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IP: "127.0.0.2"}},
		},
		{
			name:     "signal_has_device_ipv6",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
			signal:   &openrtb2.Device{IPv6: "2001:db8::1"},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IPv6: "2001:db8::1"}},
		},
		{
			name:     "request_has_device_ipv6",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IPv6: "2001:db8::2"}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IPv6: "2001:db8::2"}},
		},
		{
			name:     "both_request_and_signal_has_device_ipv6",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IPv6: "2001:db8::2"}},
			signal:   &openrtb2.Device{IPv6: "2001:db8::1"},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", IPv6: "2001:db8::1"}},
		},
		{
			name: "geo_lat_lon_present_in_request_keep_coupled_fields_from_request",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{Geo: &openrtb2.Geo{
				Lat:      ptrutil.ToPtr(float64(1.1)),
				Lon:      ptrutil.ToPtr(float64(2.2)),
				Type:     3,
				Accuracy: 10,
				LastFix:  123,
			}}},
			signal: &openrtb2.Device{Geo: &openrtb2.Geo{
				Lat:      ptrutil.ToPtr(float64(9.9)),
				Lon:      ptrutil.ToPtr(float64(8.8)),
				Type:     1,
				Accuracy: 99,
				LastFix:  999,
				Country:  "US",
			}},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{Geo: &openrtb2.Geo{
				Lat:      ptrutil.ToPtr(float64(1.1)),
				Lon:      ptrutil.ToPtr(float64(2.2)),
				Type:     3,
				Accuracy: 10,
				LastFix:  123,
				Country:  "US",
			}}},
		},
		{
			name: "geo_lat_lon_missing_in_request_copy_coupled_fields_from_signal",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{Geo: &openrtb2.Geo{
				Lat: nil,
				Lon: ptrutil.ToPtr(float64(2.2)),
			}}},
			signal: &openrtb2.Device{Geo: &openrtb2.Geo{
				Lat:      ptrutil.ToPtr(float64(9.9)),
				Lon:      ptrutil.ToPtr(float64(8.8)),
				Type:     1,
				Accuracy: 99,
				LastFix:  999,
			}},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{Geo: &openrtb2.Geo{
				Lat:      ptrutil.ToPtr(float64(9.9)),
				Lon:      ptrutil.ToPtr(float64(8.8)),
				Type:     1,
				Accuracy: 99,
				LastFix:  999,
			}}},
		},
		{
			name:    "copy_all_device_fields",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{}},
			signal: &openrtb2.Device{
				UA:             "test-ua",
				Geo:            &openrtb2.Geo{Country: "US"},
				Carrier:        "test-carrier",
				Language:       "en",
				HWV:            "1.0",
				MCCMNC:         "123",
				Make:           "test-make",
				Model:          "test-model",
				OS:             "test-os",
				OSV:            "1.0",
				JS:             ptrutil.ToPtr(int8(1)),
				DeviceType:     adcom1.DeviceType(1),
				Lmt:            ptrutil.ToPtr(int8(1)),
				ConnectionType: ptrutil.ToPtr(adcom1.ConnectionType(1)),
				W:              320,
				H:              480,
				PxRatio:        2.0,
				IFA:            "test-ifa",
				Ext:            json.RawMessage(`{"atts":1}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:             "test-ua",
				Geo:            &openrtb2.Geo{Country: "US"},
				Carrier:        "test-carrier",
				Language:       "en",
				HWV:            "1.0",
				MCCMNC:         "123",
				Make:           "test-make",
				Model:          "test-model",
				JS:             ptrutil.ToPtr(int8(1)),
				DeviceType:     adcom1.DeviceType(1),
				Lmt:            ptrutil.ToPtr(int8(1)),
				ConnectionType: ptrutil.ToPtr(adcom1.ConnectionType(1)),
				OS:             "test-os",
				OSV:            "1.0",
				W:              320,
				H:              480,
				PxRatio:        2.0,
				IFA:            "test-ifa",
				Ext:            json.RawMessage(`{"atts":1}`),
			}},
		},
		{
			name: "empty_signal_fields_not_copied",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:       "test-ua",
				Language: "en",
				Make:     "test-make",
				Model:    "test-model",
			}},
			signal: &openrtb2.Device{
				UA:       "",
				Language: "",
				Make:     "",
				Model:    "",
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:       "test-ua",
				Language: "en",
				Make:     "test-make",
				Model:    "test-model",
			}},
		},
		{
			name: "partial_signal_fields_copied",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:       "test-ua",
				Language: "en",
			}},
			signal: &openrtb2.Device{
				Make:  "test-make",
				Model: "test-model",
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:       "test-ua",
				Language: "en",
				Make:     "test-make",
				Model:    "test-model",
			}},
		},
		{
			name: "signal_has_ifv",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA: "test-ua",
			}},
			signal: &openrtb2.Device{
				Ext: json.RawMessage(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			}},
		},
		{
			name: "request_has_ifv_signal_does_not",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
			signal: &openrtb2.Device{
				Make: "test-make",
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:   "test-ua",
				Make: "test-make",
				Ext:  json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
		},
		{
			name: "both_request_and_signal_have_ifv_signal_wins",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
			signal: &openrtb2.Device{
				Ext: json.RawMessage(`{"ifv":"SIGNAL-IFV-VALUE"}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"ifv":"SIGNAL-IFV-VALUE"}`),
			}},
		},
		{
			name: "signal_has_empty_ifv_overwrites_request_ifv",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
			signal: &openrtb2.Device{
				Ext: json.RawMessage(`{"ifv":""}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"ifv":""}`),
			}},
		},
		{
			name: "signal_has_both_atts_and_ifv",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA: "test-ua",
			}},
			signal: &openrtb2.Device{
				Ext: json.RawMessage(`{"atts":3,"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: json.RawMessage(`{"atts":3,"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateDevice(tt.request, tt.signal)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(tt.request)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestUpdateUser(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.User
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil_signal_user",
			request:  &openrtb2.BidRequest{User: &openrtb2.User{Yob: 2000}},
			signal:   nil,
			expected: &openrtb2.BidRequest{User: &openrtb2.User{Yob: 2000}},
		},
		{
			name:     "nil_request_user",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.User{Yob: 2000},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{Yob: 2000}},
		},
		{
			name:    "copy_user_fields_and_ext_paths_from_signal",
			request: &openrtb2.BidRequest{User: &openrtb2.User{}},
			signal: &openrtb2.User{
				Data:     []openrtb2.Data{{ID: "1"}},
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      json.RawMessage(`{"sessionduration":300,"impdepth":1,"consent":"test","eids":[{"source":"test"}]}`),
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Data:     []openrtb2.Data{{ID: "1"}},
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      json.RawMessage(`{"sessionduration":300,"impdepth":1,"consent":"test","eids":[{"source":"test"}]}`),
			}},
		},
		{
			name: "empty_signal_fields_not_copied",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
			}},
			signal: &openrtb2.User{
				Yob:      0,
				Gender:   "",
				Keywords: "",
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
			}},
		},
		{
			name: "partial_signal_fields_copied",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Yob:    2000,
				Gender: "M",
			}},
			signal: &openrtb2.User{
				Keywords: "test,user",
				Ext:      json.RawMessage(`{"sessionduration":300}`),
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      json.RawMessage(`{"sessionduration":300}`),
			}},
		},
		{
			name: "buyeruid_on_request_preserved",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				BuyerUID: "keep-token",
				Yob:      1990,
			}},
			signal: &openrtb2.User{
				Yob:      2000,
				Gender:   "F",
				Keywords: "kw",
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				BuyerUID: "keep-token",
				Yob:      2000,
				Gender:   "F",
				Keywords: "kw",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateUser(tt.request, tt.signal)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(tt.request)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestUpdateSource(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.Source
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil_signal_source",
			request:  &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"existingfield":1}`)}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"existingfield":1}`)}},
		},
		{
			name:     "nil_request_source",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.Source{Ext: json.RawMessage(`{"omidpn":"test","omidpv":"1.0"}`)},
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"omidpn":"test","omidpv":"1.0"}`)}},
		},
		{
			name:     "merge_omidpn_and_omidpv_into_existing_source.ext",
			request:  &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"existingfield":1}`)}},
			signal:   &openrtb2.Source{Ext: json.RawMessage(`{"omidpn":"test","omidpv":"1.0"}`)},
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"existingfield":1,"omidpn":"test","omidpv":"1.0"}`)}},
		},
		{
			name:     "partial_omid_fields_copied",
			request:  &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"existingfield":1}`)}},
			signal:   &openrtb2.Source{Ext: json.RawMessage(`{"omidpn":"test"}`)},
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: json.RawMessage(`{"existingfield":1,"omidpn":"test"}`)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateSource(tt.request, tt.signal)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(tt.request)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestUpdateImpression(t *testing.T) {
	reqImpExt := json.RawMessage(`{"prebid":1}`)
	sigImpExt := json.RawMessage(`{"skadn":{"versions":["3.0"]},"owsdk":{"a":1}}`)
	mergedImpExt := json.RawMessage(updateImpExtension(reqImpExt, sigImpExt))

	tests := []struct {
		name       string
		request    *openrtb2.BidRequest
		signalImps []openrtb2.Imp
		expected   *openrtb2.BidRequest
	}{
		{
			name:       "empty_request_imp_array",
			request:    &openrtb2.BidRequest{},
			signalImps: []openrtb2.Imp{{ID: "1"}},
			expected:   &openrtb2.BidRequest{},
		},
		{
			name: "empty_signal_imp_array",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
			signalImps: []openrtb2.Imp{},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
		},
		{
			name: "copy_display_manager_version_and_clickbrowser",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
			signalImps: []openrtb2.Imp{
				{
					ID:                "1",
					DisplayManager:    "unity",
					DisplayManagerVer: "1.0",
					ClickBrowser:      ptrutil.ToPtr(int8(1)),
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:                "1",
						DisplayManager:    "unity",
						DisplayManagerVer: "1.0",
						ClickBrowser:      ptrutil.ToPtr(int8(1)),
					},
				},
			},
		},
		{
			name: "copies_instl_from_signal_aps_always_assigns_imp[0].instl",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Instl: 1},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 1}},
			},
		},
		{
			name: "copies_exp_from_signal_when_positive",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 0}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Exp: 300},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 300}},
			},
		},
		{
			name: "does_not_overwrite_request_exp_when_signal_exp_is_zero",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 120}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Exp: 0},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 120}},
			},
		},
		{
			name: "video_object_replaced_from_signal_when_signal_has_video",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
						Video: &openrtb2.Video{
							Ext: json.RawMessage(`{"reward":0}`),
						},
					},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					ID: "1",
					Video: &openrtb2.Video{
						MIMEs: []string{"video/mp4"},
					},
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
						Video: &openrtb2.Video{
							MIMEs: []string{"video/mp4"},
						},
					},
				},
			},
		},
		{
			name: "banner_api_merged_from_signal_via_modifyBanner",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:     "i1",
					TagID:  "t",
					Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](300)},
				}},
			},
			signalImps: []openrtb2.Imp{{
				ID: "1",
				Banner: &openrtb2.Banner{
					API: []adcom1.APIFramework{7},
				},
			}},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "i1",
					TagID: "t",
					Banner: &openrtb2.Banner{
						W:   ptrutil.ToPtr[int64](300),
						API: []adcom1.APIFramework{7},
					},
				}},
			},
		},
		{
			name: "imp_ext_merged_via_updateImpExtension",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "1",
						Ext: reqImpExt,
					},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					ID:  "1",
					Ext: sigImpExt,
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:  "1",
						Ext: mergedImpExt,
					},
				},
			},
		},
		{
			name: "only_first_impression_is_merged_second_impression_unchanged",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "a"},
					{ID: "b", TagID: "keep"},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					ID:                "s1",
					DisplayManager:    "dm",
					DisplayManagerVer: "2",
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:                "a",
						DisplayManager:    "dm",
						DisplayManagerVer: "2",
					},
					{ID: "b", TagID: "keep"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateImpression(tt.request, tt.signalImps)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(tt.request)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func mustMarshalSignalBidRequest(t *testing.T, br *openrtb2.BidRequest) string {
	t.Helper()
	b, err := json.Marshal(br)
	require.NoError(t, err)
	return string(b)
}
