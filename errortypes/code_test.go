package errortypes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadErrorCodeWithCodeDefined(t *testing.T) {
	err := &Timeout{Message: "code is defined"}

	result := ReadErrorCode(err)

	assert.Equal(t, result, TimeoutCode)
}

func TestReadErrorCodeWithCodeNotDefined(t *testing.T) {
	err := errors.New("missing error code")

	result := ReadErrorCode(err)

	assert.Equal(t, result, UnknownErrorCode)
}
