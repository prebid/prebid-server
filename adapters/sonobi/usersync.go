package sonobi

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/usersync"
	"text/template"
)

// func NewSonobiSyncer(externalURL string) usersync.Usersyncer {
// 	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D{{.GDPR}}%26gdpr%3D{{.GDPRConsent}}%26uid%3D%24UID"
// 	usersyncURL := "//sync.go.sonobi.com/us.gif?loc="

// 	return &syncer{
// 		familyName:          "sonobi",
// 		gdprVendorID:        104,
// 		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
// 		syncType:            SyncTypeRedirect,
// 	}
// }

func NewSonobiSyncer(temp *template.Template) usersync.Usersyncer {
	return adapters.NewSyncer("sonobi", 104, temp, adapters.SyncTypeRedirect)
}
