package telaria

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestTelariaSyncer(t *testing.T) {

	syncURL := "https://pbs.publishers.tremorhub.com/pubsync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewTelariaSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal: "0",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://pbs.publishers.tremorhub.com/pubsync?gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
	assert.Equal(t, "telaria", syncer.FamilyName())

}
