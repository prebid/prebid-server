package usersyncers

func NewRubiconSyncer(usersyncURL string) *syncer {
	return &syncer{
		familyName:          "rubicon",
		gdprVendorID:        52,
		syncEndpointBuilder: resolveMacros(usersyncURL),
		syncType:            SyncTypeRedirect,
	}
}
