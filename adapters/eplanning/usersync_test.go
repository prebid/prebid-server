package eplanning

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestEPlanningSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://ads.us.e-planning.net/getuid/1/5a1ad71d2d53a0f5?localhost/setuid?bidder=eplanning&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID"))
	syncer := NewEPlanningSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://ads.us.e-planning.net/getuid/1/5a1ad71d2d53a0f5?localhost/setuid?bidder=eplanning&gdpr=&gdpr_consent=&uid=$UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
