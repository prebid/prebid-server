package util

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

var expected string = `This is test line 2
This is test line 3
`

func TestLogRandomSample(t *testing.T) {
	rand.Seed(1337)
	var buf bytes.Buffer

	mylogger := func(msg string, args ...interface{}) {
		buf.WriteString(fmt.Sprintf(fmt.Sprintln(msg), args...))
	}

	LogRandomSample("This is test line 1", mylogger, 0.5)
	LogRandomSample("This is test line 2", mylogger, 0.5)
	LogRandomSample("This is test line 3", mylogger, 0.5)
	LogRandomSample("This is test line 4", mylogger, 0.5)
	LogRandomSample("This is test line 5", mylogger, 0.5)

	assert.EqualValues(t, expected, buf.String())
}
