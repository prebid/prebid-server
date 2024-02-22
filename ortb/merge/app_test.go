package merge

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestApp(t *testing.T) {
	testCases := []struct {
		name        string
		givenApp    openrtb2.App
		givenJson   json.RawMessage
		expectedApp openrtb2.App
		expectError bool
	}{
		{
			name:        "empty",
			givenApp:    openrtb2.App{},
			givenJson:   []byte(`{}`),
			expectedApp: openrtb2.App{},
		},
		{
			name:        "toplevel",
			givenApp:    openrtb2.App{ID: "1"},
			givenJson:   []byte(`{"id":"2"}`),
			expectedApp: openrtb2.App{ID: "2"},
		},
		{
			name:        "toplevel-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`)},
			givenJson:   []byte(`{"ext":{"b":100,"c":3}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`)},
		},
		{
			name:        "toplevel-ext-err",
			givenApp:    openrtb2.App{ID: "1", Ext: []byte(`malformed`)},
			givenJson:   []byte(`{"id":"2"}`),
			expectError: true,
		},
		{
			name:        "nested-publisher",
			givenApp:    openrtb2.App{Publisher: &openrtb2.Publisher{Name: "pub1"}},
			givenJson:   []byte(`{"publisher":{"name": "pub2"}}`),
			expectedApp: openrtb2.App{Publisher: &openrtb2.Publisher{Name: "pub2"}},
		},
		{
			name:        "nested-content",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1"}},
			givenJson:   []byte(`{"content":{"title": "content2"}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2"}},
		},
		{
			name:        "nested-content-producer",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1", Producer: &openrtb2.Producer{Name: "producer1"}}},
			givenJson:   []byte(`{"content":{"title": "content2", "producer":{"name":"producer2"}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2", Producer: &openrtb2.Producer{Name: "producer2"}}},
		},
		{
			name:        "nested-content-network",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1", Network: &openrtb2.Network{Name: "network1"}}},
			givenJson:   []byte(`{"content":{"title": "content2", "network":{"name":"network2"}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2", Network: &openrtb2.Network{Name: "network2"}}},
		},
		{
			name:        "nested-content-channel",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Title: "content1", Channel: &openrtb2.Channel{Name: "channel1"}}},
			givenJson:   []byte(`{"content":{"title": "content2", "channel":{"name":"channel2"}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Title: "content2", Channel: &openrtb2.Channel{Name: "channel2"}}},
		},
		{
			name:        "nested-publisher-ext",
			givenApp:    openrtb2.App{Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":1,"b":2}`)}},
			givenJson:   []byte(`{"publisher":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:        "nested-content-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Ext: []byte(`{"a":1,"b":2}`)}},
			givenJson:   []byte(`{"content":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Ext: []byte(`{"a":1,"b":100,"c":3}`)}},
		},
		{
			name:        "nested-content-producer-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenJson:   []byte(`{"content":{"producer":{"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-content-network-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenJson:   []byte(`{"content":{"network":{"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-content-channel-ext",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":1,"b":2}`)}}},
			givenJson:   []byte(`{"content":{"channel":{"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":1,"b":100,"c":3}`)}}},
		},
		{
			name:        "toplevel-ext-and-nested-publisher-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":10,"b":20}`)}},
			givenJson:   []byte(`{"ext":{"b":100,"c":3}, "publisher":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Publisher: &openrtb2.Publisher{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:        "toplevel-ext-and-nested-content-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Ext: []byte(`{"a":10,"b":20}`)}},
			givenJson:   []byte(`{"ext":{"b":100,"c":3}, "content":{"ext":{"b":100,"c":3}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Ext: []byte(`{"a":10,"b":100,"c":3}`)}},
		},
		{
			name:        "toplevel-ext-and-nested-content-producer-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenJson:   []byte(`{"ext":{"b":100,"c":3}, "content":{"producer": {"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "toplevel-ext-and-nested-content-network-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenJson:   []byte(`{"ext":{"b":100,"c":3}, "content":{"network": {"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "toplevel-ext-and-nested-content-channel-ext",
			givenApp:    openrtb2.App{Ext: []byte(`{"a":1,"b":2}`), Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":10,"b":20}`)}}},
			givenJson:   []byte(`{"ext":{"b":100,"c":3}, "content":{"channel": {"ext":{"b":100,"c":3}}}}`),
			expectedApp: openrtb2.App{Ext: []byte(`{"a":1,"b":100,"c":3}`), Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`{"a":10,"b":100,"c":3}`)}}},
		},
		{
			name:        "nested-publisher-ext-err",
			givenApp:    openrtb2.App{Publisher: &openrtb2.Publisher{Ext: []byte(`malformed`)}},
			givenJson:   []byte(`{"publisher":{"ext":{"b":100,"c":3}}}`),
			expectError: true,
		},
		{
			name:        "nested-content-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Ext: []byte(`malformed`)}},
			givenJson:   []byte(`{"content":{"ext":{"b":100,"c":3}}}`),
			expectError: true,
		},
		{
			name:        "nested-content-producer-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Producer: &openrtb2.Producer{Ext: []byte(`malformed`)}}},
			givenJson:   []byte(`{"content":{"producer": {"ext":{"b":100,"c":3}}}}`),
			expectError: true,
		},
		{
			name:        "nested-content-network-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Network: &openrtb2.Network{Ext: []byte(`malformed`)}}},
			givenJson:   []byte(`{"content":{"network": {"ext":{"b":100,"c":3}}}}`),
			expectError: true,
		},
		{
			name:        "nested-content-channel-ext-err",
			givenApp:    openrtb2.App{Content: &openrtb2.Content{Channel: &openrtb2.Channel{Ext: []byte(`malformed`)}}},
			givenJson:   []byte(`{"content":{"channelx": {"ext":{"b":100,"c":3}}}}`),
			expectError: true,
		},
		{
			name:        "json-err",
			givenApp:    openrtb2.App{ID: "1", Ext: []byte(`{"a":1}`)},
			givenJson:   []byte(`malformed`),
			expectError: true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			err := App(&test.givenApp, test.givenJson)

			if test.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, test.expectedApp, test.givenApp, " result app is incorrect")
			}
		})
	}
}

// TestAppObjectStructure detects when new nested objects are added to the App object,
// as these will create a gap in the merge.App logic. If this test fails, fix merge.App
// to add support and update this test to set a new baseline.
func TestAppObjectStructure(t *testing.T) {
	knownNestedStructs := []string{
		"Publisher",
		"Content",
		"Content.Producer",
		"Content.Network",
		"Content.Channel",
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
	discover("", reflect.TypeOf(openrtb2.App{}))

	assert.ElementsMatch(t, knownNestedStructs, discoveredNestedStructs)
}
