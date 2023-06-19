package adapters

import (
	"testing"

	"github.com/prebid/prebid-server/errortypes"
	"github.com/stretchr/testify/assert"
)

func TestCheckResponseStatusCodeForErrors(t *testing.T) {
	t.Run("bad_input", func(t *testing.T) {
		err := CheckResponseStatusCodeForErrors(&ResponseData{StatusCode: 400})
		expectedErr := &errortypes.BadInput{Message: "Unexpected status code: 400. Run with request.debug = 1 for more info"}
		assert.Equal(t, expectedErr.Error(), err.Error())
	})

	t.Run("internal_server_error", func(t *testing.T) {
		err := CheckResponseStatusCodeForErrors(&ResponseData{StatusCode: 500})
		expectedErrMessage := "Unexpected status code: 500. Run with request.debug = 1 for more info"
		assert.Equal(t, expectedErrMessage, err.Error())
	})
}

func TestIsResponseStatusCodeNoContent(t *testing.T) {
	assert.True(t, IsResponseStatusCodeNoContent(&ResponseData{StatusCode: 204}))
	assert.False(t, IsResponseStatusCodeNoContent(&ResponseData{StatusCode: 200}))
}
