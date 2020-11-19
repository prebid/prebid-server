package macros

import (
	"bytes"
	"text/template"
)

// EndpointTemplateParams specifies params for an endpoint template
type EndpointTemplateParams struct {
	Host        string
	PublisherID string
	ZoneID      string
	SourceId    string
	AccountID   string
	AdUnit      string
}

// UserSyncTemplateParams specifies params for an user sync URL template
type UserSyncTemplateParams struct {
	GDPR        string
	GDPRConsent string
	USPrivacy   string
}

// ResolveMacros resolves macros in the given template with the provided params
func ResolveMacros(aTemplate template.Template, params interface{}) (string, error) {
	strBuf := bytes.Buffer{}

	err := aTemplate.Execute(&strBuf, params)
	if err != nil {
		return "", err
	}
	res := strBuf.String()
	return res, nil
}
