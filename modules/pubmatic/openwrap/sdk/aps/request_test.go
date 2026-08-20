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
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/sdk/sdkutils"
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
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":3600,"video":{"ext":{"videotype":"rewarded"},"mimes":null},"secure":1,"rwdd":1}],"app":{"publisher":{"id":"pub"}}}`),
		},
		{
			name:             "s2s_request_exp_preserved_when_signal_has_no_exp",
			requestBody:      []byte(fmt.Sprintf(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":120,"banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`, validSig)),
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","displaymanager":"dm","displaymanagerver":"2.0.0","tagid":"t1","exp":120,"banner":{"w":300,"h":250},"video":{"w":300,"h":250,"mimes":null,"companionad":[{}]},"secure":1,"ext":{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
		},
		{
			name: "s2s_banner_without_exp_sets_600_ignores_signal_exp",
			requestBody: []byte(fmt.Sprintf(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`, mustMarshalSignalBidRequest(t, &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 3600, Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](300)}}},
			}))),
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":600,"banner":{"w":300,"h":250},"video":{"w":300,"h":250,"mimes":null,"companionad":[{}]},"secure":1}],"app":{"publisher":{"id":"pub"}},"user":{}}`),
		},
		{
			name: "s2s_banner_without_exp_sets_600_before_signal_adds_video",
			requestBody: []byte(fmt.Sprintf(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","banner":{"w":300,"h":250}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`, mustMarshalSignalBidRequest(t, &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}}}},
			}))),
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":600,"banner":{"w":300,"h":250},"video":{"mimes":["video/mp4"],"w":300,"h":250,"companionad":[{}]},"secure":1}],"app":{"publisher":{"id":"pub"}},"user":{}}`),
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
			expectedResponse: []byte(`{"id":"base","imp":[{"id":"i1","displaymanager":"dm","displaymanagerver":"2.0.0","tagid":"t1","secure":1,"ext":{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}}],"app":{"name":"SignalApp","publisher":{"id":"pubx"}},"device":{"ua":"Mozilla"},"user":{},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}}}`),
		},
		{
			name:             "interstitial_request_sets_instl_from_ad_format",
			requestBody:      []byte(fmt.Sprintf(`{"id":"base","imp":[{"id":"i1","tagid":"t1","instl":1,"ext":{}}],"app":{"publisher":{"id":"pubx"}},"device":{"ua":"orig"},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}},"user":{"buyeruid":%q}}`, validSig)),
			expectedResponse: []byte(`{"id":"base","imp":[{"id":"i1","displaymanager":"dm","displaymanagerver":"2.0.0","instl":1,"tagid":"t1","secure":1,"ext":{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}}],"app":{"name":"SignalApp","publisher":{"id":"pubx"}},"device":{"ua":"Mozilla"},"user":{},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}}}`),
		},
		{
			name:             "rewarded_request_sets_instl_from_ad_format",
			requestBody:      []byte(fmt.Sprintf(`{"id":"base","imp":[{"id":"i1","tagid":"t1","rwdd":1,"video":{"ext":{"videotype":"rewarded"}},"ext":{}}],"app":{"publisher":{"id":"pubx"}},"device":{"ua":"orig"},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}},"user":{"buyeruid":%q}}`, validSig)),
			expectedResponse: []byte(`{"id":"base","imp":[{"id":"i1","displaymanager":"dm","displaymanagerver":"2.0.0","instl":1,"tagid":"t1","exp":3600,"secure":1,"rwdd":1,"video":{"ext":{"videotype":"rewarded"},"mimes":null},"ext":{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}}],"app":{"name":"SignalApp","publisher":{"id":"pubx"}},"device":{"ua":"Mozilla"},"user":{},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}}}`),
		},
		{
			name:             "banner_request_clears_instl_from_ad_format",
			requestBody:      []byte(fmt.Sprintf(`{"id":"base","imp":[{"id":"i1","tagid":"t1","banner":{"w":728,"h":90},"ext":{}}],"app":{"publisher":{"id":"pubx"}},"device":{"ua":"orig"},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}},"user":{"buyeruid":%q}}`, validSig)),
			expectedResponse: []byte(`{"id":"base","imp":[{"id":"i1","displaymanager":"dm","displaymanagerver":"2.0.0","tagid":"t1","exp":600,"secure":1,"banner":{"w":728,"h":90},"ext":{"skadn":{"versions":["v1"]},"owsdk":{"x":1}}}],"app":{"name":"SignalApp","publisher":{"id":"pubx"}},"device":{"ua":"Mozilla"},"user":{},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":100}}}}}}`),
		},
		{
			name:        "mrec_banner_only_applies_banner_fields_to_final_video",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","banner":{"w":300,"h":250,"format":[{"w":300,"h":250}]}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`),
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4", "video/webm"}},
				}},
				Device: &openrtb2.Device{UA: "Mozilla"},
				App:    &openrtb2.App{Name: "SignalApp"},
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":600,"banner":{"w":300,"h":250,"format":[{"w":300,"h":250}]},"video":{"mimes":["video/mp4","video/webm"],"w":300,"h":250,"companionad":[{"format":[{"w":300,"h":250}]}]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
		},
		{
			name:        "interstitial_banner_only_applies_banner_fields_to_final_video",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","instl":1,"banner":{"w":320,"h":480,"format":[{"w":320,"h":480}]}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`),
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
				Device: &openrtb2.Device{UA: "Mozilla"},
				App:    &openrtb2.App{Name: "SignalApp"},
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","instl":1,"tagid":"t1","exp":600,"banner":{"w":320,"h":480,"format":[{"w":320,"h":480}]},"video":{"mimes":["video/mp4"],"w":320,"h":480,"companionad":[{"format":[{"w":320,"h":480}]}]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
		},
		{
			name:        "rewarded_banner_only_creates_video_from_aps_banner",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","rwdd":1,"banner":{"w":320,"h":480,"format":[{"w":320,"h":480}]}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`),
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4", "video/webm"}},
				}},
				Device: &openrtb2.Device{UA: "Mozilla"},
				App:    &openrtb2.App{Name: "SignalApp"},
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","instl":1,"tagid":"t1","exp":600,"rwdd":1,"banner":{"w":320,"h":480,"format":[{"w":320,"h":480}]},"video":{"mimes":["video/mp4","video/webm"],"w":320,"h":480,"companionad":[{"format":[{"w":320,"h":480}]}]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
		},
		{
			name:        "mrec_video_only_applies_video_fields_to_final_banner",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","video":{"w":300,"h":250,"companionad":[{"format":[{"w":300,"h":250}]}]}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`),
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID: "si1",
					Banner: &openrtb2.Banner{
						MIMEs: []string{"image/jpeg", "image/png"},
						API:   []adcom1.APIFramework{5, 6},
					},
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4", "video/webm"}},
				}},
				Device: &openrtb2.Device{UA: "Mozilla"},
				App:    &openrtb2.App{Name: "SignalApp"},
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":3600,"banner":{"w":300,"h":250,"format":[{"w":300,"h":250}],"mimes":["image/jpeg","image/png"],"api":[5,6]},"video":{"mimes":["video/mp4","video/webm"],"w":300,"h":250,"companionad":[{"format":[{"w":300,"h":250}]}]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
		},
		{
			name:        "interstitial_video_only_applies_video_fields_to_final_banner",
			requestBody: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","instl":1,"video":{"w":320,"h":480,"companionad":[{"format":[{"w":320,"h":480}]}]}}],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`),
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
				Device: &openrtb2.Device{UA: "Mozilla"},
				App:    &openrtb2.App{Name: "SignalApp"},
			},
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","instl":1,"tagid":"t1","exp":3600,"banner":{"w":320,"h":480,"format":[{"w":320,"h":480}]},"video":{"mimes":["video/mp4"],"w":320,"h":480,"companionad":[{"format":[{"w":320,"h":480}]}]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
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
			expectedResponse: []byte(`{"id":"r1","imp":[{"id":"i1","tagid":"t1","exp":3600,"video":{"battr":[1,2],"mimes":["video/mp4","video/webm"]},"secure":1}],"app":{"name":"SignalApp","publisher":{"id":"pub"}},"device":{"ua":"Mozilla"},"user":{}}`),
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
			response := a.ModifyRequestWithAPSParams(requestBody, &rctx)
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

func TestMergeBannerFromSignal(t *testing.T) {
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
		{
			name:    "copies_mimes_from_signal",
			request: &openrtb2.Banner{},
			signal:  &openrtb2.Banner{MIMEs: []string{"image/jpeg", "image/png"}},
			expected: &openrtb2.Banner{
				MIMEs: []string{"image/jpeg", "image/png"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sdkutils.MergeBanner(tt.request, tt.signal)
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
			expectedResponse: `{"foo":1,"owsdk":{"k":2},"skadn":{"productpage":7}}`,
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
		{
			name:     "signal_has_ppi",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
			signal:   &openrtb2.Device{PPI: 440},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua", PPI: 440}},
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
				Ext:      json.RawMessage(`{"sessionduration":300,"consent":"test","eids":[{"source":"test"}]}`),
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Data:     []openrtb2.Data{{ID: "1"}},
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      json.RawMessage(`{"sessionduration":300,"consent":"test","eids":[{"source":"test"}]}`),
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

func TestUpdateImpressionWithSignal(t *testing.T) {
	reqImpExt := json.RawMessage(`{"prebid":1}`)
	sigImpExt := json.RawMessage(`{"skadn":{"versions":["3.0"]},"owsdk":{"a":1}}`)
	mergedImpExt := json.RawMessage(updateImpExtension(reqImpExt, sigImpExt))

	tests := []struct {
		name       string
		adFormat   string
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
			name:     "sets_instl_from_ad_format_interstitial",
			adFormat: apsAdFormatInterstitial,
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Instl: 0},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 1}},
			},
		},
		{
			name:     "sets_instl_from_ad_format_rewarded",
			adFormat: apsAdFormatRewarded,
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Instl: 0},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 1}},
			},
		},
		{
			name:     "clears_instl_for_banner_ad_format",
			adFormat: apsAdFormatBanner,
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 1}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Instl: 1},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 0}},
			},
		},
		{
			name:     "clears_instl_for_mrec_ad_format",
			adFormat: apsAdFormatMrec,
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 1}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Instl: 1},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 0}},
			},
		},
		{
			name: "leaves_instl_unchanged_when_ad_format_empty",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 0}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Instl: 1},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Instl: 0}},
			},
		},
		{
			name: "ignores_signal_exp_preserves_s2s_request_exp",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 120}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Exp: 300},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 120}},
			},
		},
		{
			name: "ignores_signal_exp_when_s2s_request_exp_unset",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 0}},
			},
			signalImps: []openrtb2.Imp{
				{ID: "1", Exp: 3600},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1", Exp: 0}},
			},
		},
		{
			name: "preserves_s2s_request_exp_when_signal_exp_is_zero",
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
			updateImpressionWithSignal(tt.request, tt.signalImps, tt.adFormat, nil)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(tt.request)
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

// TestModifyRequestWithAPSParams_ImpExpBeforeSignalMerge_MrecInterstitial confirms imp.exp is
// derived from the original S2S ad format before OWSDK signal merge. For MREC and interstitial,
// signal merge can end up with both banner and video (UOE-13773 adds the missing format via
// applyBannerFromApsVideo / applyVideoFromApsBanner); exp must stay based on the original S2S format.
func TestModifyRequestWithAPSParams_ImpExpBeforeSignalMerge_MrecInterstitial(t *testing.T) {
	tests := []struct {
		name       string
		s2sImp     string
		signalBR   *openrtb2.BidRequest
		wantExp    int64
		wantBanner bool
		wantVideo  bool
	}{
		{
			name:   "interstitial_s2s_banner_signal_adds_video_imp_exp_stays_600",
			s2sImp: `{"id":"i1","tagid":"t1","instl":1,"banner":{"w":320,"h":480,"format":[{"w":320,"h":480}]}}`,
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
			},
			wantExp:    600,
			wantBanner: true,
			wantVideo:  true,
		},
		{
			name:   "mrec_s2s_banner_signal_adds_video_imp_exp_stays_600",
			s2sImp: `{"id":"i1","tagid":"t1","banner":{"w":300,"h":250,"format":[{"w":300,"h":250}]}}`,
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
			},
			wantExp:    600,
			wantBanner: true,
			wantVideo:  true,
		},
		{
			name:   "interstitial_s2s_video_imp_exp_stays_3600_when_signal_adds_video",
			s2sImp: `{"id":"i1","tagid":"t1","instl":1,"video":{"w":320,"h":480,"companionad":[{"format":[{"w":320,"h":480}]}]}}`,
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:    "si1",
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
			},
			wantExp:    3600,
			wantBanner: true, // banner from APS video fields is added in UOE-13773 (createBannerFromApsVideoIfMissing)
			wantVideo:  true,
		},
		{
			name:   "mrec_s2s_video_imp_exp_stays_3600_when_signal_adds_video",
			s2sImp: `{"id":"i1","tagid":"t1","video":{"w":300,"h":250,"companionad":[{"format":[{"w":300,"h":250}]}]}}`,
			signalBR: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID: "si1",
					Banner: &openrtb2.Banner{
						MIMEs: []string{"image/jpeg", "image/png"},
						API:   []adcom1.APIFramework{5, 6},
					},
					Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
			},
			wantExp:    3600,
			wantBanner: true, // banner from APS video fields is added in UOE-13773 (createBannerFromApsVideoIfMissing)
			wantVideo:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sig := mustMarshalSignalBidRequest(t, tt.signalBR)
			requestBody := []byte(fmt.Sprintf(
				`{"id":"r1","imp":[%s],"app":{"publisher":{"id":"pub"}},"user":{"buyeruid":%q}}`,
				tt.s2sImp, sig,
			))

			modified := NewAPS(nil).ModifyRequestWithAPSParams(requestBody, &models.RequestCtx{})

			var br openrtb2.BidRequest
			require.NoError(t, json.Unmarshal(modified, &br))
			require.Len(t, br.Imp, 1)

			imp := br.Imp[0]
			assert.Equal(t, tt.wantExp, imp.Exp, "imp.exp must reflect original S2S format, not post-signal banner+video")
			assert.Equal(t, tt.wantBanner, imp.Banner != nil)
			assert.Equal(t, tt.wantVideo, imp.Video != nil)
		})
	}
}

func mustMarshalSignalBidRequest(t *testing.T, br *openrtb2.BidRequest) string {
	t.Helper()
	b, err := json.Marshal(br)
	require.NoError(t, err)
	return string(b)
}

func TestCreateBannerFromApsVideoIfMissing(t *testing.T) {
	pos5 := ptrutil.ToPtr(adcom1.PlacementPosition(5))

	tests := []struct {
		name       string
		adFormat   string
		imp        openrtb2.Imp
		apsVideo   *apsVideoFields
		wantBanner *openrtb2.Banner
	}{
		{
			name:     "mrec_creates_banner_from_video_aps_fields",
			adFormat: apsAdFormatMrec,
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{
					W:   ptrutil.ToPtr[int64](300),
					H:   ptrutil.ToPtr[int64](250),
					Pos: pos5,
					CompanionAd: []openrtb2.Banner{{
						Format: []openrtb2.Format{{W: 300, H: 250}},
					}},
				},
			},
			apsVideo: &apsVideoFields{
				W:   ptrutil.ToPtr[int64](300),
				H:   ptrutil.ToPtr[int64](250),
				Pos: pos5,
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				}},
			},
			wantBanner: &openrtb2.Banner{
				W:      ptrutil.ToPtr[int64](300),
				H:      ptrutil.ToPtr[int64](250),
				Pos:    pos5,
				Format: []openrtb2.Format{{W: 300, H: 250}},
			},
		},
		{
			name:     "mrec_banner_only_skips_banner_creation",
			adFormat: apsAdFormatMrec,
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](300),
					H:      ptrutil.ToPtr[int64](250),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
			},
			wantBanner: &openrtb2.Banner{
				W:      ptrutil.ToPtr[int64](300),
				H:      ptrutil.ToPtr[int64](250),
				Pos:    pos5,
				Format: []openrtb2.Format{{W: 300, H: 250}},
			},
		},
		{
			name:     "interstitial_creates_banner_from_video",
			adFormat: apsAdFormatInterstitial,
			imp: openrtb2.Imp{
				Instl: 1,
				Video: &openrtb2.Video{
					W: ptrutil.ToPtr[int64](320),
					H: ptrutil.ToPtr[int64](480),
				},
			},
			apsVideo: &apsVideoFields{
				W: ptrutil.ToPtr[int64](320),
				H: ptrutil.ToPtr[int64](480),
			},
			wantBanner: &openrtb2.Banner{
				W: ptrutil.ToPtr[int64](320),
				H: ptrutil.ToPtr[int64](480),
			},
		},
		{
			name:     "banner_format_skips_object_creation",
			adFormat: apsAdFormatBanner,
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](728), H: ptrutil.ToPtr[int64](90)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := tt.imp
			createBannerFromApsVideoIfMissing(&imp, tt.adFormat, tt.apsVideo)

			if tt.wantBanner != nil {
				require.NotNil(t, imp.Banner)
				assertExpectedBanner(t, tt.wantBanner, imp.Banner)
				if len(tt.wantBanner.Format) > 0 {
					assert.Equal(t, tt.wantBanner.Format, imp.Banner.Format)
				}
				if tt.wantBanner.W != nil {
					require.NotNil(t, imp.Banner.W)
					assert.Equal(t, *tt.wantBanner.W, *imp.Banner.W)
				}
				if tt.wantBanner.H != nil {
					require.NotNil(t, imp.Banner.H)
					assert.Equal(t, *tt.wantBanner.H, *imp.Banner.H)
				}
			} else if tt.adFormat == apsAdFormatBanner {
				assert.Nil(t, imp.Video)
			}
		})
	}
}

func TestUpdateImpressionWithSignalAndApsMedia(t *testing.T) {
	pos5 := ptrutil.ToPtr(adcom1.PlacementPosition(5))

	tests := []struct {
		name       string
		adFormat   string
		imp        openrtb2.Imp
		apsMedia   apsImpMediaFields
		signalImps []openrtb2.Imp
		expected   openrtb2.Imp
	}{
		{
			name:     "mrec_banner_only_creates_video_from_signal_with_aps_banner_sizing",
			adFormat: apsAdFormatMrec,
			imp: openrtb2.Imp{
				ID: "i1",
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](300),
					H:      ptrutil.ToPtr[int64](250),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
			},
			apsMedia: apsImpMediaFields{
				videoMissing: true,
				banner: &apsBannerFields{
					W:      ptrutil.ToPtr[int64](300),
					H:      ptrutil.ToPtr[int64](250),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
			},
			signalImps: []openrtb2.Imp{{
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4", "video/webm"},
				},
			}},
			expected: openrtb2.Imp{
				ID:    "i1",
				Instl: 0,
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](300),
					H:      ptrutil.ToPtr[int64](250),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4", "video/webm"},
					W:     ptrutil.ToPtr[int64](300),
					H:     ptrutil.ToPtr[int64](250),
					CompanionAd: []openrtb2.Banner{{
						Format: []openrtb2.Format{{W: 300, H: 250}},
					}},
				},
			},
		},
		{
			name:     "mrec_video_only_creates_banner_from_aps_video_before_signal_merge",
			adFormat: apsAdFormatMrec,
			imp: openrtb2.Imp{
				ID: "i1",
				Video: &openrtb2.Video{
					W: ptrutil.ToPtr[int64](300),
					H: ptrutil.ToPtr[int64](250),
				},
			},
			apsMedia: apsImpMediaFields{
				video: &apsVideoFields{
					W: ptrutil.ToPtr[int64](300),
					H: ptrutil.ToPtr[int64](250),
				},
			},
			signalImps: []openrtb2.Imp{{
				Banner: &openrtb2.Banner{MIMEs: []string{"image/jpeg"}, API: []adcom1.APIFramework{5}},
				Video:  &openrtb2.Video{MIMEs: []string{"video/mp4"}},
			}},
			expected: openrtb2.Imp{
				ID:    "i1",
				Instl: 0,
				Banner: &openrtb2.Banner{
					W:     ptrutil.ToPtr[int64](300),
					H:     ptrutil.ToPtr[int64](250),
					MIMEs: []string{"image/jpeg"},
					API:   []adcom1.APIFramework{5},
				},
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4"},
					W:     ptrutil.ToPtr[int64](300),
					H:     ptrutil.ToPtr[int64](250),
				},
			},
		},
		{
			name:     "rewarded_banner_only_creates_video_from_aps_banner",
			adFormat: apsAdFormatRewarded,
			imp: openrtb2.Imp{
				ID:   "i1",
				Rwdd: 1,
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](320),
					H:      ptrutil.ToPtr[int64](480),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 320, H: 480}},
				},
			},
			apsMedia: apsImpMediaFields{
				videoMissing: true,
				banner: &apsBannerFields{
					W:      ptrutil.ToPtr[int64](320),
					H:      ptrutil.ToPtr[int64](480),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 320, H: 480}},
				},
			},
			signalImps: []openrtb2.Imp{{
				Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}},
			}},
			expected: openrtb2.Imp{
				ID:    "i1",
				Instl: 1,
				Rwdd:  1,
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](320),
					H:      ptrutil.ToPtr[int64](480),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 320, H: 480}},
				},
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4"},
					W:     ptrutil.ToPtr[int64](320),
					H:     ptrutil.ToPtr[int64](480),
					Pos:   pos5,
					CompanionAd: []openrtb2.Banner{{
						Format: []openrtb2.Format{{W: 320, H: 480}},
						Pos:    pos5,
					}},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{tt.imp}}
			updateImpressionWithSignalAndApsMedia(request, tt.signalImps, tt.adFormat, tt.apsMedia)

			expectedJSON, err := json.Marshal(tt.expected)
			require.NoError(t, err)

			actualJSON, err := json.Marshal(request.Imp[0])
			require.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestApplyApsBannerFieldsToVideo(t *testing.T) {
	pos5 := ptrutil.ToPtr(adcom1.PlacementPosition(5))

	tests := []struct {
		name      string
		adFormat  string
		imp       openrtb2.Imp
		apsBanner *apsBannerFields
		expected  *openrtb2.Video
	}{
		{
			name:     "mrec_applies_aps_banner_fields_after_signal_video",
			adFormat: apsAdFormatMrec,
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](300),
					H:      ptrutil.ToPtr[int64](250),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4"},
					W:     ptrutil.ToPtr[int64](640),
					H:     ptrutil.ToPtr[int64](360),
				},
			},
			apsBanner: &apsBannerFields{
				W:      ptrutil.ToPtr[int64](300),
				H:      ptrutil.ToPtr[int64](250),
				Pos:    pos5,
				Format: []openrtb2.Format{{W: 300, H: 250}},
			},
			expected: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				W:     ptrutil.ToPtr[int64](300),
				H:     ptrutil.ToPtr[int64](250),
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				}},
			},
		},
		{
			name:     "interstitial_applies_aps_banner_fields_after_signal_video",
			adFormat: apsAdFormatInterstitial,
			imp: openrtb2.Imp{
				Instl: 1,
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](320),
					H:      ptrutil.ToPtr[int64](480),
					Format: []openrtb2.Format{{W: 320, H: 480}},
				},
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4"},
					W:     ptrutil.ToPtr[int64](640),
					H:     ptrutil.ToPtr[int64](360),
				},
			},
			apsBanner: &apsBannerFields{
				W:      ptrutil.ToPtr[int64](320),
				H:      ptrutil.ToPtr[int64](480),
				Format: []openrtb2.Format{{W: 320, H: 480}},
			},
			expected: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				W:     ptrutil.ToPtr[int64](320),
				H:     ptrutil.ToPtr[int64](480),
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 320, H: 480}},
				}},
			},
		},
		{
			name:     "rewarded_creates_video_from_aps_banner_when_video_missing",
			adFormat: apsAdFormatRewarded,
			imp: openrtb2.Imp{
				Rwdd: 1,
				Banner: &openrtb2.Banner{
					W:      ptrutil.ToPtr[int64](320),
					H:      ptrutil.ToPtr[int64](480),
					Pos:    pos5,
					Format: []openrtb2.Format{{W: 320, H: 480}},
				},
			},
			apsBanner: &apsBannerFields{
				W:      ptrutil.ToPtr[int64](320),
				H:      ptrutil.ToPtr[int64](480),
				Pos:    pos5,
				Format: []openrtb2.Format{{W: 320, H: 480}},
			},
			expected: &openrtb2.Video{
				W:   ptrutil.ToPtr[int64](320),
				H:   ptrutil.ToPtr[int64](480),
				Pos: pos5,
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 320, H: 480}},
					Pos:    pos5,
				}},
			},
		},
		{
			name:     "banner_format_skips_video_banner_overlay",
			adFormat: apsAdFormatBanner,
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](728), H: ptrutil.ToPtr[int64](90)},
				Video:  &openrtb2.Video{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](360)},
			},
			apsBanner: &apsBannerFields{
				W: ptrutil.ToPtr[int64](728),
				H: ptrutil.ToPtr[int64](90),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imp := tt.imp
			applyApsBannerFieldsToVideo(&imp, tt.adFormat, tt.apsBanner)

			if tt.expected == nil {
				assert.Equal(t, tt.imp.Video.W, ptrutil.ToPtr[int64](640))
				return
			}

			require.NotNil(t, imp.Video)
			assert.Equal(t, tt.expected.MIMEs, imp.Video.MIMEs)
			assertExpectedVideo(t, tt.expected, imp.Video)
		})
	}
}

func TestRestoreApsVideoFields(t *testing.T) {
	pos5 := ptrutil.ToPtr(adcom1.PlacementPosition(5))

	tests := []struct {
		name     string
		adFormat string
		video    *openrtb2.Video
		apsVideo *apsVideoFields
		expected *openrtb2.Video
	}{
		{
			name:     "mrec_skips_pos_on_video_and_companion",
			adFormat: apsAdFormatMrec,
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				W:     ptrutil.ToPtr[int64](640),
				H:     ptrutil.ToPtr[int64](360),
			},
			apsVideo: &apsVideoFields{
				W:   ptrutil.ToPtr[int64](300),
				H:   ptrutil.ToPtr[int64](250),
				Pos: pos5,
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 300, H: 250}},
					Pos:    pos5,
				}},
			},
			expected: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				W:     ptrutil.ToPtr[int64](300),
				H:     ptrutil.ToPtr[int64](250),
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				}},
			},
		},
		{
			name:     "interstitial_restores_pos_on_video_and_companion",
			adFormat: apsAdFormatInterstitial,
			video: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				W:     ptrutil.ToPtr[int64](640),
				H:     ptrutil.ToPtr[int64](360),
			},
			apsVideo: &apsVideoFields{
				W:   ptrutil.ToPtr[int64](320),
				H:   ptrutil.ToPtr[int64](480),
				Pos: pos5,
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 320, H: 480}},
					Pos:    pos5,
				}},
			},
			expected: &openrtb2.Video{
				MIMEs: []string{"video/mp4"},
				W:     ptrutil.ToPtr[int64](320),
				H:     ptrutil.ToPtr[int64](480),
				Pos:   pos5,
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 320, H: 480}},
					Pos:    pos5,
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			video := tt.video
			restoreApsVideoFields(video, tt.apsVideo, tt.adFormat)
			assertExpectedVideo(t, tt.expected, video)
		})
	}
}

func TestDetermineAdFormat(t *testing.T) {
	tests := []struct {
		name string
		imp  openrtb2.Imp
		want string
	}{
		{
			name: "rewarded_via_rwdd",
			imp: openrtb2.Imp{
				Rwdd:  1,
				Video: &openrtb2.Video{},
			},
			want: apsAdFormatRewarded,
		},
		{
			name: "rewarded_via_video_ext_videotype",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{Ext: json.RawMessage(`{"videotype":"rewarded"}`)},
			},
			want: apsAdFormatRewarded,
		},
		{
			name: "interstitial",
			imp: openrtb2.Imp{
				Instl:  1,
				Banner: &openrtb2.Banner{},
				Video:  &openrtb2.Video{},
			},
			want: apsAdFormatInterstitial,
		},
		{
			name: "mrec_via_banner_dimensions",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)},
			},
			want: apsAdFormatMrec,
		},
		{
			name: "mrec_via_banner_format",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				},
			},
			want: apsAdFormatMrec,
		},
		{
			name: "mrec_via_video_dimensions",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)},
			},
			want: apsAdFormatMrec,
		},
		{
			name: "banner_non_mrec_dimensions",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](728), H: ptrutil.ToPtr[int64](90)},
			},
			want: apsAdFormatBanner,
		},
		{
			name: "banner_via_format_non_mrec_dimensions",
			imp: openrtb2.Imp{
				Banner: &openrtb2.Banner{
					Format: []openrtb2.Format{{W: 728, H: 90}},
				},
			},
			want: apsAdFormatBanner,
		},
		{
			name: "video_only_non_mrec_dimensions",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](360)},
			},
			want: "",
		},
		{
			name: "video_only_without_dimensions",
			imp: openrtb2.Imp{
				Video: &openrtb2.Video{},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, determineAdFormat(&tt.imp))
		})
	}
}

func TestApplyAdFormatModifications(t *testing.T) {
	tests := []struct {
		name           string
		adFormat       string
		signalExt      []byte
		request        *openrtb2.BidRequest
		prepSignalImps []openrtb2.Imp
		prepSignalUser *openrtb2.User
		apsVideo       *apsVideoFields
		expected       adFormatExpected
	}{
		{
			name:     "banner_removes_video_and_sets_user_ext",
			adFormat: apsAdFormatBanner,
			signalExt: json.RawMessage(`{
				"extendedsignal": {
					"banner": {"impdepth": 2, "lastadomain": "banner.example.com"}
				}
			}`),
			request: &openrtb2.BidRequest{
				Imp:  []openrtb2.Imp{{ID: "1", Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}},
				User: &openrtb2.User{},
			},
			expected: adFormatExpected{
				Imp: expectedImp{
					VideoIsNil: true,
				},
				User: expectedUser{
					Ext: json.RawMessage(`{"impdepth":2,"lastadomain":"banner.example.com"}`),
				},
			},
		},
		{
			name:     "mrec_applies_video_and_user_ext",
			adFormat: apsAdFormatMrec,
			signalExt: json.RawMessage(`{
				"extendedsignal": {
					"mrec": {
						"videoplacement": 5,
						"videoplcmt": 3,
						"ctaoverlay": true,
						"impdepth": 1,
						"lastadomain": "mrec.example.com"
					}
				}
			}`),
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:     "1",
					Banner: &openrtb2.Banner{},
					Video:  &openrtb2.Video{MIMEs: []string{"video/mp4"}},
				}},
				User: &openrtb2.User{},
			},
			apsVideo: &apsVideoFields{
				W: ptrutil.ToPtr[int64](320),
				H: ptrutil.ToPtr[int64](250),
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				}},
			},
			expected: adFormatExpected{
				Imp: expectedImp{
					Video: &openrtb2.Video{
						W:         ptrutil.ToPtr[int64](320),
						H:         ptrutil.ToPtr[int64](250),
						Placement: adcom1.VideoPlacementSubtype(5),
						Plcmt:     adcom1.VideoPlcmtSubtype(3),
						CompanionAd: []openrtb2.Banner{{
							Format: []openrtb2.Format{{W: 300, H: 250}},
						}},
					},
					Ext: json.RawMessage(`{"owsdk":{"ctaoverlay":true}}`),
				},
				User: expectedUser{
					Ext: json.RawMessage(`{"impdepth":1,"lastadomain":"mrec.example.com"}`),
				},
			},
		},
		{
			name:     "interstitial_sets_skoverlay_on_ios",
			adFormat: apsAdFormatInterstitial,
			signalExt: json.RawMessage(`{
				"extendedsignal": {
					"interstitial": {
						"videoplacement": 5,
						"videoplcmt": 3,
						"companionapi": [5, 6],
						"skoverlay": 1,
						"ctaoverlay": true,
						"impdepth": 0,
						"lastadomain": "example.com"
					}
				}
			}`),
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:     "1",
					Banner: &openrtb2.Banner{},
					Video:  &openrtb2.Video{},
				}},
				Device: &openrtb2.Device{OS: "iOS"},
				User:   &openrtb2.User{},
			},
			apsVideo: &apsVideoFields{W: ptrutil.ToPtr[int64](320), H: ptrutil.ToPtr[int64](480)},
			expected: adFormatExpected{
				Imp: expectedImp{
					Banner: &openrtb2.Banner{},
					Video: &openrtb2.Video{
						W:         ptrutil.ToPtr[int64](320),
						H:         ptrutil.ToPtr[int64](480),
						Placement: adcom1.VideoPlacementSubtype(5),
						Plcmt:     adcom1.VideoPlcmtSubtype(3),
						CompanionAd: []openrtb2.Banner{{
							API: []adcom1.APIFramework{5, 6},
						}},
					},
					Ext: json.RawMessage(`{"owsdk":{"ctaoverlay":true},"skadn":{"skoverlay":1}}`),
				},
				User: expectedUser{
					Ext: json.RawMessage(`{"impdepth":0,"lastadomain":"example.com"}`),
				},
			},
		},
		{
			name:     "interstitial_skips_skoverlay_on_android",
			adFormat: apsAdFormatInterstitial,
			signalExt: json.RawMessage(`{
				"extendedsignal": {
					"interstitial": {"skoverlay": 1, "ctaoverlay": true}
				}
			}`),
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:     "1",
					Banner: &openrtb2.Banner{},
					Video:  &openrtb2.Video{},
				}},
				Device: &openrtb2.Device{OS: "android"},
				User:   &openrtb2.User{},
			},
			expected: adFormatExpected{
				Imp: expectedImp{
					Video: &openrtb2.Video{
						CompanionAd: []openrtb2.Banner{{}},
					},
					Ext: json.RawMessage(`{"owsdk":{"ctaoverlay":true}}`),
				},
				User: expectedUser{ExtIsNil: true},
			},
		},
		{
			name:     "mrec_applies_extended_signal_over_prepopulated_signal_merge",
			adFormat: apsAdFormatMrec,
			signalExt: json.RawMessage(`{
				"extendedsignal": {
					"mrec": {
						"videoplacement": 5,
						"videoplcmt": 3,
						"ctaoverlay": true,
						"impdepth": 1,
						"lastadomain": "mrec.example.com"
					}
				}
			}`),
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:     "i1",
					Banner: &openrtb2.Banner{W: ptrutil.ToPtr[int64](300), H: ptrutil.ToPtr[int64](250)},
					Video: &openrtb2.Video{
						W:     ptrutil.ToPtr[int64](320),
						H:     ptrutil.ToPtr[int64](250),
						BAttr: []adcom1.CreativeAttribute{1, 2},
					},
				}},
				User: &openrtb2.User{},
			},
			prepSignalImps: []openrtb2.Imp{{
				DisplayManager:    "signal-dm",
				DisplayManagerVer: "2.0.0",
				Banner: &openrtb2.Banner{
					API: []adcom1.APIFramework{7},
				},
				Video: &openrtb2.Video{
					MIMEs: []string{"video/mp4"},
				},
				Ext: json.RawMessage(`{"skadn":{"versions":["2.0"]},"owsdk":{"existing":1}}`),
			}},
			prepSignalUser: &openrtb2.User{
				Ext: json.RawMessage(`{"sessionduration":300,"consent":"test"}`),
			},
			apsVideo: &apsVideoFields{
				W:     ptrutil.ToPtr[int64](320),
				H:     ptrutil.ToPtr[int64](250),
				BAttr: []adcom1.CreativeAttribute{1, 2},
				CompanionAd: []openrtb2.Banner{{
					Format: []openrtb2.Format{{W: 300, H: 250}},
				}},
			},
			expected: adFormatExpected{
				Imp: expectedImp{
					DisplayManager:    "signal-dm",
					DisplayManagerVer: "2.0.0",
					Banner:            &openrtb2.Banner{API: []adcom1.APIFramework{7}},
					Video: &openrtb2.Video{
						W:         ptrutil.ToPtr[int64](320),
						H:         ptrutil.ToPtr[int64](250),
						Placement: adcom1.VideoPlacementSubtype(5),
						Plcmt:     adcom1.VideoPlcmtSubtype(3),
						MIMEs:     []string{"video/mp4"},
						BAttr:     []adcom1.CreativeAttribute{1, 2},
						CompanionAd: []openrtb2.Banner{{
							Format: []openrtb2.Format{{W: 300, H: 250}},
						}},
					},
					Ext: json.RawMessage(`{"skadn":{"versions":["2.0"]},"owsdk":{"existing":1,"ctaoverlay":true}}`),
				},
				User: expectedUser{
					Ext: json.RawMessage(`{"sessionduration":300,"consent":"test","impdepth":1,"lastadomain":"mrec.example.com"}`),
				},
			},
		},
		{
			name:     "rewarded_removes_banner",
			adFormat: apsAdFormatRewarded,
			signalExt: json.RawMessage(`{
				"extendedsignal": {
					"rewarded": {
						"videoplacement": 5,
						"videoplcmt": 3,
						"impdepth": 3,
						"lastadomain": "reward.example.com"
					}
				}
			}`),
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					ID:     "1",
					Banner: &openrtb2.Banner{},
					Video:  &openrtb2.Video{},
				}},
				User: &openrtb2.User{},
			},
			apsVideo: &apsVideoFields{W: ptrutil.ToPtr[int64](640), H: ptrutil.ToPtr[int64](360)},
			expected: adFormatExpected{
				Imp: expectedImp{
					BannerIsNil: true,
					Video: &openrtb2.Video{
						W:           ptrutil.ToPtr[int64](640),
						H:           ptrutil.ToPtr[int64](360),
						Placement:   adcom1.VideoPlacementSubtype(5),
						Plcmt:       adcom1.VideoPlcmtSubtype(3),
						CompanionAd: []openrtb2.Banner{{}},
					},
				},
				User: expectedUser{
					Ext: json.RawMessage(`{"impdepth":3,"lastadomain":"reward.example.com"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepSignalImps != nil {
				updateImpressionWithSignal(tt.request, tt.prepSignalImps, tt.adFormat, tt.apsVideo)
			} else if tt.apsVideo != nil && len(tt.request.Imp) > 0 && tt.request.Imp[0].Video != nil {
				restoreApsVideoFields(tt.request.Imp[0].Video, tt.apsVideo, tt.adFormat)
			}
			if tt.prepSignalUser != nil {
				updateUser(tt.request, tt.prepSignalUser)
			}

			applyAdFormatModifications(tt.request, tt.adFormat, tt.signalExt)
			assertAdFormatExpected(t, tt.request, tt.expected)
		})
	}
}

type adFormatExpected struct {
	Imp  expectedImp
	User expectedUser
}

type expectedImp struct {
	DisplayManager    string
	DisplayManagerVer string
	Banner            *openrtb2.Banner
	BannerIsNil       bool
	Video             *openrtb2.Video
	VideoIsNil        bool
	Ext               json.RawMessage
}

type expectedUser struct {
	Ext      json.RawMessage
	ExtIsNil bool
}

func assertAdFormatExpected(t *testing.T, request *openrtb2.BidRequest, expected adFormatExpected) {
	t.Helper()

	require.NotEmpty(t, request.Imp)
	imp := request.Imp[0]

	if expected.Imp.DisplayManager != "" {
		assert.Equal(t, expected.Imp.DisplayManager, imp.DisplayManager)
	}
	if expected.Imp.DisplayManagerVer != "" {
		assert.Equal(t, expected.Imp.DisplayManagerVer, imp.DisplayManagerVer)
	}

	if expected.Imp.BannerIsNil {
		assert.Nil(t, imp.Banner)
	} else if expected.Imp.Banner != nil {
		require.NotNil(t, imp.Banner)
		assertExpectedBanner(t, expected.Imp.Banner, imp.Banner)
	}

	if expected.Imp.VideoIsNil {
		assert.Nil(t, imp.Video)
	} else if expected.Imp.Video != nil {
		require.NotNil(t, imp.Video)
		assertExpectedVideo(t, expected.Imp.Video, imp.Video)
	}

	if len(expected.Imp.Ext) > 0 {
		assert.JSONEq(t, string(expected.Imp.Ext), string(imp.Ext))
	}

	require.NotNil(t, request.User)
	if expected.User.ExtIsNil {
		assert.Nil(t, request.User.Ext)
	} else if len(expected.User.Ext) > 0 {
		assert.JSONEq(t, string(expected.User.Ext), string(request.User.Ext))
	}
}

func assertExpectedBanner(t *testing.T, expected, actual *openrtb2.Banner) {
	t.Helper()

	if expected.Pos != nil {
		require.NotNil(t, actual.Pos)
		assert.Equal(t, *expected.Pos, *actual.Pos)
	}
}

func assertExpectedVideo(t *testing.T, expected, actual *openrtb2.Video) {
	t.Helper()

	if expected.W != nil {
		require.NotNil(t, actual.W)
		assert.Equal(t, *expected.W, *actual.W)
	}
	if expected.H != nil {
		require.NotNil(t, actual.H)
		assert.Equal(t, *expected.H, *actual.H)
	}
	if expected.Pos != nil {
		require.NotNil(t, actual.Pos)
		assert.Equal(t, *expected.Pos, *actual.Pos)
	}
	if expected.Placement != 0 {
		assert.Equal(t, expected.Placement, actual.Placement)
	}
	if expected.Plcmt != 0 {
		assert.Equal(t, expected.Plcmt, actual.Plcmt)
	}
	if len(expected.MIMEs) > 0 {
		assert.Equal(t, expected.MIMEs, actual.MIMEs)
	}
	if len(expected.BAttr) > 0 {
		assert.Equal(t, expected.BAttr, actual.BAttr)
	}
	if len(expected.CompanionAd) > 0 {
		require.NotEmpty(t, actual.CompanionAd)
		assertExpectedBanner(t, &expected.CompanionAd[0], &actual.CompanionAd[0])
		if len(expected.CompanionAd[0].Format) > 0 {
			require.NotEmpty(t, actual.CompanionAd[0].Format)
			assert.Equal(t, expected.CompanionAd[0].Format[0].W, actual.CompanionAd[0].Format[0].W)
			assert.Equal(t, expected.CompanionAd[0].Format[0].H, actual.CompanionAd[0].Format[0].H)
		}
		if len(expected.CompanionAd[0].API) > 0 {
			assert.Equal(t, expected.CompanionAd[0].API, actual.CompanionAd[0].API)
		}
	}
}
