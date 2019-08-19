package pubmatic

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestPubmaticSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"))
	syncer := NewPubmaticSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assert.NoError(t, err)
	assert.Equal(t, "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D1%26gdpr_consent%3DBONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw%26uid%3D", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 76, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
