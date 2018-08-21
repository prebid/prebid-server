package usersyncers

import (
	"testing"
)

func TestAdkernelAdnSyncer(t *testing.T) {
	syncr := NewAdkernelAdnSyncer("https://localhost:8888", "https://tag.adkernel.com/syncr?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&r=")
	syncInfo := syncr.GetUsersyncInfo("1", "BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw")
	url := "https://tag.adkernel.com/syncr?gdpr=1&gdpr_consent=BONciguONcjGKADACHENAOLS1rAHDAFAAEAASABQAMwAeACEAFw&r=https%3A%2F%2Flocalhost%3A8888%2Fsetuid%3Fbidder%3DadkernelAdn%26uid%3D%7BUID%7D"
	assertStringsMatch(t, url, syncInfo.URL)
	assertStringsMatch(t, "redirect", syncInfo.Type)
}
