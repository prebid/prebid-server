package usersync

func NewIndexSyncer(userSyncURL string) Usersyncer {
	return &syncer{
		familyName: "indexExchange",
		syncInfo: &UsersyncInfo{
			URL:         userSyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
