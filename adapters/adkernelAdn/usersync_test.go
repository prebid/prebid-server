package adkernelAdn

import (
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestAdkernelAdnSyncer(t *testing.T) {
	temp := template.Must(template.New("sync-template").Parse("https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3DadkernelAdn%26uid%3D%7BUID%7D"))
	syncer := NewAdkernelAdnSyncer(temp)
	syncInfo, err := syncer.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	assert.NoError(t, err)
	assert.Equal(t, "https://tag.adkernel.com/syncr?gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&r=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3DadkernelAdn%26uid%3D%7BUID%7D", syncInfo.URL)
	assert.Equal(t, "redirect", syncInfo.Type)
	assert.EqualValues(t, adkernelGDPRVendorID, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
