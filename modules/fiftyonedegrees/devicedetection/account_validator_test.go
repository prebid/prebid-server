package devicedetection

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/stretchr/testify/assert"
)

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name           string
		allowList      []string
		expectedResult bool
	}{
		{
			name:           "allowed",
			allowList:      []string{"1001"},
			expectedResult: true,
		},
		{
			name:           "empty",
			allowList:      []string{},
			expectedResult: true,
		},
		{
			name:           "disallowed",
			allowList:      []string{"1002"},
			expectedResult: false,
		},
		{
			name:           "allow_list_is_nil",
			allowList:      nil,
			expectedResult: true,
		},
		{
			name:           "allow_list_contains_multiple",
			allowList:      []string{"1000", "1001", "1002"},
			expectedResult: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			validator := newAccountValidator()
			cfg := config{
				AccountFilter: accountFilter{AllowList: test.allowList},
			}

			res := validator.isAllowed(
				cfg, toBytes(
					&openrtb2.BidRequest{
						App: &openrtb2.App{
							Publisher: &openrtb2.Publisher{
								ID: "1001",
							},
						},
					},
				),
			)
			assert.Equal(t, test.expectedResult, res)
		})
	}
}

func toBytes(v interface{}) []byte {
	res, _ := json.Marshal(v)
	return res
}
