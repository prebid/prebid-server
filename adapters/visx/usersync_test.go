package visx

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestVisxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("//not_localhost/synclocalhost%2Fsetuid%3Fbidder%3Dvisx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUUID%7D"))
	syncer := NewVisxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "//not_localhost/synclocalhost%2Fsetuid%3Fbidder%3Dvisx%26gdpr%3D0%26gdpr_consent%3D%26uid%3D%24%7BUUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
