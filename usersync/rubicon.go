package usersync

func NewRubiconSyncer(usersyncURL string) Usersyncer {
	return &syncer{
		familyName:   "rubicon",
		gdprVendorID: 52,
		syncInfo: &UsersyncInfo{
			URL:         usersyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
