package maputil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadEmbeddedMap(t *testing.T) {
	testCases := []struct {
		value       map[string]interface{}
		key         string
		expectedMap map[string]interface{}
		expectedOK  bool
	}{
		{
			value:       nil,
			key:         "",
			expectedMap: nil,
			expectedOK:  false,
		},
	}

	for _, test := range testCases {
		resultMap, resultOK := ReadEmbeddedMap(test.value, test.key)

		assert.Equal(t, test.expectedMap, resultMap)
		assert.Equal(t, test.expectedOK, resultOK)
	}
}
