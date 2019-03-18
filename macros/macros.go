package macros

import (
	"strings"
	"text/template"
)

// EndpointTemplateParams specifies params for an endpoint template
type EndpointTemplateParams struct {
	Host        string
	PublisherID int
}

// UserSyncTemplateParams specifies params for an user sync URL template
type UserSyncTemplateParams struct {
	GDPR        string
	GDPRConsent string
}

// ResolveMacros resolves macros in the given template with the provided params
func ResolveMacros(aTemplate template.Template, params interface{}) (string, error) {
	strBuilder := strings.Builder{}
	err := aTemplate.Execute(&strBuilder, params)
	if err != nil {
		return "", err
	}
	res := strBuilder.String()
	return res, nil
}
