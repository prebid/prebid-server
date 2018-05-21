package usersyncers

import (
	"testing"
)

func TestPulsepointSyncer(t *testing.T) {
	pulsepoint := NewPulsepointSyncer("http://localhost")
	info := pulsepoint.GetUsersyncInfo("", "")
	assertStringsMatch(t, "redirect", info.Type)
	assertStringsMatch(t, "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D%26gdpr_consent%3D%26uid%3D%25%25VGUID%25%25", info.URL)
	if pulsepoint.GDPRVendorID() != 81 {
		t.Errorf("Wrong Pulsepoint GDPR VendorID. Got %d", pulsepoint.GDPRVendorID())
	}
}
