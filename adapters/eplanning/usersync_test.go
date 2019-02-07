package eplanning

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestEPlanningSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("http://sync.e-planning.net/um?uidlocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewEPlanningSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "http://sync.e-planning.net/um?uidlocalhost%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D%26gdpr_consent%3D%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
