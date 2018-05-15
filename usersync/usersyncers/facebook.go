package usersyncers

import "github.com/prebid/prebid-server/usersync"

func NewFacebookSyncer(syncUrl string) *syncer {
	return &syncer{
		familyName: "audienceNetwork",
		syncInfo: &usersync.UsersyncInfo{
			URL:         syncUrl,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
