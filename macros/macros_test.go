package macros

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

const validEndpointTemplate = "http://{{.Host}}/publisher/{{.PublisherID}}"

func TestResolveMacros(t *testing.T) {
	endpointTemplate := template.Must(template.New("endpointTemplate").Parse(validEndpointTemplate))

	testCases := []struct {
		givenTemplate  *template.Template
		givenParams    interface{}
		expectedResult string
		expectedError  bool
	}{
		{
			givenTemplate:  endpointTemplate,
			givenParams:    EndpointTemplateParams{Host: "SomeHost", PublisherID: "1"},
			expectedResult: "http://SomeHost/publisher/1",
			expectedError:  false,
		},
		{
			givenTemplate:  endpointTemplate,
			givenParams:    UserSyncPrivacy{GDPR: "SomeGDPR", GDPRConsent: "SomeGDPRConsent"},
			expectedResult: "",
			expectedError:  true,
		},
	}

	for _, test := range testCases {
		result, err := ResolveMacros(test.givenTemplate, test.givenParams)

		if test.expectedError {
			assert.NotNil(t, err, "Error shouldn't be nil")
			assert.Empty(t, result, "Result should be empty")
		} else {
			assert.Nil(t, err, "Err should be nil")
			assert.Equal(t, result, test.expectedResult, "String after resolving macros should be %s", test.expectedResult)
		}
	}
}
