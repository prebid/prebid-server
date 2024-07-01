package device_detection

import (
	"net/http"
	"testing"

	dd "github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestEvidenceExtractorStringsFromHeaders(t *testing.T) {
	extractor := NewEvidenceExtractor()

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

	evidence := extractor.FromHeaders(&req, evidenceKeys)

	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, evidence[0].Value, "Value")
	assert.Equal(t, evidence[0].Key, "header")
	assert.Equal(t, evidence[1].Value, "Chrome;12")
	assert.Equal(t, evidence[1].Key, "Sec-CH-UA-Full-Version-List")
}

func TestEvidenceExtractorStringsFromSUATag(t *testing.T) {
	extractor := NewEvidenceExtractor()

	req := http.Request{
		Header: make(map[string][]string),
	}
	payload := []byte(`{
		"device": {
			"connectiontype": 2,
			"ext": {
				"atts": 0,
				"ifv": "1B8EFA09-FF8F-4123-B07F-7283B50B3870"
			},
			"h": 852,
			"ifa": "00000000-0000-0000-0000-000000000000",
			"language": "en",
			"lmt": 1,
			"make": "Apple",
			"model": "iPhone",
			"os": "iOS",
			"osv": "17.0",
			"pxratio": 3,
			"ua": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
			"w": 393,
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

	evidence := extractor.FromSuaPayload(&req, payload)

	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, evidence[1].Value, "arm")
	assert.Equal(t, evidence[1].Key, "Sec-Ch-Ua-Arch")
}

func TestEvidenceExtractorUAFromHeaders(t *testing.T) {
	extractor := NewEvidenceExtractor()

	req := http.Request{
		Header: make(map[string][]string),
	}
	payload := []byte(`{
		"device": {
			"connectiontype": 2,
			"ext": {
				"atts": 0,
				"ifv": "1B8EFA09-FF8F-4123-B07F-7283B50B3870"
			},
			"h": 852,
			"ifa": "00000000-0000-0000-0000-000000000000",
			"language": "en",
			"lmt": 1,
			"make": "Apple",
			"model": "iPhone",
			"os": "iOS",
			"osv": "17.0",
			"pxratio": 3,
			"w": 393,
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

	evidence := extractor.FromSuaPayload(&req, payload)

	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, evidence[0].Value, "arm")
	assert.Equal(t, evidence[0].Key, "Sec-Ch-Ua-Arch")
}

func TestEvidenceExtractorEvidenceFromSUATag(t *testing.T) {
	extractor := NewEvidenceExtractor()

	ctx := hookstage.ModuleContext{
		EvidenceFromSuaCtxKey: []StringEvidence{
			{
				Prefix: "sua",
				Key:    "Sec-Ch-Ua-Full-Version-List",
				Value:  "arm",
			},
			{
				Prefix: "sua",
				Key:    "User-Agent",
				Value:  "ua",
			},
		},
	}

	evidence, userAgent, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, len(evidence), 2)
	assert.Equal(t, userAgent, "ua")
}

func TestEvidenceExtractorEvidenceFromHeaders(t *testing.T) {
	extractor := NewEvidenceExtractor()

	ctx := hookstage.ModuleContext{
		EvidenceFromHeadersCtxKey: []StringEvidence{
			{
				Prefix: QueryPrefix,
				Key:    SecUaFullVersionList,
				Value:  "Chrome;14",
			},
			{
				Prefix: "sua",
				Key:    SecChUaArch,
				Value:  "arm",
			},
			{
				Prefix: "sua",
				Key:    "User-Agent",
				Value:  "ua",
			},
		},
	}

	evidence, userAgent, err := extractor.Extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, len(evidence), 3)
	assert.Equal(t, userAgent, "ua")
}

func TestEvidenceExtractorEmptyEvidence(t *testing.T) {
	extractor := NewEvidenceExtractor()

	evidence, userAgent, err := extractor.Extract(nil)

	assert.Error(t, err)
	assert.Nil(t, evidence)
	assert.Equal(t, userAgent, "")
}

func TestEvidenceExtractorBadEvidence(t *testing.T) {
	_, err := NewEvidenceExtractor().getEvidenceStrings("123")
	assert.Error(t, err)
}

func TestExtractBadContext(t *testing.T) {
	extractor := NewEvidenceExtractor()

	cases := []struct {
		ctx hookstage.ModuleContext
	}{
		{
			ctx: hookstage.ModuleContext{
				EvidenceFromHeadersCtxKey: "bad value",
			},
		},
		{
			ctx: hookstage.ModuleContext{
				EvidenceFromSuaCtxKey:     []StringEvidence{},
				EvidenceFromHeadersCtxKey: "bad value",
			},
		},
		{
			ctx: hookstage.ModuleContext{
				EvidenceFromSuaCtxKey: "bad value",
			},
		},
	}

	for _, s := range cases {
		_, _, err := extractor.Extract(s.ctx)

		assert.Error(t, err)
	}

}
