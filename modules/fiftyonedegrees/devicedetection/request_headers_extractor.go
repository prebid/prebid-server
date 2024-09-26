package devicedetection

import (
	"net/http"
	"strings"

	"github.com/51Degrees/device-detection-go/v4/dd"
)

// evidenceFromRequestHeadersExtractor is a struct that extracts evidence from http request headers
type evidenceFromRequestHeadersExtractor struct{}

func newEvidenceFromRequestHeadersExtractor() evidenceFromRequestHeadersExtractor {
	return evidenceFromRequestHeadersExtractor{}
}

func (x evidenceFromRequestHeadersExtractor) extract(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []stringEvidence {
	return x.extractEvidenceStrings(request, httpHeaderKeys)
}

func (x evidenceFromRequestHeadersExtractor) extractEvidenceStrings(r *http.Request, keys []dd.EvidenceKey) []stringEvidence {
	evidence := make([]stringEvidence, 0)
	for _, e := range keys {
		if e.Prefix == dd.HttpEvidenceQuery {
			continue
		}

		// Get evidence from headers
		headerVal := r.Header.Get(e.Key)
		if headerVal == "" {
			continue
		}

		if e.Key != secUaFullVersionList && e.Key != secChUa {
			headerVal = strings.Replace(headerVal, "\"", "", -1)
		}

		if headerVal != "" {
			evidence = append(evidence, stringEvidence{
				Prefix: headerPrefix,
				Key:    e.Key,
				Value:  headerVal,
			})
		}
	}
	return evidence
}
