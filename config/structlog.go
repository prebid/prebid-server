package config

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/golang/glog"
)

var mapregex *regexp.Regexp = regexp.MustCompile(`mapstructure:"([^"]+)"`)
var blacklistregexp []*regexp.Regexp = []*regexp.Regexp{
	regexp.MustCompile("password"),
}

// LogGeneral will log nearly any sort of value, but requires the name of the root object to be in the
// prefix if you want that name to be logged. Structs will append .<fieldname> recursively to the prefix
// to document deeper structure.
func logGeneral(v reflect.Value, prefix string) {
	logGeneralWithLogger(v, prefix, glog.Infof)
}

func logGeneralWithLogger(v reflect.Value, prefix string, logger func(msg string, args ...interface{})) {
	switch v.Kind() {
	case reflect.Struct:
		logStructWithLogger(v, prefix, logger)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		logger("%s: %d", prefix, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		logger("%s: %d", prefix, v.Uint())
	case reflect.Float32, reflect.Float64:
		logger("%s: %f", prefix, v.Float())
	case reflect.Bool:
		logger("%s: %t", prefix, v.Bool())
	default:
		// logString, by using v.String(), will not fail, and indicate what additional cases we need to handle
		logger("%s: %s", prefix, v.String())
	}
}

func logStructWithLogger(v reflect.Value, prefix string, logger func(msg string, args ...interface{})) {
	if v.Kind() != reflect.Struct {
		glog.Fatalf("LogStruct called on type %s, whuch is not a struct!", v.Type().String())
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fieldname := fieldNameByTag(t.Field(i))
		if allowedName(fieldname) {
			logGeneralWithLogger(v.Field(i), extendPrefix(prefix, fieldname), logger)
		} else {
			logger("%s.%s: <REDACTED>", prefix, fieldname)
		}
	}
}

func fieldNameByTag(f reflect.StructField) string {
	match := mapregex.FindStringSubmatch(string(f.Tag))
	if len(match) == 0 || len(match[1]) == 0 {
		return fmt.Sprintf("((%s))", f.Name)
	}
	return match[1]
}

func allowedName(name string) bool {
	for _, r := range blacklistregexp {
		if r.MatchString(name) {
			return false
		}
	}
	return true
}

func extendPrefix(prefix string, field string) string {
	if len(strings.Trim(prefix, " \t")) == 0 {
		return fmt.Sprintf("%s%s", prefix, field)
	}
	return fmt.Sprintf("%s.%s", prefix, field)
}
