package consumable

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestConsumableSyncer(t *testing.T) {

	temp := template.Must(template.New("sync-template").Parse(
		"//e.serverbid.com/udb/9969/match?redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D"))

	syncer := NewConsumableSyncer(temp)

	u, _ := syncer.GetUsersyncInfo("0", "")
	assert.Equal(t, "//e.serverbid.com/udb/9969/match?redir=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D0%26gdpr_consent%3D%26uid%3D", u.URL)
	assert.Equal(t, "redirect", u.Type)
	assert.Equal(t, uint16(65535), syncer.GDPRVendorID())
	assert.Equal(t, false, u.SupportCORS)
}
