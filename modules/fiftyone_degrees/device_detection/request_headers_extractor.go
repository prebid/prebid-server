package device_detection

import (
	"net/http"
	"strings"

	dd "github.com/51Degrees/device-detection-go/v4/dd"
)

// EvidenceFromRequestHeadersExtractor is a struct that extracts evidence from http request headers
type EvidenceFromRequestHeadersExtractor struct{}

func NewEvidenceFromRequestHeadersExtractor() *EvidenceFromRequestHeadersExtractor {
	return &EvidenceFromRequestHeadersExtractor{}
}

func (x EvidenceFromRequestHeadersExtractor) Extract(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []StringEvidence {
	return x.extractEvidenceStrings(request, httpHeaderKeys)
}

func (x EvidenceFromRequestHeadersExtractor) extractEvidenceStrings(r *http.Request, keys []dd.EvidenceKey) []StringEvidence {
	evidence := make([]StringEvidence, 0)
	for _, e := range keys {
		lowerKey := strings.ToLower(e.Key)
		if e.Prefix != dd.HttpEvidenceQuery {
			// Get evidence from headers
			headerVal := r.Header.Get(lowerKey)
			if headerVal != "" {
				if e.Key != SecUaFullVersionList && e.Key != SecChUa {
					headerVal = strings.Replace(headerVal, "\"", "", -1)
					if headerVal != "" {
						evidence = append(
							evidence,
							StringEvidence{
								Prefix: HeaderPrefix,
								Key:    e.Key,
								Value:  headerVal,
							},
						)
					}
				} else {
					evidence = append(
						evidence,
						StringEvidence{
							Prefix: HeaderPrefix,
							Key:    e.Key,
							Value:  headerVal,
						},
					)
				}
			}
		}
	}
	return evidence
}
