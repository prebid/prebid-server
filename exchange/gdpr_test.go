package exchange

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/stretchr/testify/assert"
)

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
