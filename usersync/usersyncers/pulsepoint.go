package usersyncers

import (
	"net/url"
)

func NewPulsepointSyncer(externalURL string) *syncer {
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%25%25VGUID%25%25"
	usersyncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl="

	return &syncer{
		familyName:          "pulsepoint",
		gdprVendorID:        81,
		syncEndpointBuilder: resolveMacros(usersyncURL + redirectURI),
		syncType:            SyncTypeRedirect,
	}
}
