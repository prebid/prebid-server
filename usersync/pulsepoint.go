package usersync

import (
	"fmt"
	"net/url"
)

func NewPulsepointSyncer(externalURL string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=pulsepoint&uid=%s", externalURL, "%%VGUID%%")
	usersyncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl="
	return &syncer{
		familyName: "pulsepoint",
		syncInfo: &UsersyncInfo{
			URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
