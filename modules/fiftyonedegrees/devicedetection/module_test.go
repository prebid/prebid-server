package devicedetection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"testing"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockAccValidator struct {
	mock.Mock
}

func (m *mockAccValidator) isAllowed(cfg config, req []byte) bool {
	args := m.Called(cfg, req)
	return args.Bool(0)
}

type mockEvidenceExtractor struct {
	mock.Mock
}

func (m *mockEvidenceExtractor) fromHeaders(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []stringEvidence {
	args := m.Called(request, httpHeaderKeys)

	return args.Get(0).([]stringEvidence)
}

func (m *mockEvidenceExtractor) fromSuaPayload(payload []byte) []stringEvidence {
	args := m.Called(payload)

	return args.Get(0).([]stringEvidence)
}

func (m *mockEvidenceExtractor) extract(ctx hookstage.ModuleContext) ([]onpremise.Evidence, string, error) {
	args := m.Called(ctx)

	res := args.Get(0)
	if res == nil {
		return nil, args.String(1), args.Error(2)
	}

	return res.([]onpremise.Evidence), args.String(1), args.Error(2)
}

type mockDeviceDetector struct {
	mock.Mock
}

func (m *mockDeviceDetector) getSupportedHeaders() []dd.EvidenceKey {
	args := m.Called()
	return args.Get(0).([]dd.EvidenceKey)
}

func (m *mockDeviceDetector) getDeviceInfo(evidence []onpremise.Evidence, ua string) (*deviceInfo, error) {

	args := m.Called(evidence, ua)

	res := args.Get(0)

	if res == nil {
		return nil, args.Error(1)
	}

	return res.(*deviceInfo), args.Error(1)
}

func TestHandleEntrypointHookAccountNotAllowed(t *testing.T) {
	var mockValidator mockAccValidator

	mockValidator.On("isAllowed", mock.Anything, mock.Anything).Return(false)

	module := Module{
		accountValidator: &mockValidator,
	}

	_, err := module.HandleEntrypointHook(nil, hookstage.ModuleInvocationContext{}, hookstage.EntrypointPayload{})
	assert.Error(t, err)
	assert.Equal(t, "hook execution failed: account not allowed", err.Error())
}

func TestHandleEntrypointHookAccountAllowed(t *testing.T) {
	var mockValidator mockAccValidator

	mockValidator.On("isAllowed", mock.Anything, mock.Anything).Return(true)

	var mockEvidenceExtractor mockEvidenceExtractor
	mockEvidenceExtractor.On("fromHeaders", mock.Anything, mock.Anything).Return(
		[]stringEvidence{{
			Prefix: "123",
			Key:    "key",
			Value:  "val",
		}},
	)

	mockEvidenceExtractor.On("fromSuaPayload", mock.Anything, mock.Anything).Return(
		[]stringEvidence{{
			Prefix: "123",
			Key:    "User-Agent",
			Value:  "ua",
		}},
	)

	var mockDeviceDetector mockDeviceDetector

	mockDeviceDetector.On("getSupportedHeaders").Return(
		[]dd.EvidenceKey{{
			Prefix: dd.HttpEvidenceQuery,
			Key:    "key",
		}},
	)

	module := Module{
		deviceDetector:    &mockDeviceDetector,
		evidenceExtractor: &mockEvidenceExtractor,
		accountValidator:  &mockValidator,
	}

	result, err := module.HandleEntrypointHook(nil, hookstage.ModuleInvocationContext{}, hookstage.EntrypointPayload{})
	assert.NoError(t, err)

	assert.Equal(
		t, result.ModuleContext[evidenceFromHeadersCtxKey], []stringEvidence{{
			Prefix: "123",
			Key:    "key",
			Value:  "val",
		}},
	)

	assert.Equal(
		t, result.ModuleContext[evidenceFromSuaCtxKey], []stringEvidence{{
			Prefix: "123",
			Key:    "User-Agent",
			Value:  "ua",
		}},
	)
}

func TestHandleRawAuctionHookNoCtx(t *testing.T) {
	module := Module{}

	_, err := module.HandleRawAuctionHook(
		nil,
		hookstage.ModuleInvocationContext{},
		hookstage.RawAuctionRequestPayload{},
	)
	assert.Errorf(t, err, "entrypoint hook was not configured")
}

func TestHandleRawAuctionHookExtractError(t *testing.T) {
	var mockValidator mockAccValidator

	mockValidator.On("isAllowed", mock.Anything, mock.Anything).Return(true)

	var evidenceExtractorM mockEvidenceExtractor
	evidenceExtractorM.On("extract", mock.Anything).Return(
		nil,
		"ua",
		nil,
	)

	var mockDeviceDetector mockDeviceDetector

	module := Module{
		deviceDetector:    &mockDeviceDetector,
		evidenceExtractor: &evidenceExtractorM,
		accountValidator:  &mockValidator,
	}

	mctx := make(hookstage.ModuleContext)

	mctx[ddEnabledCtxKey] = true

	result, err := module.HandleRawAuctionHook(
		context.TODO(), hookstage.ModuleInvocationContext{
			ModuleContext: mctx,
		},
		hookstage.RawAuctionRequestPayload{},
	)

	assert.NoError(t, err)
	assert.Equal(t, len(result.ChangeSet.Mutations()), 1)
	assert.Equal(t, result.ChangeSet.Mutations()[0].Type(), hookstage.MutationUpdate)

	mutation := result.ChangeSet.Mutations()[0]

	body := []byte(`{}`)

	_, err = mutation.Apply(body)
	assert.Errorf(t, err, "error extracting evidence")

	var mockEvidenceErrExtractor mockEvidenceExtractor
	mockEvidenceErrExtractor.On("extract", mock.Anything).Return(
		nil,
		"",
		errors.New("error"),
	)

	module.evidenceExtractor = &mockEvidenceErrExtractor

	result, err = module.HandleRawAuctionHook(
		context.TODO(), hookstage.ModuleInvocationContext{
			ModuleContext: mctx,
		},
		hookstage.RawAuctionRequestPayload{},
	)

	assert.NoError(t, err)

	assert.Equal(t, len(result.ChangeSet.Mutations()), 1)

	assert.Equal(t, result.ChangeSet.Mutations()[0].Type(), hookstage.MutationUpdate)

	mutation = result.ChangeSet.Mutations()[0]

	_, err = mutation.Apply(body)
	assert.Errorf(t, err, "error extracting evidence error")

}

func TestHandleRawAuctionHookEnrichment(t *testing.T) {
	var mockValidator mockAccValidator

	mockValidator.On("isAllowed", mock.Anything, mock.Anything).Return(true)

	var mockEvidenceExtractor mockEvidenceExtractor
	mockEvidenceExtractor.On("extract", mock.Anything).Return(
		[]onpremise.Evidence{
			{
				Key:   "key",
				Value: "val",
			},
		},
		"ua",
		nil,
	)

	var deviceDetectorM mockDeviceDetector

	deviceDetectorM.On("getDeviceInfo", mock.Anything, mock.Anything).Return(
		&deviceInfo{
			HardwareVendor:        "Apple",
			HardwareName:          "Macbook",
			DeviceType:            "device",
			PlatformVendor:        "Apple",
			PlatformName:          "MacOs",
			PlatformVersion:       "14",
			BrowserVendor:         "Google",
			BrowserName:           "Crome",
			BrowserVersion:        "12",
			ScreenPixelsWidth:     1024,
			ScreenPixelsHeight:    1080,
			PixelRatio:            223,
			Javascript:            true,
			GeoLocation:           true,
			HardwareFamily:        "Macbook",
			HardwareModel:         "Macbook",
			HardwareModelVariants: "Macbook",
			UserAgent:             "ua",
			DeviceId:              "",
		},
		nil,
	)

	module := Module{
		deviceDetector:    &deviceDetectorM,
		evidenceExtractor: &mockEvidenceExtractor,
		accountValidator:  &mockValidator,
	}

	mctx := make(hookstage.ModuleContext)
	mctx[ddEnabledCtxKey] = true

	result, err := module.HandleRawAuctionHook(
		nil, hookstage.ModuleInvocationContext{
			ModuleContext: mctx,
		},
		[]byte{},
	)
	assert.NoError(t, err)
	assert.Equal(t, len(result.ChangeSet.Mutations()), 1)
	assert.Equal(t, result.ChangeSet.Mutations()[0].Type(), hookstage.MutationUpdate)

	mutation := result.ChangeSet.Mutations()[0]

	body := []byte(`{
		"device": {
			"connectiontype": 2,
			"ext": {
				"atts": 0,
				"ifv": "1B8EFA09-FF8F-4123-B07F-7283B50B3870"
			},
			"sua": {
				"source": 2,
				"browsers": [
					{
						"brand": "Not A(Brand",
						"version": [
							"99",
							"0",
							"0",
							"0"
						]
					},
					{
						"brand": "Google Chrome",
						"version": [
							"121",
							"0",
							"6167",
							"184"
						]
					},
					{
						"brand": "Chromium",
						"version": [
							"121",
							"0",
							"6167",
							"184"
						]
					}
				],
				"platform": {
					"brand": "macOS",
					"version": [
						"14",
						"0",
						"0"
					]
				},
				"mobile": 0,
				"architecture": "arm",
				"model": ""
			}
		}
	}`)

	mutationResult, err := mutation.Apply(body)

	require.JSONEq(t, string(mutationResult), `{
		"device": {
			"connectiontype": 2,
			"ext": {
				"atts": 0,
				"ifv": "1B8EFA09-FF8F-4123-B07F-7283B50B3870",
				"fiftyonedegrees_deviceId":""
			},
			"sua": {
				"source": 2,
				"browsers": [
					{
						"brand": "Not A(Brand",
						"version": [
							"99",
							"0",
							"0",
							"0"
						]
					},
					{
						"brand": "Google Chrome",
						"version": [
							"121",
							"0",
							"6167",
							"184"
						]
					},
					{
						"brand": "Chromium",
						"version": [
							"121",
							"0",
							"6167",
							"184"
						]
					}
				],
				"platform": {
					"brand": "macOS",
					"version": [
						"14",
						"0",
						"0"
					]
				},
				"mobile": 0,
				"architecture": "arm",
				"model": ""
			}
		,"devicetype":2,"ua":"ua","make":"Apple","model":"Macbook","os":"MacOs","osv":"14","h":1080,"w":1024,"pxratio":223,"js":1,"geoFetch":1}
	}`)

	var deviceDetectorErrM mockDeviceDetector

	deviceDetectorErrM.On("getDeviceInfo", mock.Anything, mock.Anything).Return(
		nil,
		errors.New("error"),
	)

	module.deviceDetector = &deviceDetectorErrM

	result, err = module.HandleRawAuctionHook(
		nil, hookstage.ModuleInvocationContext{
			ModuleContext: mctx,
		},
		[]byte{},
	)

	assert.NoError(t, err)

	assert.Equal(t, len(result.ChangeSet.Mutations()), 1)

	assert.Equal(t, result.ChangeSet.Mutations()[0].Type(), hookstage.MutationUpdate)

	mutation = result.ChangeSet.Mutations()[0]

	_, err = mutation.Apply(body)
	assert.Errorf(t, err, "error getting device info")
}

func TestHandleRawAuctionHookEnrichmentWithErrors(t *testing.T) {
	var mockValidator mockAccValidator

	mockValidator.On("isAllowed", mock.Anything, mock.Anything).Return(true)

	var mockEvidenceExtractor mockEvidenceExtractor
	mockEvidenceExtractor.On("extract", mock.Anything).Return(
		[]onpremise.Evidence{
			{
				Key:   "key",
				Value: "val",
			},
		},
		"ua",
		nil,
	)

	var mockDeviceDetector mockDeviceDetector

	mockDeviceDetector.On("getDeviceInfo", mock.Anything, mock.Anything).Return(
		&deviceInfo{
			HardwareVendor:        "Apple",
			HardwareName:          "Macbook",
			DeviceType:            "device",
			PlatformVendor:        "Apple",
			PlatformName:          "MacOs",
			PlatformVersion:       "14",
			BrowserVendor:         "Google",
			BrowserName:           "Crome",
			BrowserVersion:        "12",
			ScreenPixelsWidth:     1024,
			ScreenPixelsHeight:    1080,
			PixelRatio:            223,
			Javascript:            true,
			GeoLocation:           true,
			HardwareFamily:        "Macbook",
			HardwareModel:         "Macbook",
			HardwareModelVariants: "Macbook",
			UserAgent:             "ua",
			DeviceId:              "",
			ScreenInchesHeight:    7,
		},
		nil,
	)

	module := Module{
		deviceDetector:    &mockDeviceDetector,
		evidenceExtractor: &mockEvidenceExtractor,
		accountValidator:  &mockValidator,
	}

	mctx := make(hookstage.ModuleContext)
	mctx[ddEnabledCtxKey] = true

	result, err := module.HandleRawAuctionHook(
		nil, hookstage.ModuleInvocationContext{
			ModuleContext: mctx,
		},
		[]byte{},
	)
	assert.NoError(t, err)
	assert.Equal(t, len(result.ChangeSet.Mutations()), 1)
	assert.Equal(t, result.ChangeSet.Mutations()[0].Type(), hookstage.MutationUpdate)

	mutation := result.ChangeSet.Mutations()[0]

	mutationResult, err := mutation.Apply(hookstage.RawAuctionRequestPayload(`{"device":{}}`))
	assert.NoError(t, err)
	require.JSONEq(t, string(mutationResult), `{"device":{"devicetype":2,"ua":"ua","make":"Apple","model":"Macbook","os":"MacOs","osv":"14","h":1080,"w":1024,"pxratio":223,"js":1,"geoFetch":1,"ppi":154,"ext":{"fiftyonedegrees_deviceId":""}}}`)
}

func TestConfigHashFromConfig(t *testing.T) {
	cfg := config{
		Performance: performance{
			Profile:        "",
			Concurrency:    nil,
			Difference:     nil,
			AllowUnmatched: nil,
			Drift:          nil,
		},
	}

	result := configHashFromConfig(&cfg)
	assert.Equal(t, result.PerformanceProfile(), dd.Default)
	assert.Equal(t, result.Concurrency(), uint16(0xa))
	assert.Equal(t, result.Difference(), int32(0))
	assert.Equal(t, result.AllowUnmatched(), false)
	assert.Equal(t, result.Drift(), int32(0))

	concurrency := 1
	difference := 1
	allowUnmatched := true
	drift := 1

	cfg = config{
		Performance: performance{
			Profile:        "Balanced",
			Concurrency:    &concurrency,
			Difference:     &difference,
			AllowUnmatched: &allowUnmatched,
			Drift:          &drift,
		},
	}

	result = configHashFromConfig(&cfg)
	assert.Equal(t, result.PerformanceProfile(), dd.Balanced)
	assert.Equal(t, result.Concurrency(), uint16(1))
	assert.Equal(t, result.Difference(), int32(1))
	assert.Equal(t, result.AllowUnmatched(), true)
	assert.Equal(t, result.Drift(), int32(1))

	cfg = config{
		Performance: performance{
			Profile: "InMemory",
		},
	}
	result = configHashFromConfig(&cfg)
	assert.Equal(t, result.PerformanceProfile(), dd.InMemory)

	cfg = config{
		Performance: performance{
			Profile: "HighPerformance",
		},
	}
	result = configHashFromConfig(&cfg)
	assert.Equal(t, result.PerformanceProfile(), dd.HighPerformance)
}

func TestSignDeviceData(t *testing.T) {
	devicePld := map[string]any{
		"ext": map[string]any{
			"my-key": "my-value",
		},
	}

	deviceInfo := deviceInfo{
		DeviceId: "test-device-id",
	}

	result := signDeviceData(devicePld, &deviceInfo)
	r, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	require.JSONEq(
		t,
		`{"ext":{"fiftyonedegrees_deviceId":"test-device-id","my-key":"my-value"}}`,
		string(r),
	)
}

func TestBuilderWithInvalidJson(t *testing.T) {
	_, err := Builder([]byte(`{`), moduledeps.ModuleDeps{})
	assert.Error(t, err)
	assert.Errorf(t, err, "failed to parse config")
}

func TestBuilderWithInvalidConfig(t *testing.T) {
	_, err := Builder([]byte(`{"data_file":{}}`), moduledeps.ModuleDeps{})
	assert.Error(t, err)
	assert.Errorf(t, err, "invalid config")
}

func TestBuilderHandleDeviceDetectorError(t *testing.T) {
	var mockConfig config
	mockConfig.Performance.Profile = "default"
	testFile, _ := os.Create("test-builder-config.hash")
	defer testFile.Close()
	defer os.Remove("test-builder-config.hash")

	_, err := Builder(
		[]byte(`{
			"enabled": true,
			"data_file": {
				"path": "test-builder-config.hash",
				"update": {
					"auto": true,
					"url": "https://my.datafile.com/datafile.gz",
					"polling_interval": 3600,
					"licence_key": "your_licence_key",
					"product": "V4Enterprise"
            	}
          	},
			"account_filter": {"allow_list": ["123"]},
			"performance": {
				"profile": "123",
				"concurrency": 1,
				"difference": 1,
				"allow_unmatched": true,
				"drift": 1	
			}
		}`), moduledeps.ModuleDeps{},
	)
	assert.Error(t, err)
	assert.Errorf(t, err, "failed to create device detector")
}

func TestHydrateFields(t *testing.T) {
	deviceInfo := &deviceInfo{
		HardwareVendor:        "Apple",
		HardwareName:          "Macbook",
		DeviceType:            "device",
		PlatformVendor:        "Apple",
		PlatformName:          "MacOs",
		PlatformVersion:       "14",
		BrowserVendor:         "Google",
		BrowserName:           "Crome",
		BrowserVersion:        "12",
		ScreenPixelsWidth:     1024,
		ScreenPixelsHeight:    1080,
		PixelRatio:            223,
		Javascript:            true,
		GeoLocation:           true,
		HardwareFamily:        "Macbook",
		HardwareModel:         "Macbook",
		HardwareModelVariants: "Macbook",
		UserAgent:             "ua",
		DeviceId:              "dev-ide",
	}

	rawPld := `{
		"imp": [{
			"id": "",
			"banner": {
				"topframe": 1,
				"format": [
					{
						"w": 728,
						"h": 90
					}
				],
				"pos": 1
			},
			"bidfloor": 0.01,
			"bidfloorcur": "USD"
		}],
		"device": {
			"model": "Macintosh",
			"w": 843,
			"h": 901,
			"dnt": 0,
			"ua": "Mozilla/5.0 (Linux; Android 13; SAMSUNG SM-A037U) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36",
			"language": "en",
			"sua": {"browsers":[{"brand":"Not/A)Brand","version":["99","0","0","0"]},{"brand":"Samsung Internet","version":["23","0","1","1"]},{"brand":"Chromium","version":["115","0","5790","168"]}],"platform":{"brand":"Android","version":["13","0","0"]},"mobile":1,"model":"SM-A037U","source":2},
			"ext": {"h":"901","w":843}
		},
		"cur": [
			"USD"
		],
		"tmax": 1700
	}`

	payload, err := hydrateFields(deviceInfo, []byte(rawPld))
	assert.NoError(t, err)

	var deviceHolder struct {
		Device json.RawMessage `json:"device"`
	}

	err = json.Unmarshal(payload, &deviceHolder)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}

	require.JSONEq(
		t,
		`{"devicetype":2,"dnt":0,"ext":{"fiftyonedegrees_deviceId":"dev-ide","h":"901","w":843},"geoFetch":1,"h":901,"js":1,"language":"en","make":"Apple","model":"Macintosh","os":"MacOs","osv":"14","pxratio":223,"sua":{"browsers":[{"brand":"Not/A)Brand","version":["99","0","0","0"]},{"brand":"Samsung Internet","version":["23","0","1","1"]},{"brand":"Chromium","version":["115","0","5790","168"]}],"mobile":1,"model":"SM-A037U","platform":{"brand":"Android","version":["13","0","0"]},"source":2},"ua":"Mozilla/5.0 (Linux; Android 13; SAMSUNG SM-A037U) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/23.0 Chrome/115.0.0.0 Mobile Safari/537.36","w":843}`,
		string(deviceHolder.Device),
	)
}
