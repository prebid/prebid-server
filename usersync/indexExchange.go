package usersync

func NewIndexSyncer(userSyncURL string) Usersyncer {
	return &syncer{
		familyName:   "indexExchange",
		gdprVendorID: 10,
		syncInfo: &UsersyncInfo{
			URL:         userSyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
