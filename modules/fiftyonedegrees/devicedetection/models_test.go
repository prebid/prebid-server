package devicedetection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEvidenceByKey(t *testing.T) {
	populatedEvidence := []stringEvidence{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}

	tests := []struct {
		name           string
		evidence       []stringEvidence
		key            string
		expectEvidence stringEvidence
		expectFound    bool
	}{
		{
			name:           "nil_evidence",
			evidence:       nil,
			key:            "key2",
			expectEvidence: stringEvidence{},
			expectFound:    false,
		},
		{
			name:           "empty_evidence",
			evidence:       []stringEvidence{},
			key:            "key2",
			expectEvidence: stringEvidence{},
			expectFound:    false,
		},
		{
			name:     "key_found",
			evidence: populatedEvidence,
			key:      "key2",
			expectEvidence: stringEvidence{
				Key:   "key2",
				Value: "value2",
			},
			expectFound: true,
		},
		{
			name:           "key_not_found",
			evidence:       populatedEvidence,
			key:            "key4",
			expectEvidence: stringEvidence{},
			expectFound:    false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, exists := getEvidenceByKey(test.evidence, test.key)
			assert.Equal(t, test.expectFound, exists)
			assert.Equal(t, test.expectEvidence, result)
		})
	}
}
