package usersync

import (
	"fmt"
	"github.com/golang/glog"
	"net/url"
)

func NewAppnexusSyncer(externalURL string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	glog.Info("\nredirect_uri    	: ", redirect_uri)
	glog.Info("\nusersyncURL    	: ", usersyncURL)
	glog.Info("\xurl    		: ", externalURL)

	return &syncer{
		familyName: "adnxs",
		syncInfo: &UsersyncInfo{
			// URL:         fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri)),
			URL: 	     "http://yourmomshouse.com",
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
