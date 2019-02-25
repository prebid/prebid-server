package util

import (
	"regexp"
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

// MacroRegExp is the regexp to check if a string contains a macro or not
const MacroRegExp string = `\{\{\.(\w+)\}\}`

// BuildTemplate builds a text template for a given string
func BuildTemplate(templateStr string) (*template.Template, error) {
	template, err := template.New("aTemplate").Parse(templateStr)
	if err != nil {
		return nil, err
	}
	return template, nil
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

// ContainsMacro checks and returns if the provided string contains a macro or not
func ContainsMacro(str string) bool {
	r, _ := regexp.Compile(MacroRegExp)
	return r.MatchString(str)
}
