package macros

import (
	"bytes"
	"text/template"
)

// EndpointTemplateParams specifies macros for bidder endpoints.
type EndpointTemplateParams struct {
	Host        string
	PublisherID string
	ZoneID      string
	SourceId    string
	AccountID   string
	AdUnit      string
	MediaType   string
	GvlID       string
	PageID      string
	SupplyId    string
	ImpID       string
	SspId       string
	SspID       string
	SeatID      string
	TokenID     string
}

// UserSyncPrivacy specifies privacy policy macros, represented as strings, for user sync urls.
type UserSyncPrivacy struct {
	GDPR        string
	GDPRConsent string
	USPrivacy   string
	GPP         string
	GPPSID      string
}

// ResolveMacros resolves macros in the given template with the provided params
func ResolveMacros(aTemplate *template.Template, params interface{}) (string, error) {
	strBuf := bytes.Buffer{}

	if err := aTemplate.Execute(&strBuf, params); err != nil {
		return "", err
	}

	res := strBuf.String()
	return res, nil
}
