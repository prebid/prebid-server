package constants

import "testing"

func TestFamilyName_String(t *testing.T) {
	if FNAppnexus.String() != "adnxs" {
		t.Error("FNApnnexus != 'adnxs'")
	}
	if FNFacebook.String() != "audienceNetwork" {
		t.Error("FNFacebook != 'audienceNetwork'")
	}
	if FNRubicon.String() != "rubicon" {
		t.Error("FNRubicon != 'rubicon'")
	}
}

func TestUIDsToMap(t *testing.T) {
	a := NewUIDArray()
	a[0] = "1234"
	a[2] = "3456"
	a[3] = "Banana"

	m := UIDsToMap(a)
	a2 := UIDsToArray(m)

	for i, v := range a {
		if a2[i] != v {
			t.Error("UIDToMap round trip failed")
		}
	}
}