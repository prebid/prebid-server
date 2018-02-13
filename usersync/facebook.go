package usersync

import (
	"github.com/prebid/prebid-server/pbs"
)

func NewFacebookSyncer(syncUrl string) Usersyncer {
	return &syncer{
		familyName: "audienceNetwork",
		syncInfo: &pbs.UsersyncInfo{
			URL:         syncUrl,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
