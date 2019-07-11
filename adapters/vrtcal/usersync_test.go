package vrtcal

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestVrtcalSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("http://usync-prebid.vrtcal.com/s?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"))
	syncer := NewVrtcalSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "http://usync-prebid.vrtcal.com/s?gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 0, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
	assert.Equal(t, "vrtcal", syncer.FamilyName())
}
