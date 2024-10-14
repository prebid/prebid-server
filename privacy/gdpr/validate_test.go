package gdpr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConsent(t *testing.T) {
	testCases := []struct {
		description string
		consent     string
		expected    bool
	}{
		{
			description: "Invalid",
			consent:     "<any invalid>",
			expected:    false,
		},
		{
			description: "TCF2 Valid",
			consent:     "COzTVhaOzTVhaGvAAAENAiCIAP_AAH_AAAAAAEEUACCKAAA",
			expected:    true,
		},
	}

	for _, test := range testCases {
		result := ValidateConsent(test.consent)
		assert.Equal(t, test.expected, result, test.description)
	}
}
