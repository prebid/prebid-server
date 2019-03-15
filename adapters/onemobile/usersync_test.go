package onemobile

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestOneMobileSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://pixel.advertising.com/ups/58207/occ?http://localhost/%2Fsetuid%3Fbidder%3Donemobile%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID"))
	syncer := NewOneMobileSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("", "")
	assert.NoError(t, err)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 25, syncer.GDPRVendorID())
}
