package merge

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/ortb"
	"github.com/prebid/prebid-server/v2/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestUser(t *testing.T) {
	testCases := []struct {
		name         string
		givenUser    openrtb2.User
		givenJson    json.RawMessage
		expectedUser openrtb2.User
		expectError  bool
	}{
		{
			name:         "empty",
			givenUser:    openrtb2.User{},
			givenJson:    []byte(`{}`),
			expectedUser: openrtb2.User{},
		},
		{
			name:         "toplevel",
			givenUser:    openrtb2.User{ID: "1"},
			givenJson:    []byte(`{"id":"2"}`),
			expectedUser: openrtb2.User{ID: "2"},
		},
		{
			name:         "toplevel-ext",
			givenUser:    openrtb2.User{Ext: []byte(`{"a":1,"b":2}`)},
			givenJson:    []byte(`{"ext":{"b":100,"c":3}}`),
			expectedUser: openrtb2.User{Ext: []byte(`{"a":1,"b":100,"c":3}`)},
		},
		{
			name:        "toplevel-ext-err",
			givenUser:   openrtb2.User{ID: "1", Ext: []byte(`malformed`)},
			givenJson:   []byte(`{"id":"2"}`),
			expectError: true,
		},
		{
			name:         "nested-geo",
			givenUser:    openrtb2.User{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(1.0)}},
			givenJson:    []byte(`{"geo":{"lat": 2}}`),
			expectedUser: openrtb2.User{Geo: &openrtb2.Geo{Lat: ptrutil.ToPtr(2.0)}},
		},
		{
			name:         "nested-geo-ext",
			givenUser:    openrtb2.User{Geo: &openrtb2.Geo{Ext: []byte(`{"a":1,"b":2}`)}},
			givenJson:    []byte(`{"geo":{"ext":{"b":100,"c":3}}}`),
			expectedUser: openrtb2.User{Geo: &openrtb2.Geo{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:         "toplevel-ext-and-nested-geo-ext",
			givenUser:    openrtb2.User{Ext: []byte(`{"a":1,"b":2}`), Geo: &openrtb2.Geo{Ext: []byte(`{"a":10,"b":20}`)}},
			givenJson:    []byte(`{"ext":{"b":100,"c":3}, "geo":{"ext":{"b":100,"c":3}}}`),
			expectedUser: openrtb2.User{Ext: []byte(`{"a":1,"b":100,"c":3}`), Geo: &openrtb2.Geo{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:        "nested-geo-ext-err",
			givenUser:   openrtb2.User{Geo: &openrtb2.Geo{Ext: []byte(`malformed`)}},
			givenJson:   []byte(`{"geo":{"ext":{"b":100,"c":3}}}`),
			expectError: true,
		},
		{
			name:        "json-err",
			givenUser:   openrtb2.User{ID: "1", Ext: []byte(`{"a":1}`)},
			givenJson:   []byte(`malformed`),
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			originalUser := ortb.CloneUser(&test.givenUser)
			merged, err := User(&test.givenUser, test.givenJson)

			assert.Equal(t, &test.givenUser, originalUser)

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, &test.expectedUser, merged, "result user is incorrect")
			}
		})
	}
}

// TestUserObjectStructure detects when new nested objects are added to the User object,
// as these will create a gap in the merge.User logic. If this test fails, fix merge.User
// to add support and update this test to set a new baseline.
func TestUserObjectStructure(t *testing.T) {
	knownNestedStructs := []string{
		"Geo",
	}

	discoveredNestedStructs := []string{}

	var discover func(parent string, t reflect.Type)
	discover = func(parent string, t reflect.Type) {
		fields := reflect.VisibleFields(t)
		for _, field := range fields {
			if field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct {
				discoveredNestedStructs = append(discoveredNestedStructs, parent+field.Name)
				discover(parent+field.Name+".", field.Type.Elem())
			}
		}
	}
	discover("", reflect.TypeOf(openrtb2.User{}))

	assert.ElementsMatch(t, knownNestedStructs, discoveredNestedStructs)
}
