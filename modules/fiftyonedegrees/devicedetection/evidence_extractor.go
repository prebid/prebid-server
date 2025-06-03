package devicedetection

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
)

type defaultEvidenceExtractor struct {
	valFromHeaders evidenceFromRequestHeadersExtractor
	valFromSUA     evidenceFromSUAPayloadExtractor
}

func newEvidenceExtractor() *defaultEvidenceExtractor {
	evidenceExtractor := &defaultEvidenceExtractor{
		valFromHeaders: newEvidenceFromRequestHeadersExtractor(),
		valFromSUA:     newEvidenceFromSUAPayloadExtractor(),
	}

	return evidenceExtractor
}

func (x *defaultEvidenceExtractor) fromHeaders(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []stringEvidence {
	return x.valFromHeaders.extract(request, httpHeaderKeys)
}

func (x *defaultEvidenceExtractor) fromSuaPayload(payload []byte) []stringEvidence {
	return x.valFromSUA.extract(payload)
}

// merge merges two slices of stringEvidence into one slice of stringEvidence
func merge(val1, val2 []stringEvidence) []stringEvidence {
	evidenceMap := make(map[string]stringEvidence)
	for _, e := range val1 {
		evidenceMap[e.Key] = e
	}

	for _, e := range val2 {
		_, exists := evidenceMap[e.Key]
		if !exists {
			evidenceMap[e.Key] = e
		}
	}

	evidence := make([]stringEvidence, 0)

	for _, e := range evidenceMap {
		evidence = append(evidence, e)
	}

	return evidence
}

func (x *defaultEvidenceExtractor) extract(ctx hookstage.ModuleContext) ([]onpremise.Evidence, string, error) {
	if ctx == nil {
		return nil, "", errors.New("context is nil")
	}

	suaStrings, err := x.getEvidenceStrings(ctx[evidenceFromSuaCtxKey])
	if err != nil {
		return nil, "", fmt.Errorf("error extracting sua evidence: %w", err)
	}
	headerString, err := x.getEvidenceStrings(ctx[evidenceFromHeadersCtxKey])
	if err != nil {
		return nil, "", fmt.Errorf("error extracting header evidence: %w", err)
	}

	// Merge evidence from headers and SUA, sua has higher priority
	evidenceStrings := merge(suaStrings, headerString)

	if len(evidenceStrings) > 0 {
		userAgentE, exists := getEvidenceByKey(evidenceStrings, userAgentHeader)
		if !exists {
			return nil, "", errors.New("User-Agent not found")
		}

		evidence := x.extractEvidenceFromStrings(evidenceStrings)

		return evidence, userAgentE.Value, nil
	}

	return nil, "", nil
}

func (x *defaultEvidenceExtractor) getEvidenceStrings(source interface{}) ([]stringEvidence, error) {
	if source == nil {
		return []stringEvidence{}, nil
	}

	evidenceStrings, ok := source.([]stringEvidence)
	if !ok {
		return nil, errors.New("bad cast to []stringEvidence")
	}

	return evidenceStrings, nil
}

func (x *defaultEvidenceExtractor) extractEvidenceFromStrings(strEvidence []stringEvidence) []onpremise.Evidence {
	evidenceResult := make([]onpremise.Evidence, len(strEvidence))
	for i, e := range strEvidence {
		prefix := dd.HttpHeaderString
		if e.Prefix == queryPrefix {
			prefix = dd.HttpEvidenceQuery
		}

		evidenceResult[i] = onpremise.Evidence{
			Prefix: prefix,
			Key:    e.Key,
			Value:  e.Value,
		}
	}

	return evidenceResult
}
