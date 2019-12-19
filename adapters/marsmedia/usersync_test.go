package marsmedia

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestMarsmediaSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("http://dmp.rtbsrv.com/dmp/profiles/cm?p_id=179&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirect=localhost:8000%2Fsetuid%3Fbidder%3Dmarsmedia%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUUID%7D"))
	syncer := NewMarsmediaSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})
	assert.NoError(t, err)
	assert.Equal(t, "http://dmp.rtbsrv.com/dmp/profiles/cm?p_id=179&gdpr=&gdpr_consent=&redirect=localhost:8000%2Fsetuid%3Fbidder%3Dmarsmedia%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
