package usersync

func NewRubiconSyncer(usersyncURL string) Usersyncer {
	return &syncer{
		familyName: "rubicon",
		syncInfo: &UsersyncInfo{
			URL:         usersyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
