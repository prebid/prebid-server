package devicedetection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	siteRequestPayload = []byte(`
		{
			"site": {
				"publisher": {
					"id": "p-bid-config-test-005"
				}
			}
		}
	`)

	mobileRequestPayload = []byte(`
		{
			"app": {
				"publisher": {
					"id": "p-bid-config-test-005"
				}
			}
		}
	`)

	emptyPayload = []byte(`{}`)
)

func TestPublisherIdExtraction(t *testing.T) {
	tests := []struct {
		name      string
		payload   []byte
		expected  string
		expectNil bool
	}{
		{
			name:     "SiteRequest",
			payload:  siteRequestPayload,
			expected: "p-bid-config-test-005",
		},
		{
			name:     "MobileRequest",
			payload:  mobileRequestPayload,
			expected: "p-bid-config-test-005",
		},
		{
			name:      "EmptyPublisherId",
			payload:   emptyPayload,
			expectNil: true,
		},
		{
			name:      "EmptyPayload",
			payload:   nil,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := newAccountInfoExtractor()
			accountInfo := extractor.extract(tt.payload)

			if tt.expectNil {
				assert.Nil(t, accountInfo)
			} else {
				assert.Equal(t, tt.expected, accountInfo.Id)
			}
		})
	}
}
