package crossinstall

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestCrossInstallSyncer(t *testing.T) {
	syncURL := "https://www.crossinstall.com/sync.php?p=prebid&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewCrossInstallSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://www.crossinstall.com/sync.php?p=prebid&gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 807, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
	assert.Equal(t, "crossinstall", syncer.FamilyName())
}
