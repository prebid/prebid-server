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

func assertStringsEqual(t *testing.T, expected string, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}
