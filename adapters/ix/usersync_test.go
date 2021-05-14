package ix

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestIxSyncer(t *testing.T) {
	syncURL := "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewIxSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D%26gdpr_consent%3D%26uid%3D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
