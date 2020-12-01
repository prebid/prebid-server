package exchange

import (
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestExtractGDPR(t *testing.T) {
	var gdprInt8 int8 = 1

	testCases := []struct {
		desc                  string
		inGdpr                *int8
		inUsersyncIfAmbiguous bool
		outGdpr               int
	}{
		{
			desc:                  "nil GDPR, usersync if ambiguous is false, expect 1",
			inGdpr:                nil,
			inUsersyncIfAmbiguous: false,
			outGdpr:               1,
		},
		{
			desc:                  "nil GDPR, usersync if ambiguous true, expect 0",
			inGdpr:                nil,
			inUsersyncIfAmbiguous: true,
			outGdpr:               0,
		},
		{
			desc:                  "GDPR was provided expect GDPR value",
			inGdpr:                &gdprInt8,
			inUsersyncIfAmbiguous: true,
			outGdpr:               int(gdprInt8),
		},
	}
	for _, test := range testCases {
		//run test
		actualGDPR := extractGDPR(test.inGdpr, test.inUsersyncIfAmbiguous)

		//assert
		assert.Equal(t, test.outGdpr, actualGDPR, "GDPR value mismatch. Test: %s \n", test.desc)
	}
}

func TestExtractConsent(t *testing.T) {
	testCases := []struct {
		desc       string
		inExtInfo  AuctionExtInfo
		outConsent string
	}{
		{
			desc:       "Nil unmarshalled user extension",
			inExtInfo:  AuctionExtInfo{},
			outConsent: "",
		},
		{
			desc: "Non-nil unmarshalled user extension comes with an empty consent string",
			inExtInfo: AuctionExtInfo{
				UserExt: &openrtb_ext.ExtUser{},
			},
			outConsent: "",
		},
		{
			desc: "Non-nil unmarshalled user extension with non-empty consent string",
			inExtInfo: AuctionExtInfo{
				UserExt: &openrtb_ext.ExtUser{
					Consent: "MY_CONSENT_STRING",
				},
			},
			outConsent: "MY_CONSENT_STRING",
		},
	}
	for _, test := range testCases {
		//run test
		actualConsent := extractConsent(test.inExtInfo)

		//assert
		assert.Equal(t, test.outConsent, actualConsent, "Consent string mismatch. Test: %s \n", test.desc)
	}
}

/*
func TestExtractGDPRFound(t *testing.T) {
	gdprTest := openrtb.BidRequest{
		User: &openrtb.User{
			Ext: json.RawMessage(`{"consent": "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"}`),
		},
		Regs: &openrtb.Regs{
			Ext: json.RawMessage(`{"gdpr": 1}`),
		},
	}
	gdpr := extractGDPR(&gdprTest, false)
	consent := extractConsent(&gdprTest)
	assert.Equal(t, 1, gdpr)
	assert.Equal(t, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA", consent)

	gdprTest.Regs.Ext = json.RawMessage(`{"gdpr": 0}`)
	gdpr = extractGDPR(&gdprTest, true)
	consent = extractConsent(&gdprTest)
	assert.Equal(t, 0, gdpr)
	assert.Equal(t, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA", consent)
}

func TestGDPRUnknown(t *testing.T) {
	gdprTest := openrtb.BidRequest{}

	gdpr := extractGDPR(&gdprTest, false)
	consent := extractConsent(&gdprTest)
	assert.Equal(t, 1, gdpr)
	assert.Equal(t, "", consent)

	gdpr = extractGDPR(&gdprTest, true)
	consent = extractConsent(&gdprTest)
	assert.Equal(t, 0, gdpr)

}
*/
