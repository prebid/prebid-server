package sharethrough

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestSharethroughSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://match.sharethrough.com?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}"))
	syncer := NewSharethroughSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("0", "")
	assert.NoError(t, err)
	assert.Equal(t, "https://match.sharethrough.com?gdpr=0&gdpr_consent=", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, 80, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
	assert.Equal(t, "sharethrough", syncer.FamilyName())
}
