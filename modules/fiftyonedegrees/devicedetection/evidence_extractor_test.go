package devicedetection

import (
	"net/http"
	"testing"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
	"github.com/stretchr/testify/assert"
)

func TestEvidenceExtractorStringsFromHeaders(t *testing.T) {
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

func TestEvidenceExtractorStringsFromSUATag(t *testing.T) {
	extractor := newEvidenceExtractor()

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

	evidence := extractor.fromSuaPayload(payload)

	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, evidence[1].Value, "arm")
	assert.Equal(t, evidence[1].Key, "Sec-Ch-Ua-Arch")
}

func TestEvidenceExtractorUAFromHeaders(t *testing.T) {
	extractor := newEvidenceExtractor()

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

	evidence := extractor.fromSuaPayload(payload)

	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, evidence[0].Value, "arm")
	assert.Equal(t, evidence[0].Key, "Sec-Ch-Ua-Arch")
}

func TestEvidenceExtractorEvidenceFromSUATag(t *testing.T) {
	extractor := newEvidenceExtractor()

	ctx := hookstage.ModuleContext{
		evidenceFromSuaCtxKey: []stringEvidence{
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

	evidence, userAgent, err := extractor.extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, len(evidence), 2)
	assert.Equal(t, userAgent, "ua")
}

func TestEvidenceExtractorEvidenceFromHeaders(t *testing.T) {
	extractor := newEvidenceExtractor()

	ctx := hookstage.ModuleContext{
		evidenceFromHeadersCtxKey: []stringEvidence{
			{
				Prefix: queryPrefix,
				Key:    secUaFullVersionList,
				Value:  "Chrome;14",
			},
			{
				Prefix: "sua",
				Key:    secChUaArch,
				Value:  "arm",
			},
			{
				Prefix: "sua",
				Key:    "User-Agent",
				Value:  "ua",
			},
		},
	}

	evidence, userAgent, err := extractor.extract(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, evidence)
	assert.NotEmpty(t, evidence)
	assert.Equal(t, len(evidence), 3)
	assert.Equal(t, userAgent, "ua")
}

func TestEvidenceExtractorEmptyEvidence(t *testing.T) {
	extractor := newEvidenceExtractor()

	evidence, userAgent, err := extractor.extract(nil)

	assert.Error(t, err)
	assert.Nil(t, evidence)
	assert.Equal(t, userAgent, "")
}

func TestEvidenceExtractorBadEvidence(t *testing.T) {
	_, err := newEvidenceExtractor().getEvidenceStrings("123")
	assert.Error(t, err)
}

func TestExtractBadContext(t *testing.T) {
	extractor := newEvidenceExtractor()

	cases := []struct {
		ctx hookstage.ModuleContext
	}{
		{
			ctx: hookstage.ModuleContext{
				evidenceFromHeadersCtxKey: "bad value",
			},
		},
		{
			ctx: hookstage.ModuleContext{
				evidenceFromSuaCtxKey:     []stringEvidence{},
				evidenceFromHeadersCtxKey: "bad value",
			},
		},
		{
			ctx: hookstage.ModuleContext{
				evidenceFromSuaCtxKey: "bad value",
			},
		},
	}

	for _, s := range cases {
		_, _, err := extractor.extract(s.ctx)

		assert.Error(t, err)
	}

}
