package device_detection

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEvidenceByKey(t *testing.T) {
	// Tests for EvidenceByKey
	evidence := []StringEvidence{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}

	result, exists := GetEvidenceByKey(evidence, "key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", result.Value)

	result, exists = GetEvidenceByKey(evidence, "key4")
	assert.False(t, exists)
	assert.Equal(t, "", result.Value)

}
