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

func TestCleanPI(t *testing.T) {
	bidReqOrig := openrtb.BidRequest{}

	bidReqCopy := bidReqOrig
	// Make sure cleanIP handles the empty case
	cleanPI(&bidReqCopy, false)

	// Add values to clean
	bidReqOrig.User = &openrtb.User{
		BuyerUID: "abc123",
	}
	bidReqOrig.Device = &openrtb.Device{
		DIDMD5: "teapot",
		IP:     "12.123.56.128",
		IPv6:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		Geo: &openrtb.Geo{
			Lat: 123.4567,
			Lon: 7.9836,
		},
	}
	// Make a shallow copy
	bidReqCopy = bidReqOrig

	cleanPI(&bidReqCopy, false)

	// Verify cleaned values
	assertStringEmpty(t, bidReqCopy.User.BuyerUID)
	assertStringEmpty(t, bidReqCopy.Device.DIDMD5)
	assert.Equal(t, "12.123.56.0", bidReqCopy.Device.IP)
	assert.Equal(t, "2001:0db8:85a3:0000:0000:8a2e:0370:0000", bidReqCopy.Device.IPv6)
	assert.Equal(t, 123.46, bidReqCopy.Device.Geo.Lat)
	assert.Equal(t, 7.98, bidReqCopy.Device.Geo.Lon)

	// verify original untouched, as we want to only modify the cleaned copy for the bidder
	assert.Equal(t, "abc123", bidReqOrig.User.BuyerUID)
	assert.Equal(t, "teapot", bidReqOrig.Device.DIDMD5)
	assert.Equal(t, "12.123.56.128", bidReqOrig.Device.IP)
	assert.Equal(t, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", bidReqOrig.Device.IPv6)
	assert.Equal(t, 123.4567, bidReqOrig.Device.Geo.Lat)
	assert.Equal(t, 7.9836, bidReqOrig.Device.Geo.Lon)

}

func TestCleanPIAmp(t *testing.T) {
	bidReqOrig := openrtb.BidRequest{}

	bidReqCopy := bidReqOrig
	// Make sure cleanIP handles the empty case
	cleanPI(&bidReqCopy, false)

	// Add values to clean
	bidReqOrig.User = &openrtb.User{
		BuyerUID: "abc123",
	}
	bidReqOrig.Device = &openrtb.Device{
		DIDMD5: "teapot",
		IP:     "12.123.56.128",
		IPv6:   "2001:0db8:85a3:0000:0000:8a2e:0370:7334",
		Geo: &openrtb.Geo{
			Lat: 123.4567,
			Lon: 7.9836,
		},
	}
	// Make a shallow copy
	bidReqCopy = bidReqOrig

	cleanPI(&bidReqCopy, true)

	// Verify cleaned values
	assert.Equal(t, "abc123", bidReqCopy.User.BuyerUID)
	assertStringEmpty(t, bidReqCopy.Device.DIDMD5)
	assert.Equal(t, "12.123.56.0", bidReqCopy.Device.IP)
	assert.Equal(t, "2001:0db8:85a3:0000:0000:8a2e:0370:0000", bidReqCopy.Device.IPv6)
	assert.Equal(t, 123.46, bidReqCopy.Device.Geo.Lat)
	assert.Equal(t, 7.98, bidReqCopy.Device.Geo.Lon)
}

func assertStringEmpty(t *testing.T, str string) {
	t.Helper()
	if str != "" {
		t.Errorf("Expected an empty string, got %s", str)
	}
}

func TestBadIPs(t *testing.T) {
	assertStringEmpty(t, cleanIP("not an IP"))
	assertStringEmpty(t, cleanIP(""))
	assertStringEmpty(t, cleanIP("36278042"))
	assertStringEmpty(t, cleanIPv6("not an IP"))
	assertStringEmpty(t, cleanIPv6(""))
}
