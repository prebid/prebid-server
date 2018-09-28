package usersyncers

func NewIxSyncer(userSyncURL string) *syncer {
	return &syncer{
		familyName:          "ix",
		gdprVendorID:        10,
		syncEndpointBuilder: resolveMacros(userSyncURL),
		syncType:            SyncTypeRedirect,
	}
}
