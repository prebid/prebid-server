package macros

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

const validEndpointTemplate = "http://{{.Host}}/publisher/{{.PublisherID}}"

func TestResolveMacros(t *testing.T) {
	endpointTemplate, _ := template.New("endpointTemplate").Parse(validEndpointTemplate)

	testCases := []struct {
		aTemplate template.Template
		params    interface{}
		result    string
		hasError  bool
	}{
		{aTemplate: *endpointTemplate, params: EndpointTemplateParams{Host: "SomeHost", PublisherID: "1"}, result: "http://SomeHost/publisher/1", hasError: false},
		{aTemplate: *endpointTemplate, params: UserSyncTemplateParams{GDPR: "SomeGDPR", GDPRConsent: "SomeGDPRConsent"}, result: "", hasError: true},
	}

	for _, test := range testCases {
		res, err := ResolveMacros(test.aTemplate, test.params)

		if test.hasError {
			assert.NotNil(t, err, "Error shouldn't be nil")
			assert.Empty(t, res, "Result should be empty")
		} else {
			assert.Nil(t, err, "Err should be nil")
			assert.Equal(t, res, test.result, "String after resolving macros should be %s", test.result)
		}
	}
}
