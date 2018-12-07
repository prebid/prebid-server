package lifestreet

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestLifestreetSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%24visitor_cookie%24%24"))
	syncer := NewLifestreetSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "//ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl=localhost%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%24visitor_cookie%24%24", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 67, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
