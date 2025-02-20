package pixfuture

import (
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestPixfutureParams(t *testing.T) {
	testCases := []struct {
		name          string
		params        openrtb_ext.ImpExtPixfuture
		expectedError string
	}{
		{
			name: "Valid Params",
			params: openrtb_ext.ImpExtPixfuture{
				PlacementID: "123",
			},
			expectedError: "",
		},
		{
			name: "Missing PlacementID",
			params: openrtb_ext.ImpExtPixfuture{
				PlacementID: "",
			},
			expectedError: "PlacementID is required",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := validatePixfutureParams(test.params)
			if test.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.expectedError)
			}
		})
	}
}

func validatePixfutureParams(params openrtb_ext.ImpExtPixfuture) error {
	if params.PlacementID == "" {
		return fmt.Errorf("PlacementID is required")
	}
	return nil
}
