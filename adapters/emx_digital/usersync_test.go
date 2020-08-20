package emx_digital

import (
	"testing"
	"text/template"

	"github.com/prebid/prebid-server/privacy"
	"github.com/prebid/prebid-server/privacy/ccpa"
	"github.com/prebid/prebid-server/privacy/gdpr"
	"github.com/stretchr/testify/assert"
)

func TestEMXDigitalSyncer(t *testing.T) {
	syncURL := "https://cs.emxdgt.com/um?ssp=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewEMXDigitalSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA",
		},
		CCPA: ccpa.Policy{
			Value: "1NYN",
		},
	})

	// Validate TCFV1 consent string
	assert.NoError(t, gdpr.ValidateConsent("BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA"))
	assert.NoError(t, err)
	assert.Equal(t, "https://cs.emxdgt.com/um?ssp=pbs&gdpr=1&gdpr_consent=BOPVK28OVJoTBABABAENBs-AAAAhuAKAANAAoACwAGgAPAAxAB0AHgAQAAiABOADkA&us_privacy=1NYN&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 183, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}

// Test TCFv2
func TestEMXDigitalSyncerTCF2(t *testing.T) {
	syncURL := "https://cs.emxdgt.com/um?ssp=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID"
	syncURLTemplate := template.Must(
		template.New("sync-template").Parse(syncURL),
	)

	syncer := NewEMXDigitalSyncer(syncURLTemplate)
	syncInfo, err := syncer.GetUsersyncInfo(privacy.Policies{
		GDPR: gdpr.Policy{
			Signal:  "1",
			Consent: "CO2d3mMO2d3mMDGAABNBAuCsAP_AAH_AAKiQGJNX_T5fb2vj-3Z99_tkeYwf95y3p-wzhheMs-8NyYeH7BoGv2MwvBX4JiQKGRgksjLBAQdtHGhcSQgBgIhViTLMYk2MjzNKJLJAilsbe0NYGD9unsHT3ZCY70-vu__7P3ff_wMSSmyigA3JNDRmwpgyRAoBgvQRlFDhhCCAwQYUgBgQcGACAJBEZheAvATEAEMDAIIEEAAgraOEAwEgACABADEEWIxJMYBGSESSABFAY0sgaQMCrQNIOFMwAxnJNRdXGmfO0bpAAA",
		},
		CCPA: ccpa.Policy{
			Value: "1NYN",
		},
	})
	// Validate TCFV2 consent string
	assert.NoError(t, gdpr.ValidateConsent("CO2d3mMO2d3mMDGAABNBAuCsAP_AAH_AAKiQGJNX_T5fb2vj-3Z99_tkeYwf95y3p-wzhheMs-8NyYeH7BoGv2MwvBX4JiQKGRgksjLBAQdtHGhcSQgBgIhViTLMYk2MjzNKJLJAilsbe0NYGD9unsHT3ZCY70-vu__7P3ff_wMSSmyigA3JNDRmwpgyRAoBgvQRlFDhhCCAwQYUgBgQcGACAJBEZheAvATEAEMDAIIEEAAgraOEAwEgACABADEEWIxJMYBGSESSABFAY0sgaQMCrQNIOFMwAxnJNRdXGmfO0bpAAA"))
	assert.NoError(t, err)
	assert.Equal(t, "https://cs.emxdgt.com/um?ssp=pbs&gdpr=1&gdpr_consent=CO2d3mMO2d3mMDGAABNBAuCsAP_AAH_AAKiQGJNX_T5fb2vj-3Z99_tkeYwf95y3p-wzhheMs-8NyYeH7BoGv2MwvBX4JiQKGRgksjLBAQdtHGhcSQgBgIhViTLMYk2MjzNKJLJAilsbe0NYGD9unsHT3ZCY70-vu__7P3ff_wMSSmyigA3JNDRmwpgyRAoBgvQRlFDhhCCAwQYUgBgQcGACAJBEZheAvATEAEMDAIIEEAAgraOEAwEgACABADEEWIxJMYBGSESSABFAY0sgaQMCrQNIOFMwAxnJNRdXGmfO0bpAAA&us_privacy=1NYN&redirect=localhost%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID", syncInfo.URL)
	assert.Equal(t, "iframe", syncInfo.Type)
	assert.EqualValues(t, 183, syncer.GDPRVendorID())
	assert.Equal(t, false, syncInfo.SupportCORS)
}
