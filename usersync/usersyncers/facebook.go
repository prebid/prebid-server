package usersyncers

func NewFacebookSyncer(syncUrl string) *syncer {
	return &syncer{
		familyName:          "audienceNetwork",
		syncEndpointBuilder: resolveMacros(syncUrl),
		syncType:            SyncTypeRedirect,
	}
}
