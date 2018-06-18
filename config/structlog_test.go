package config

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type testStruct struct {
	myint    int    `mapstructure:"this_int"`
	mystring string `mapstructure:"mystring"`
	flag     bool
	sub      innerStruct `mapstructure:"sub"`
}

type innerStruct struct {
	int1     int    `mapstructure:"int1"`
	password string `mapstructure:"password"`
}

var expected string = `this_int: 5
mystring: foobar
((flag)): false
sub.int1: 3
sub.password: <REDACTED>
`

func TestBasic(t *testing.T) {
	var buf bytes.Buffer

	mylogger := func(msg string, args ...interface{}) {
		buf.WriteString(fmt.Sprintf(fmt.Sprintln(msg), args...))
	}

	testCfg := testStruct{
		myint:    5,
		mystring: "foobar",
		sub: innerStruct{
			int1:     3,
			password: "secret",
		},
	}

	logStructWithLogger(reflect.ValueOf(testCfg), "", mylogger)

	result := buf.String()

	if expected != result {
		t.Errorf("Did not log properly.\ndesired:%s\nfound:%s\nsource: %v", expected, result, testCfg)
	}
}
