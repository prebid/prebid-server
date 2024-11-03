package devicedetection

import (
	"net/http"
	"testing"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestFromHeaders(t *testing.T) {
	extractor := newEvidenceExtractor()

	req := http.Request{
		Header: make(map[string][]string),
	}
	req.Header.Add("header", "Value")
	req.Header.Add("Sec-CH-UA-Full-Version-List", "Chrome;12")
	evidenceKeys := []dd.EvidenceKey{
		{
			Prefix: dd.EvidencePrefix(10),
			Key:    "header",
		},
		{
			Prefix: dd.EvidencePrefix(10),
			Key:    "Sec-CH-UA-Full-Version-List",
		},
	}

	evidence := extractor.fromHeaders(&req, evidenceKeys)

	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, evidence[0].Value, "Value")
	assert.Equal(t, evidence[0].Key, "header")
	assert.Equal(t, evidence[1].Value, "Chrome;12")
	assert.Equal(t, evidence[1].Key, "Sec-CH-UA-Full-Version-List")
}

func TestFromSuaPayload(t *testing.T) {
	tests := []struct {
		name             string
		payload          []byte
		evidenceSize     int
		evidenceKeyOrder int
		expectedKey      string
		expectedValue    string
	}{
		{
			name: "from_SUA_tag",
			payload: []byte(`{
				"device": {
					"sua": {
						"browsers": [
							{
								"brand": "Google Chrome",
								"version": ["121", "0", "6167", "184"]
							}
						],
						"platform": {
							"brand": "macOS",
							"version": ["14", "0", "0"]
						},
						"architecture": "arm"
					}
				}
			}`),
			evidenceSize:     4,
			evidenceKeyOrder: 0,
			expectedKey:      "Sec-Ch-Ua-Arch",
			expectedValue:    "arm",
		},
		{
			name: "from_UA_headers",
			payload: []byte(`{
				"device": {
					"ua": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
					"sua": {
						"architecture": "arm"
					}
				}
			}`),
			evidenceSize:     2,
			evidenceKeyOrder: 1,
			expectedKey:      "Sec-Ch-Ua-Arch",
			expectedValue:    "arm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := newEvidenceExtractor()

			evidence := extractor.fromSuaPayload(tt.payload)

			assert.NotNil(t, evidence)
			assert.NotEmpty(t, evidence)
			assert.Equal(t, len(evidence), tt.evidenceSize)
			assert.Equal(t, evidence[tt.evidenceKeyOrder].Key, tt.expectedKey)
			assert.Equal(t, evidence[tt.evidenceKeyOrder].Value, tt.expectedValue)
		})
	}
}

func TestExtract(t *testing.T) {
	uaEvidence1 := stringEvidence{
		Prefix: "ua1",
		Key:    userAgentHeader,
		Value:  "uav1",
	}
	uaEvidence2 := stringEvidence{
		Prefix: "ua2",
		Key:    userAgentHeader,
		Value:  "uav2",
	}
	evidence1 := stringEvidence{
		Prefix: "e1",
		Key:    "k1",
		Value:  "v1",
	}
	emptyEvidence := stringEvidence{
		Prefix: "empty",
		Key:    "e1",
		Value:  "",
	}

	tests := []struct {
		name              string
		ctx               hookstage.ModuleContext
		wantEvidenceCount int
		wantUserAgent     string
		wantError         bool
	}{
		{
			name:      "nil",
			ctx:       nil,
			wantError: true,
		},
		{
			name: "empty",
			ctx: hookstage.ModuleContext{
				evidenceFromSuaCtxKey:     []stringEvidence{},
				evidenceFromHeadersCtxKey: []stringEvidence{},
			},
			wantEvidenceCount: 0,
			wantUserAgent:     "",
		},
		{
			name: "from_headers",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{uaEvidence1},
			},
			wantEvidenceCount: 1,
			wantUserAgent:     "uav1",
		},
		{
			name: "from_headers_no_user_agent",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{evidence1},
			},
			wantError: true,
		},
		{
			name: "from_sua",
			ctx: hookstage.ModuleContext{
				evidenceFromSuaCtxKey: []stringEvidence{uaEvidence1},
			},
			wantEvidenceCount: 1,
			wantUserAgent:     "uav1",
		},
		{
			name: "from_sua_no_user_agent",
			ctx: hookstage.ModuleContext{
				evidenceFromSuaCtxKey: []stringEvidence{evidence1},
			},
			wantError: true,
		},
		{
			name: "from_headers_error",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: "bad value",
			},
			wantError: true,
		},
		{
			name: "from_sua_error",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{},
				evidenceFromSuaCtxKey:     "bad value",
			},
			wantError: true,
		},
		{
			name: "from_sua_and_headers",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{uaEvidence1},
				evidenceFromSuaCtxKey:     []stringEvidence{evidence1},
			},
			wantEvidenceCount: 2,
			wantUserAgent:     "uav1",
		},
		{
			name: "from_sua_and_headers_sua_can_overwrite_if_ua_present",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{uaEvidence1},
				evidenceFromSuaCtxKey:     []stringEvidence{uaEvidence2},
			},
			wantEvidenceCount: 1,
			wantUserAgent:     "uav2",
		},
		{
			name: "empty_string_values",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{emptyEvidence},
			},
			wantError: true,
		},
		{
			name: "empty_sua_values",
			ctx: hookstage.ModuleContext{
				evidenceFromSuaCtxKey: []stringEvidence{emptyEvidence},
			},
			wantError: true,
		},
		{
			name: "mixed_valid_and_invalid",
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: []stringEvidence{uaEvidence1},
				evidenceFromSuaCtxKey:     "bad value",
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			extractor := newEvidenceExtractor()
			evidence, userAgent, err := extractor.extract(test.ctx)

			if test.wantError {
				assert.Error(t, err)
				assert.Nil(t, evidence)
				assert.Equal(t, userAgent, "")
			} else if test.wantEvidenceCount == 0 {
				assert.NoError(t, err)
				assert.Nil(t, evidence)
				assert.Equal(t, userAgent, "")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, len(evidence), test.wantEvidenceCount)
				assert.Equal(t, userAgent, test.wantUserAgent)
			}
		})
	}
}
