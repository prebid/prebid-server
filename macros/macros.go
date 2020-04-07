package macros

import (
	"bytes"
	"text/template"
)

// EndpointTemplateParams specifies params for an endpoint template
type EndpointTemplateParams struct {
	Host        string
	PublisherID string
<<<<<<< HEAD
=======
	ZoneID      string
	SourceId    string
>>>>>>> fb386190f4491648bb1e8d1b0345a333be1c0393
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
