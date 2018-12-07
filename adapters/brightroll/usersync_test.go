package brightroll

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestBrightrollSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url=localhost%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D"))
	syncer := NewBrightrollSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr=&euconsent=&url=localhost%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 25, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
