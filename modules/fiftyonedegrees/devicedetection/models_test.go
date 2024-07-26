package devicedetection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvidenceByKey(t *testing.T) {
	// Tests for EvidenceByKey
	evidence := []stringEvidence{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}

	result, exists := getEvidenceByKey(evidence, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", result.Value)

	result, exists = getEvidenceByKey(evidence, "key4")
	assert.False(t, exists)
	assert.Equal(t, "", result.Value)

}
