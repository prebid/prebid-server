package ix

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestIxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"))
	syncer := NewIxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3Dix%26gdpr%3D%26gdpr_consent%3D%26uid%3D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 10, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
