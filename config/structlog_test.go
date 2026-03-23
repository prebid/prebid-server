package config

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

type testStruct struct {
	Myint          int    `mapstructure:"this_int"`
	Mystring       string `mapstructure:"mystring"`
	Flag           bool
	Sub            innerStruct `mapstructure:"sub"`
	Caps           map[string]string
	NestedStruct   *nestedStruct
	NillableStruct *nestedStruct
}

type innerStruct struct {
	int1     int    `mapstructure:"int1"`
	password string `mapstructure:"password"`
}

type nestedStruct struct {
	var1 string
}

var expected string = `this_int: 5
mystring: foobar
((Flag)): false
sub.int1: 3
sub.password: <REDACTED>
((Caps))[Alabama]: Montgomery
((NestedStruct)).((var1)): abc
`

func TestBasic(t *testing.T) {
	var buf bytes.Buffer

	mylogger := func(msg string, args ...interface{}) {
		buf.WriteString(fmt.Sprintf(fmt.Sprintln(msg), args...))
	}

	testCfg := testStruct{
		Myint:    5,
		Mystring: "foobar",
		Sub: innerStruct{
			int1:     3,
			password: "secret",
		},
		// Can't do more than one entry as order is not guaranteed.
		Caps: map[string]string{
			"Alabama": "Montgomery",
		},
		NestedStruct:   &nestedStruct{var1: "abc"},
		NillableStruct: nil,
	}

	logStructWithLogger(reflect.ValueOf(testCfg), "", mylogger)

	result := buf.String()

	if expected != result {
		t.Errorf("Did not log properly.\ndesired:%s\nfound:%s\nsource: %v", expected, result, testCfg)
	}
}
