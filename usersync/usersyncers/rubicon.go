package usersyncers

import "github.com/prebid/prebid-server/usersync"

func NewRubiconSyncer(usersyncURL string) *syncer {
	return &syncer{
		familyName:   "rubicon",
		gdprVendorID: 52,
		syncInfo: &usersync.UsersyncInfo{
			URL:         usersyncURL,
			Type:        "redirect",
			SupportCORS: false,
		},
	}
}
