package usersync

import (
	"fmt"
	"github.com/golang/glog"
)

func NewAppnexusSyncer(externalURL string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=adnxs&uid=$UID", externalURL)
	usersyncURL := "//ib.adnxs.com/getuid?"

	glog.Info("\nredirect_uri    	: ", redirect_uri)
	glog.Info("\nusersyncURL    	: ", usersyncURL)
	glog.Info("\nxurl    		: ", externalURL)

	return &syncer{
		familyName: "adnxs",
		syncInfo: &UsersyncInfo{
			URL: 	     "http://yourmomshouse.com",
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
