package usersync

func NewFacebookSyncer(syncUrl string) Usersyncer {
	return &syncer{
		familyName: "audienceNetwork",
		syncInfo: &UsersyncInfo{
			URL:         syncUrl,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
