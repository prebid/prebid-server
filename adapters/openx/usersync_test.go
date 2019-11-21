package openx

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestOpenxSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://rtb.openx.net/sync/prebid?r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D"))
	syncer := NewOpenxSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://rtb.openx.net/sync/prebid?r=localhost%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 69, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
