package exchange

import (
	"testing"

	"github.com/mxmCherry/openrtb"
)

func TestExtractGDPRFound(t *testing.T) {
	gdprTest := openrtb.BidRequest{
		User: &openrtb.User{
			Ext: openrtb.RawJSON(`{"consent": "BOS2bx5OS2bx5ABABBAAABoAAAAAFA"}`),
		},
		Regs: &openrtb.Regs{
			Ext: openrtb.RawJSON(`{"gdpr": 1}`),
		},
	}
	gdpr, consent, err := extractGDPR(&gdprTest, false)
	assertNilErr(t, err)
	assertIntsEqual(t, 1, gdpr)
	assertStringsEqual(t, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA", consent)

	gdprTest.Regs.Ext = openrtb.RawJSON(`{"gdpr": 0}`)
	gdpr, consent, err = extractGDPR(&gdprTest, true)
	assertNilErr(t, err)
	assertIntsEqual(t, 0, gdpr)
	assertStringsEqual(t, "BOS2bx5OS2bx5ABABBAAABoAAAAAFA", consent)
}

func TestGDPRUnknown(t *testing.T) {
	gdprTest := openrtb.BidRequest{}

	gdpr, consent, err := extractGDPR(&gdprTest, false)
	assertIntsEqual(t, 0, gdpr)
	assertStringsEqual(t, "", consent)
	assertNilErr(t, err)

	gdpr, consent, err = extractGDPR(&gdprTest, true)
	assertIntsEqual(t, 1, gdpr)

}

func TestCleanPI(t *testing.T) {
	bidReqOrig := openrtb.BidRequest{}

	bidReqCopy := bidReqOrig
	// Make sure cleanIP handles the empty case
	cleanPI(&bidReqCopy)

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

	cleanPI(&bidReqCopy)

	// Verify cleaned values
	assertStringEmpty(t, bidReqCopy.User.BuyerUID)
	assertStringEmpty(t, bidReqCopy.Device.DIDMD5)
	assertStringsEqual(t, "12.123.56.000", bidReqCopy.Device.IP)
	assertStringsEqual(t, "2001:0db8:85a3:0000:0000:8a2e:0370:0000", bidReqCopy.Device.IPv6)
	assertFloatsEqual(t, 123.46, bidReqCopy.Device.Geo.Lat)
	assertFloatsEqual(t, 7.98, bidReqCopy.Device.Geo.Lon)

	// verify original untouched, as we want to only modify the cleaned copy for the bidder
	assertStringsEqual(t, "abc123", bidReqOrig.User.BuyerUID)
	assertStringsEqual(t, "teapot", bidReqOrig.Device.DIDMD5)
	assertStringsEqual(t, "12.123.56.128", bidReqOrig.Device.IP)
	assertStringsEqual(t, "2001:0db8:85a3:0000:0000:8a2e:0370:7334", bidReqOrig.Device.IPv6)
	assertFloatsEqual(t, 123.4567, bidReqOrig.Device.Geo.Lat)
	assertFloatsEqual(t, 7.9836, bidReqOrig.Device.Geo.Lon)

}

func assertNilErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func assertIntsEqual(t *testing.T, expected int, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %d, got %d", expected, actual)
	}
}

func assertFloatsEqual(t *testing.T, expected float64, actual float64) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func assertStringsEqual(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func assertStringEmpty(t *testing.T, str string) {
	t.Helper()
	if str != "" {
		t.Errorf("Expected an empty string, got %s", str)
	}
}
