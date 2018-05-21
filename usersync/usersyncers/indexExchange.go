package usersyncers

func NewIndexSyncer(userSyncURL string) *syncer {
	return &syncer{
		familyName:          "indexExchange",
		gdprVendorID:        10,
		syncEndpointBuilder: resolveMacros(userSyncURL),
		syncType:            SyncTypeRedirect,
	}
}
