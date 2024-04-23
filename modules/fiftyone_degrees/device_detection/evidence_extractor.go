package device_detection

import (
	"github.com/51Degrees/device-detection-go/v4/onpremise"
	"github.com/pkg/errors"
	"net/http"

	dd "github.com/51Degrees/device-detection-go/v4/dd"
	"github.com/prebid/prebid-server/v2/hooks/hookstage"
)

type EvidenceExtractor struct {
	fromHeaders *EvidenceFromRequestHeadersExtractor
	fromSUA     *EvidenceFromSUAPayloadExtractor
}

func NewEvidenceExtractor() *EvidenceExtractor {
	evidenceExtractor := &EvidenceExtractor{
		fromHeaders: NewEvidenceFromRequestHeadersExtractor(),
		fromSUA:     NewEvidenceFromSUAPayloadExtractor(),
	}

	return evidenceExtractor
}

func (x *EvidenceExtractor) FromHeaders(request *http.Request, httpHeaderKeys []dd.EvidenceKey) []StringEvidence {
	return x.fromHeaders.Extract(request, httpHeaderKeys)
}

func (x *EvidenceExtractor) FromSuaPayload(request *http.Request, payload []byte) []StringEvidence {
	return x.fromSUA.Extract(request, payload)
}

// merge merges two slices of StringEvidence into one slice of StringEvidence
func merge(val1, val2 []StringEvidence) []StringEvidence {
	evidenceMap := make(map[string]StringEvidence)
	for _, e := range val1 {
		evidenceMap[e.Key] = e
	}

	for _, e := range val2 {
		_, exists := evidenceMap[e.Key]
		if !exists {
			evidenceMap[e.Key] = e
		}
	}

	evidence := make([]StringEvidence, 0)

	for _, e := range evidenceMap {
		evidence = append(evidence, e)
	}

	return evidence
}

func (x *EvidenceExtractor) Extract(ctx hookstage.ModuleContext) ([]onpremise.Evidence, string, error) {
	if ctx == nil {
		return nil, "", errors.New("context is nil")
	}

	suaStrings, err := x.getEvidenceStrings(ctx[EvidenceFromSuaCtxKey])
	if err != nil {
		return nil, "", errors.Wrap(err, "error extracting evidence")
	}
	headerString, err := x.getEvidenceStrings(ctx[EvidenceFromHeadersCtxKey])
	if err != nil {
		return nil, "", errors.Wrap(err, "error extracting evidence")
	}

	// Merge evidence from headers and SUA, sua has higher priority
	evidenceStrings := merge(suaStrings, headerString)

	if evidenceStrings != nil && len(evidenceStrings) > 0 {
		userAgentE, exists := GetEvidenceByKey(evidenceStrings, UserAgentHeader)
		if !exists {
			return nil, "", errors.Wrap(err, "User-Agent not found")
		}

		evidence := x.extractEvidenceFromStrings(evidenceStrings)

		return evidence, userAgentE.Value, nil
	}

	return nil, "", nil
}

func (x *EvidenceExtractor) getEvidenceStrings(source interface{}) ([]StringEvidence, error) {
	if source == nil {
		return []StringEvidence{}, nil
	}

	evidenceStrings, ok := source.([]StringEvidence)
	if !ok {
		return nil, errors.New("bad cast to []StringEvidence")
	}

	return evidenceStrings, nil
}

func (x *EvidenceExtractor) extractEvidenceFromStrings(strEvidence []StringEvidence) []onpremise.Evidence {
	evidenceResult := make([]onpremise.Evidence, len(strEvidence))
	for i, e := range strEvidence {
		prefix := dd.HttpHeaderString
		if e.Prefix == QueryPrefix {
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
