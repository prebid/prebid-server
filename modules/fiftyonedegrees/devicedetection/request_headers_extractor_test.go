package devicedetection

import (
	"net/http"
	"testing"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/stretchr/testify/assert"
)

func TestExtractEvidenceStrings(t *testing.T) {
	tests := []struct {
		name             string
		headers          map[string]string
		keys             []dd.EvidenceKey
		expectedEvidence []stringEvidence
	}{
		{
			name: "Ignored query evidence",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0",
			},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpEvidenceQuery, Key: "User-Agent"},
			},
			expectedEvidence: []stringEvidence{},
		},
		{
			name:    "Empty headers",
			headers: map[string]string{},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpHeaderString, Key: "User-Agent"},
			},
			expectedEvidence: []stringEvidence{},
		},
		{
			name: "Single header",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0",
			},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpHeaderString, Key: "User-Agent"},
			},
			expectedEvidence: []stringEvidence{
				{Prefix: headerPrefix, Key: "User-Agent", Value: "Mozilla/5.0"},
			},
		},
		{
			name: "Multiple headers",
			headers: map[string]string{
				"User-Agent": "Mozilla/5.0",
				"Accept":     "text/html",
			},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpHeaderString, Key: "User-Agent"},
				{Prefix: dd.HttpEvidenceQuery, Key: "Query"},
				{Prefix: dd.HttpHeaderString, Key: "Accept"},
			},
			expectedEvidence: []stringEvidence{
				{Prefix: headerPrefix, Key: "User-Agent", Value: "Mozilla/5.0"},
				{Prefix: headerPrefix, Key: "Accept", Value: "text/html"},
			},
		},
		{
			name: "Header with quotes removed",
			headers: map[string]string{
				"IP-List": "\"92.0.4515.159\"",
			},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpHeaderString, Key: "IP-List"},
			},
			expectedEvidence: []stringEvidence{
				{Prefix: headerPrefix, Key: "IP-List", Value: "92.0.4515.159"},
			},
		},
		{
			name: "Sec-CH-UA headers with quotes left",
			headers: map[string]string{
				"Sec-CH-UA": "\"Chromium\";v=\"92\", \"Google Chrome\";v=\"92\"",
			},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpHeaderString, Key: secChUa},
			},
			expectedEvidence: []stringEvidence{
				{Prefix: headerPrefix, Key: secChUa, Value: "\"Chromium\";v=\"92\", \"Google Chrome\";v=\"92\""},
			},
		},
		{
			name: "Sec-CH-UA-Full-Version-List headers with quotes left",
			headers: map[string]string{
				"Sec-CH-UA-Full-Version-List": "\"92.0.4515.159\"",
			},
			keys: []dd.EvidenceKey{
				{Prefix: dd.HttpHeaderString, Key: secUaFullVersionList},
			},
			expectedEvidence: []stringEvidence{
				{Prefix: headerPrefix, Key: secUaFullVersionList, Value: "\"92.0.4515.159\""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := http.Request{
				Header: make(map[string][]string),
			}

			for key, value := range test.headers {
				req.Header.Set(key, value)
			}

			extractor := newEvidenceFromRequestHeadersExtractor()
			evidence := extractor.extractEvidenceStrings(&req, test.keys)

			assert.Equal(t, test.expectedEvidence, evidence)
		})
	}
}
