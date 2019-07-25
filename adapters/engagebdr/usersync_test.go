package engagebdr

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"text/template"
)

func TestEngageBDRSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://match.bnmla.com/usersync?sspid=99999&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redir=http%3A%2F%2Flocalhost"))
	syncer := NewEngageBDRSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA")
	assert.NoError(t, err)
	assert.Equal(t, "https://match.bnmla.com/usersync?sspid=99999&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&redir=http%3A%2F%2Flocalhost", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 62, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
