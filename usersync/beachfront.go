package usersync

import (
	"fmt"
	"net/url"
)

func NewBeachfrontSyncer(external string) Usersyncer {
	redirect_uri := fmt.Sprintf("%s/setuid?bidder=beachfront&uid=$UID", external)
	// usersyncURL := "//sync.bfmio.com?url="
	usersyncURL := "http://10.0.0.181/fakesync.html?nothing="

	url := fmt.Sprintf("%s%s", usersyncURL, url.QueryEscape(redirect_uri))
	/*
	glog.Info("\nexternal    	: ", external)
	glog.Info("\nredirect_uri    	: ", redirect_uri)
	glog.Info("\nusersyncURL    	: ", usersyncURL)
	glog.Info("\nurl    		: ", url)
	*/
	// https://mysite.comsetuid?bidder=beachfront&uid=$UID"
	// https://usersync.bfmio.com?url=https%3A%2F%2Fmysite.comsetuid%3Fbidder%3Dbeachfront%26uid%3D%24UID%22

	return &syncer{
		familyName: "beachfront",
		syncInfo: &UsersyncInfo{
			URL:         url,
			Type:        "redirect",
			SupportCORS: true,
		},
	}
}
