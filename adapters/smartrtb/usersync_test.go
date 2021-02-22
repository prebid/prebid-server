package smartrtb

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestSmartRTBSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("http://market-east.smrtb.com/sync/all?nid=smartrtb&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&url=localhost%2Fsetuid%3Fbidder%smartrtb%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D"))
	syncer := NewSmartRTBSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})
	assert.NoError(t, err)
	assert.Equal(t, "http://market-east.smrtb.com/sync/all?nid=smartrtb&gdpr=&gdpr_consent=&url=localhost%2Fsetuid%3Fbidder%smartrtb%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
