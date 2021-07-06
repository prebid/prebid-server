package verizonmedia

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/stretchr/testify/assert"
)

func TestVerizonMediaSyncer(t *testing.T) {
	syncURL := "https://pixel.advertising.com/ups/58207/occ?http://localhost/%2Fsetuid%3Fbidder%3Dverizonmedia%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewVerizonMediaSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{})

	assert.NoError(t, err)
	assert.Equal(t, "redirect", syncInfo.Type)
}
