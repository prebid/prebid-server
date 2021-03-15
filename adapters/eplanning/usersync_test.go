package eplanning

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestEPlanningSyncer(t *testing.T) {
	syncURL := "https://ads.us.e-planning.net/uspd/1/?du=https%3A%2F%2Fads.us.e-planning.net%2Fgetuid%2F1%2F5a1ad71d2d53a0f5%3Flocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewEPlanningSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "https://ads.us.e-planning.net/uspd/1/?du=https%3A%2F%2Fads.us.e-planning.net%2Fgetuid%2F1%2F5a1ad71d2d53a0f5%3Flocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.Equal(t, false, syncInfo.SupportCORS)
}
