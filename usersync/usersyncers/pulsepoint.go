package usersyncers

import (
	"fmt"
	"net/url"
)

func NewPulsepointSyncer(externalURL string) *syncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pulsepoint&uid=%s", externalURL, "%%VGUID%%")
	usersyncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl="

	return &syncer{
		familyName:          "pulsepoint",
		gdprVendorID:        81,
		syncEndpointBuilder: constEndpoint(fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri))),
		syncType:            SyncTypeRedirect,
	}
}
