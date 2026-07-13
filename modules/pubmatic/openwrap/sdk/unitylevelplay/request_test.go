package unitylevelplay

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	mock_metrics "github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/metrics/mock"
	"github.com/prebid/prebid-server/v3/modules/pubmatic/openwrap/models"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestModifyRequestWithUnityLevelPlayParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetrics := mock_metrics.NewMockMetricsEngine(ctrl)

	tests := []struct {
		name             string
		requestBody      []byte
		expectedResponse []byte
		expectedError    bool
		metricsSetup     func(*mock_metrics.MockMetricsEngine)
	}{
		{
			name:             "empty request body",
			requestBody:      nil,
			expectedResponse: nil,
		},
		{
			name:             "invalid JSON request body",
			requestBody:      []byte(`invalid json`),
			expectedResponse: []byte(`invalid json`),
			expectedError:    true,
		},
		{
			name:             "request with app but no ext",
			requestBody:      []byte(`{"id":"test","app":{"id":"app1"}}`),
			expectedResponse: []byte(`{"id":"test","app":{"id":"app1"}, "imp":null}`),
		},
		{
			name:        "request with publisher ID and missing token",
			requestBody: []byte(`{"id":"test","app":{"id":"app1","publisher":{"id":"pub1"},"ext":{}}}`),
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("pub1", "", models.MissingSignal)
			},
			expectedResponse: []byte(`{"id":"test","imp":null,"app":{"id":"app1","publisher":{"id":"pub1"},"ext":{}}}`),
		},
		{
			name:        "request with profile ID and missing token",
			requestBody: []byte(`{"id":"test","app":{"id":"app1","ext":{}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":"12345"}}}}}}`),
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("", "12345", models.MissingSignal)
			},
			expectedResponse: []byte(`{"id":"test","imp":null,"app":{"id":"app1","ext":{}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":"12345"}}}}}}`),
		},
		{
			name:        "request with both publisher ID and profile ID and missing token",
			requestBody: []byte(`{"id":"test","app":{"id":"app1","publisher":{"id":"pub1"},"ext":{}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":"12345"}}}}}}`),
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("pub1", "12345", models.MissingSignal)
			},
			expectedResponse: []byte(`{"id":"test","imp":null,"app":{"id":"app1","publisher":{"id":"pub1"},"ext":{}},"ext":{"prebid":{"bidderparams":{"pubmatic":{"wrapper":{"profileid":"12345"}}}}}}`),
		},
		{
			name:        "request with invalid token",
			requestBody: []byte(`{"id":"test","app":{"id":"app1","ext":{"token":"aW52YWxpZCB0b2tlbg=="}}}`),
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("", "", models.InvalidSignal)
			},
			expectedResponse: []byte(`{"id":"test","imp":null,"app":{"id":"app1","ext":{"token":"aW52YWxpZCB0b2tlbg=="}}}`),
		},
		{
			name:             "request with valid token and signal data",
			requestBody:      []byte(`{"id":"test","app":{"id":"app1","ext":{"token":"eyJpZCI6InNpZ25hbCIsImFwcCI6eyJuYW1lIjoidGVzdGFwcCJ9LCJpbXAiOlt7ImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCIsImNsaWNrYnJvd3NlciI6MSwidmlkZW8iOnsibWltZXMiOlsidmlkZW8vbXA0Il0sImV4dCI6eyJyZXdhcmQiOjF9fX1dfQ=="}},"imp":[{"id":"1","video":{"ext":{"reward":1}}}]}`),
			expectedResponse: []byte(`{"id":"test","app":{"id":"app1","name":"testapp","ext":{"token":"eyJpZCI6InNpZ25hbCIsImFwcCI6eyJuYW1lIjoidGVzdGFwcCJ9LCJpbXAiOlt7ImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCIsImNsaWNrYnJvd3NlciI6MSwidmlkZW8iOnsibWltZXMiOlsidmlkZW8vbXA0Il0sImV4dCI6eyJyZXdhcmQiOjF9fX1dfQ=="}},"imp":[{"id":"1","displaymanager":"unity","displaymanagerver":"1.0","clickbrowser":1,"secure":1,"video":{"mimes":["video/mp4"],"ext":{"reward":1}},"instl":1,"rwdd":1}]}`),
			metricsSetup:     func(m *mock_metrics.MockMetricsEngine) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.metricsSetup != nil {
				tt.metricsSetup(mockMetrics)
			}

			levelPlay := NewLevelPlay(mockMetrics)
			response := levelPlay.ModifyRequestWithUnityLevelPlayParams(tt.requestBody)

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

func TestModifyRequestWithSignalData(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMetrics := mock_metrics.NewMockMetricsEngine(ctrl)

	tests := []struct {
		name         string
		request      *openrtb2.BidRequest
		expected     *openrtb2.BidRequest
		metricsSetup func(*mock_metrics.MockMetricsEngine)
	}{
		{
			name:     "nil request",
			request:  nil,
			expected: nil,
		},
		{
			name:     "request with no app",
			request:  &openrtb2.BidRequest{},
			expected: &openrtb2.BidRequest{},
		},
		{
			name: "request with app but no ext",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{},
			},
			expected: &openrtb2.BidRequest{
				App: &openrtb2.App{},
			},
		},
		{
			name: "request with app and ext but no token",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"otherfield":1}`),
				},
			},
			expected: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"otherfield":1}`),
				},
			},
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("", "", models.MissingSignal)
			},
		},
		{
			name: "request with invalid base64 token",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"token":"invalid-base64"}`),
				},
			},
			expected: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"token":"invalid-base64"}`),
				},
			},
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("", "", models.InvalidSignal)
			},
		},
		{
			name: "request with valid base64 but invalid JSON",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"token":"aW52YWxpZCBqc29u"}`), // base64 for "invalid json"
				},
			},
			expected: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"token":"aW52YWxpZCBqc29u"}`),
				},
			},
			metricsSetup: func(m *mock_metrics.MockMetricsEngine) {
				m.EXPECT().RecordSignalDataStatus("", "", models.InvalidSignal)
			},
		},
		{
			name: "request with empty imp array and valid signal",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOlt7ImlkIjoiMSIsImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCJ9XX0="}`), // base64 encoded signal with imp array
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOlt7ImlkIjoiMSIsImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCJ9XX0="}`),
				},
			},
		},
		{
			name: "request with imp but signal has empty imp array",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
					},
				},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOltdfQ=="}`), // base64 encoded signal with empty imp array
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
					},
				},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOltdfQ=="}`),
				},
			},
		},
		{
			name: "request with nil video and valid signal data",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
					},
				},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOlt7ImlkIjoiMSIsImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCIsInZpZGVvIjp7Im1pbWVzIjpbInZpZGVvL21wNCJdLCJleHQiOnsicmV3YXJkIjoxfX19XX0="}`), // base64 encoded signal with video
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:                "1",
						DisplayManager:    "unity",
						DisplayManagerVer: "1.0",
						Video: &openrtb2.Video{
							MIMEs: []string{"video/mp4"},
							Ext:   []byte(`{"reward":1}`),
						},
					},
				},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOlt7ImlkIjoiMSIsImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCIsInZpZGVvIjp7Im1pbWVzIjpbInZpZGVvL21wNCJdLCJleHQiOnsicmV3YXJkIjoxfX19XX0="}`),
				},
			},
		},
		{
			name: "complete signal data",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
						Video: &openrtb2.Video{
							Ext: []byte(`{"reward":1}`),
						},
					},
				},
				App: &openrtb2.App{
					Ext: []byte(`{"token":"eyJpbXAiOlt7ImlkIjoiMSIsImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCIsInZpZGVvIjp7Im1pbWVzIjpbInZpZGVvL21wNCJdLCJleHQiOnsicmV3YXJkIjoxfX19XSwiYXBwIjp7Im5hbWUiOiJ0ZXN0YXBwIiwiZG9tYWluIjoiZXhhbXBsZS5jb20iLCJjYXQiOlsiSUFCMSJdLCJrZXl3b3JkcyI6InRlc3QsYXBwIiwidmVyIjoiMS4wIn0sImRldmljZSI6eyJ1YSI6InRlc3QtdWEiLCJnZW8iOnsiY291bnRyeSI6IlVTIn0sImNhcnJpZXIiOiJ0ZXN0LWNhcnJpZXIiLCJsYW5ndWFnZSI6ImVuIiwiaHd2IjoiMS4wIiwibWNjbW5jIjoiMTIzIiwibWFrZSI6InRlc3QtbWFrZSIsIm1vZGVsIjoidGVzdC1tb2RlbCIsIm9zIjoidGVzdC1vcyIsIm9zdiI6IjEuMCIsInciOjMyMCwiaCI6NDgwLCJweHJhdGlvIjoyLjAsImlmYSI6InRlc3QtaWZhIiwiZXh0Ijp7ImF0dHMiOjF9fSwidXNlciI6eyJ5b2IiOjIwMDAsImdlbmRlciI6Ik0iLCJrZXl3b3JkcyI6InRlc3QsdXNlciIsImRhdGEiOlt7ImlkIjoiMSJ9XSwiZXh0Ijp7ImNvbnNlbnQiOiJ0ZXN0IiwiZWlkcyI6W3sic291cmNlIjoidGVzdCJ9XSwiaW1wZGVwdGgiOjEsInNlc3Npb25kdXJhdGlvbiI6MzAwfX0sInJlZ3MiOnsiZXh0Ijp7ImdkcHIiOjEsInVzX3ByaXZhY3kiOiJ0ZXN0In19LCJzb3VyY2UiOnsiZXh0Ijp7Im9taWRwbiI6InRlc3QiLCJvbWlkcHYiOiIxLjAifX0sImV4dCI6eyJ3cmFwcGVyIjp7ImNsaWVudGNvbmZpZyI6MX19fQ=="}`), // base64 encoded complete signal
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID:                "1",
						DisplayManager:    "unity",
						DisplayManagerVer: "1.0",
						Video: &openrtb2.Video{
							MIMEs: []string{"video/mp4"},
							Ext:   []byte(`{"reward":1}`),
						},
					},
				},
				App: &openrtb2.App{
					Name:     "testapp",
					Domain:   "example.com",
					Cat:      []string{"IAB1"},
					Keywords: "test,app",
					Ver:      "1.0",
					Ext:      []byte(`{"token":"eyJpbXAiOlt7ImlkIjoiMSIsImRpc3BsYXltYW5hZ2VyIjoidW5pdHkiLCJkaXNwbGF5bWFuYWdlcnZlciI6IjEuMCIsInZpZGVvIjp7Im1pbWVzIjpbInZpZGVvL21wNCJdLCJleHQiOnsicmV3YXJkIjoxfX19XSwiYXBwIjp7Im5hbWUiOiJ0ZXN0YXBwIiwiZG9tYWluIjoiZXhhbXBsZS5jb20iLCJjYXQiOlsiSUFCMSJdLCJrZXl3b3JkcyI6InRlc3QsYXBwIiwidmVyIjoiMS4wIn0sImRldmljZSI6eyJ1YSI6InRlc3QtdWEiLCJnZW8iOnsiY291bnRyeSI6IlVTIn0sImNhcnJpZXIiOiJ0ZXN0LWNhcnJpZXIiLCJsYW5ndWFnZSI6ImVuIiwiaHd2IjoiMS4wIiwibWNjbW5jIjoiMTIzIiwibWFrZSI6InRlc3QtbWFrZSIsIm1vZGVsIjoidGVzdC1tb2RlbCIsIm9zIjoidGVzdC1vcyIsIm9zdiI6IjEuMCIsInciOjMyMCwiaCI6NDgwLCJweHJhdGlvIjoyLjAsImlmYSI6InRlc3QtaWZhIiwiZXh0Ijp7ImF0dHMiOjF9fSwidXNlciI6eyJ5b2IiOjIwMDAsImdlbmRlciI6Ik0iLCJrZXl3b3JkcyI6InRlc3QsdXNlciIsImRhdGEiOlt7ImlkIjoiMSJ9XSwiZXh0Ijp7ImNvbnNlbnQiOiJ0ZXN0IiwiZWlkcyI6W3sic291cmNlIjoidGVzdCJ9XSwiaW1wZGVwdGgiOjEsInNlc3Npb25kdXJhdGlvbiI6MzAwfX0sInJlZ3MiOnsiZXh0Ijp7ImdkcHIiOjEsInVzX3ByaXZhY3kiOiJ0ZXN0In19LCJzb3VyY2UiOnsiZXh0Ijp7Im9taWRwbiI6InRlc3QiLCJvbWlkcHYiOiIxLjAifX0sImV4dCI6eyJ3cmFwcGVyIjp7ImNsaWVudGNvbmZpZyI6MX19fQ=="}`),
				},
				Device: &openrtb2.Device{
					UA:       "test-ua",
					Geo:      &openrtb2.Geo{Country: "US"},
					Carrier:  "test-carrier",
					Language: "en",
					HWV:      "1.0",
					MCCMNC:   "123",
					Make:     "test-make",
					Model:    "test-model",
					OS:       "test-os",
					OSV:      "1.0",
					W:        320,
					H:        480,
					PxRatio:  2.0,
					IFA:      "test-ifa",
					Ext:      []byte(`{"atts":1}`),
				},
				User: &openrtb2.User{
					Yob:      2000,
					Gender:   "M",
					Keywords: "test,user",
					Data:     []openrtb2.Data{{ID: "1"}},
					Ext:      []byte(`{"consent":"test","eids":[{"source":"test"}],"impdepth":1,"sessionduration":300}`),
				},
				Regs: &openrtb2.Regs{
					Ext: []byte(`{"gdpr":1,"us_privacy":"test"}`),
				},
				Source: &openrtb2.Source{
					Ext: []byte(`{"omidpn":"test","omidpv":"1.0"}`),
				},
				Ext: []byte(`{"wrapper":{"clientconfig":1}}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup metrics mock expectations if provided
			if tt.metricsSetup != nil {
				tt.metricsSetup(mockMetrics)
			}

			levelPlay := &LevelPlay{
				metricsEngine: mockMetrics,
			}
			levelPlay.modifyRequestWithSignalData(tt.request)

			if tt.expected == nil {
				assert.Nil(t, tt.request)
				return
			}

			// Compare entire request by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyBanner(t *testing.T) {
	tests := []struct {
		name           string
		requestBanner  *openrtb2.Banner
		signalBanner   *openrtb2.Banner
		expectedBanner *openrtb2.Banner
	}{
		{
			name:           "nil signal banner",
			requestBanner:  &openrtb2.Banner{API: []adcom1.APIFramework{1}},
			signalBanner:   nil,
			expectedBanner: &openrtb2.Banner{API: []adcom1.APIFramework{1}},
		},
		{
			name:           "nil request banner API",
			requestBanner:  nil,
			signalBanner:   &openrtb2.Banner{API: []adcom1.APIFramework{1}},
			expectedBanner: nil,
		},
		{
			name:           "request banner with API",
			requestBanner:  &openrtb2.Banner{API: []adcom1.APIFramework{1}},
			signalBanner:   &openrtb2.Banner{API: []adcom1.APIFramework{2}},
			expectedBanner: &openrtb2.Banner{API: []adcom1.APIFramework{2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyBanner(tt.requestBanner, tt.signalBanner)

			// Compare banners by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expectedBanner)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.requestBanner)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyImpression(t *testing.T) {
	tests := []struct {
		name       string
		request    *openrtb2.BidRequest
		signalImps []openrtb2.Imp
		expected   *openrtb2.BidRequest
	}{
		{
			name:       "empty request imp array",
			request:    &openrtb2.BidRequest{},
			signalImps: []openrtb2.Imp{{ID: "1"}},
			expected:   &openrtb2.BidRequest{},
		},
		{
			name: "empty signal imp array",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
			signalImps: []openrtb2.Imp{},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{ID: "1"}},
			},
		},
		{
			name: "copy display fields",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{ID: "1"},
				},
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
			name: "video object",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
						Video: &openrtb2.Video{
							Ext: []byte(`{"reward":0}`),
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
			name: "native object",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
						Native: &openrtb2.Native{
							Request: "request-native",
						},
					},
				},
			},
			signalImps: []openrtb2.Imp{
				{
					ID: "1",
					Native: &openrtb2.Native{
						Request: "signal-native",
					},
				},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{
					{
						ID: "1",
						Native: &openrtb2.Native{
							Request: "signal-native",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyImpression(tt.request, tt.signalImps)

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyImpExtension(t *testing.T) {
	tests := []struct {
		name           string
		requestImpExt  []byte
		signalImpExt   []byte
		expectedImpExt []byte
	}{
		{
			name:           "nil signal imp ext",
			requestImpExt:  []byte(`{"existingfield":1}`),
			signalImpExt:   nil,
			expectedImpExt: []byte(`{"existingfield":1}`),
		},
		{
			name:           "empty request imp ext",
			requestImpExt:  []byte{},
			signalImpExt:   []byte(`{"skadn":{"version":"2.0"}}`),
			expectedImpExt: []byte(`{"skadn":{"version":"2.0"}}`),
		},
		{
			name:           "copy all skadn fields",
			requestImpExt:  []byte(`{"existingfield":1}`),
			signalImpExt:   []byte(`{"skadn":{"versions":["2.0","3.0"],"version":"2.0","skoverlay":true,"productpage":"https://example.com","skadnetids":["123","456"]}}`),
			expectedImpExt: []byte(`{"existingfield":1,"skadn":{"versions":["2.0","3.0"],"version":"2.0","skoverlay":true,"productpage":"https://example.com","skadnetids":["123","456"]}}`),
		},
		{
			name:           "copy gpid",
			requestImpExt:  []byte(`{"existingfield":1}`),
			signalImpExt:   []byte(`{"gpid":"test-gpid"}`),
			expectedImpExt: []byte(`{"existingfield":1,"gpid":"test-gpid"}`),
		},
		{
			name:           "copy partial skadn fields",
			requestImpExt:  []byte(`{"existingfield":1}`),
			signalImpExt:   []byte(`{"skadn":{"version":"2.0","skoverlay":true}}`),
			expectedImpExt: []byte(`{"existingfield":1,"skadn":{"version":"2.0","skoverlay":true}}`),
		},
		{
			name:           "copy owsdk fields",
			requestImpExt:  []byte(`{"existingfield":1}`),
			signalImpExt:   []byte(`{"owsdk":{"ctaoverlay":1}}`),
			expectedImpExt: []byte(`{"existingfield":1,"owsdk":{"ctaoverlay":1}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualImpExt := modifyImpExtension(tt.requestImpExt, tt.signalImpExt)
			assert.JSONEq(t, string(tt.expectedImpExt), string(actualImpExt))
		})
	}
}

func TestModifyRegs(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.Regs
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil signal regs",
			request:  &openrtb2.BidRequest{Regs: &openrtb2.Regs{COPPA: 1}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Regs: &openrtb2.Regs{COPPA: 1}},
		},
		{
			name:     "nil request regs",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.Regs{COPPA: 1},
			expected: &openrtb2.BidRequest{Regs: &openrtb2.Regs{COPPA: 1}},
		},
		{
			name:     "copy COPPA",
			request:  &openrtb2.BidRequest{Regs: &openrtb2.Regs{}},
			signal:   &openrtb2.Regs{COPPA: 1},
			expected: &openrtb2.BidRequest{Regs: &openrtb2.Regs{COPPA: 1}},
		},
		{
			name:     "zero COPPA not copied",
			request:  &openrtb2.BidRequest{Regs: &openrtb2.Regs{COPPA: 1}},
			signal:   &openrtb2.Regs{COPPA: 0},
			expected: &openrtb2.BidRequest{Regs: &openrtb2.Regs{COPPA: 1}},
		},
		{
			name:     "copy all extension fields",
			request:  &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: []byte(`{"existingfield":1}`)}},
			signal:   &openrtb2.Regs{Ext: []byte(`{"gpp":"test-gpp","gpp_sid":[1,2],"gdpr":1,"us_privacy":"test","dsa":{"dsarequired":true,"pubrender":true,"datatopub":true}}`)},
			expected: &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: []byte(`{"existingfield":1,"gpp":"test-gpp","gpp_sid":[1,2],"gdpr":1,"us_privacy":"test","dsa":{"dsarequired":true,"pubrender":true,"datatopub":true}}`)}},
		},
		{
			name:     "copy partial extension fields",
			request:  &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: []byte(`{"existingfield":1}`)}},
			signal:   &openrtb2.Regs{Ext: []byte(`{"gdpr":1,"us_privacy":"test"}`)},
			expected: &openrtb2.BidRequest{Regs: &openrtb2.Regs{Ext: []byte(`{"existingfield":1,"gdpr":1,"us_privacy":"test"}`)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyRegs(tt.request, tt.signal)

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyApp(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.App
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil signal app",
			request:  &openrtb2.BidRequest{App: &openrtb2.App{Name: "test"}},
			signal:   nil,
			expected: &openrtb2.BidRequest{App: &openrtb2.App{Name: "test"}},
		},
		{
			name:     "nil request app",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.App{Name: "test"},
			expected: &openrtb2.BidRequest{App: &openrtb2.App{Name: "test"}},
		},
		{
			name:    "copy all app fields",
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
			name: "empty signal fields not copied",
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
			name: "partial signal fields copied",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyApp(tt.request, tt.signal)

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyDevice(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.Device
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil signal device",
			request:  &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{UA: "test-ua"}},
		},
		{
			name:     "nil request device",
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
			name:    "copy all device fields",
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
				Ext:            []byte(`{"atts":1}`),
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
				Ext:            []byte(`{"atts":1}`),
			}},
		},
		{
			name: "empty signal fields not copied",
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
			name: "partial signal fields copied",
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
				Ext: []byte(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			}},
		},
		{
			name: "request_has_ifv_signal_does_not",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
			signal: &openrtb2.Device{
				Make: "test-make",
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:   "test-ua",
				Make: "test-make",
				Ext:  []byte(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
		},
		{
			name: "both_request_and_signal_have_ifv_signal_wins",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
			signal: &openrtb2.Device{
				Ext: []byte(`{"ifv":"SIGNAL-IFV-VALUE"}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"ifv":"SIGNAL-IFV-VALUE"}`),
			}},
		},
		{
			name: "signal_has_empty_ifv_overwrites_request_ifv",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"ifv":"REQUEST-IFV-VALUE"}`),
			}},
			signal: &openrtb2.Device{
				Ext: []byte(`{"ifv":""}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"ifv":""}`),
			}},
		},
		{
			name: "signal_has_both_atts_and_ifv",
			request: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA: "test-ua",
			}},
			signal: &openrtb2.Device{
				Ext: []byte(`{"atts":3,"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			},
			expected: &openrtb2.BidRequest{Device: &openrtb2.Device{
				UA:  "test-ua",
				Ext: []byte(`{"atts":3,"ifv":"193DBF06-B1D8-4684-BE35-0FB0770C463C"}`),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyDevice(tt.request, tt.signal)

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyUser(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.User
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil signal user",
			request:  &openrtb2.BidRequest{User: &openrtb2.User{Yob: 2000}},
			signal:   nil,
			expected: &openrtb2.BidRequest{User: &openrtb2.User{Yob: 2000}},
		},
		{
			name:     "nil request user",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.User{Yob: 2000},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{Yob: 2000}},
		},
		{
			name:    "copy all user fields",
			request: &openrtb2.BidRequest{User: &openrtb2.User{}},
			signal: &openrtb2.User{
				Data:     []openrtb2.Data{{ID: "1"}},
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      []byte(`{"sessionduration":300,"impdepth":1,"consent":"test","eids":[{"source":"test"}]}`),
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Data:     []openrtb2.Data{{ID: "1"}},
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      []byte(`{"sessionduration":300,"impdepth":1,"consent":"test","eids":[{"source":"test"}]}`),
			}},
		},
		{
			name: "empty signal fields not copied",
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
			name: "partial signal fields copied",
			request: &openrtb2.BidRequest{User: &openrtb2.User{
				Yob:    2000,
				Gender: "M",
			}},
			signal: &openrtb2.User{
				Keywords: "test,user",
				Ext:      []byte(`{"sessionduration":300}`),
			},
			expected: &openrtb2.BidRequest{User: &openrtb2.User{
				Yob:      2000,
				Gender:   "M",
				Keywords: "test,user",
				Ext:      []byte(`{"sessionduration":300}`),
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifyUser(tt.request, tt.signal)

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifyRequestWithStaticData(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil request",
			request:  nil,
			expected: nil,
		},
		{
			name: "request with video reward=1",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					Video: &openrtb2.Video{
						Ext: []byte(`{"reward":1}`),
					},
					Banner: &openrtb2.Banner{},
				}},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					Instl:  1,
					Rwdd:   1,
					Secure: ptrutil.ToPtr(int8(1)),
				}},
			},
		},
		{
			name: "request with video but no reward",
			request: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					Video: &openrtb2.Video{
						Ext: []byte(`{}`),
					},
					Banner: &openrtb2.Banner{},
				}},
			},
			expected: &openrtb2.BidRequest{
				Imp: []openrtb2.Imp{{
					Banner: &openrtb2.Banner{},
					Secure: ptrutil.ToPtr(int8(1)),
				}},
			},
		},
		{
			name: "request with app having sessionDepth",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{"sessionDepth":5}`),
				},
				Imp: []openrtb2.Imp{{
					Video:  &openrtb2.Video{},
					Banner: &openrtb2.Banner{},
				}},
			},
			expected: &openrtb2.BidRequest{
				App: &openrtb2.App{
					Ext: []byte(`{}`),
				},
				Imp: []openrtb2.Imp{{
					Banner: &openrtb2.Banner{},
					Secure: ptrutil.ToPtr(int8(1)),
				}},
			},
		},
		{
			name: "request with app but no sessionDepth",
			request: &openrtb2.BidRequest{
				App: &openrtb2.App{},
				Imp: []openrtb2.Imp{{
					Video:  &openrtb2.Video{},
					Banner: &openrtb2.Banner{},
				}},
			},
			expected: &openrtb2.BidRequest{
				App: &openrtb2.App{},
				Imp: []openrtb2.Imp{{
					Banner: &openrtb2.Banner{},
					Secure: ptrutil.ToPtr(int8(1)),
				}},
			},
		},
	}

	l := &LevelPlay{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.modifyRequestWithStaticData(tt.request)

			if tt.expected == nil {
				assert.Nil(t, tt.request)
				return
			}

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}

func TestModifySource(t *testing.T) {
	tests := []struct {
		name     string
		request  *openrtb2.BidRequest
		signal   *openrtb2.Source
		expected *openrtb2.BidRequest
	}{
		{
			name:     "nil signal source",
			request:  &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"existingfield":1}`)}},
			signal:   nil,
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"existingfield":1}`)}},
		},
		{
			name:     "nil request source",
			request:  &openrtb2.BidRequest{},
			signal:   &openrtb2.Source{Ext: []byte(`{"omidpn":"test","omidpv":"1.0"}`)},
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"omidpn":"test","omidpv":"1.0"}`)}},
		},
		{
			name:     "copy all source fields",
			request:  &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"existingfield":1}`)}},
			signal:   &openrtb2.Source{Ext: []byte(`{"omidpn":"test","omidpv":"1.0"}`)},
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"existingfield":1,"omidpn":"test","omidpv":"1.0"}`)}},
		},
		{
			name:     "partial source fields copied",
			request:  &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"existingfield":1}`)}},
			signal:   &openrtb2.Source{Ext: []byte(`{"omidpn":"test"}`)},
			expected: &openrtb2.BidRequest{Source: &openrtb2.Source{Ext: []byte(`{"existingfield":1,"omidpn":"test"}`)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modifySource(tt.request, tt.signal)

			// Compare requests by marshaling to JSON
			expectedJSON, err := jsoniterator.Marshal(tt.expected)
			assert.NoError(t, err)

			actualJSON, err := jsoniterator.Marshal(tt.request)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(actualJSON))
		})
	}
}
