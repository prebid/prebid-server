package errortypes

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadCodeWithCodeDefined(t *testing.T) {
	err := &Timeout{Message: "code is defined"}

	result := ReadCode(err)

	assert.Equal(t, result, TimeoutErrorCode)
}

func TestReadCodeWithCodeNotDefined(t *testing.T) {
	err := errors.New("missing error code")

	result := ReadCode(err)

	assert.Equal(t, result, UnknownErrorCode)
}
