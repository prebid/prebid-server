package adapters

import (
	"testing"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestCheckResponseStatusCodeForErrors(t *testing.T) {
	testCases := []struct {
		name           string
		responseStatus int
		expectedErr    error
	}{
		{
			name:           "bad_input",
			responseStatus: 400,
			expectedErr: &errortypes.BadInput{
				Message: "Unexpected status code: 400. Run with request.debug = 1 for more info",
			},
		},
		{
			name:           "internal_server_error",
			responseStatus: 500,
			expectedErr: &errortypes.BadServerResponse{
				Message: "Unexpected status code: 500. Run with request.debug = 1 for more info",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckResponseStatusCodeForErrors(&ResponseData{StatusCode: tc.responseStatus})
			assert.Equal(t, tc.expectedErr, err)
		})
	}
}

func TestIsResponseStatusCodeNoContent(t *testing.T) {
	assert.True(t, IsResponseStatusCodeNoContent(&ResponseData{StatusCode: 204}))
	assert.False(t, IsResponseStatusCodeNoContent(&ResponseData{StatusCode: 200}))
}
