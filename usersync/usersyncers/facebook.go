package usersyncers

func NewFacebookSyncer(syncUrl string) *syncer {
	return &syncer{
		familyName:          "audienceNetwork",
		syncEndpointBuilder: constEndpoint(syncUrl),
		syncType:            SyncTypeRedirect,
	}
}
