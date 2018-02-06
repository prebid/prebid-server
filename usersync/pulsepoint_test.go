package usersync

import (
	"fmt"
	"testing"
)

func TestPulsepointSyncer(t *testing.T) {
	info := NewPulsepointSyncer("http://localhost").GetUsersyncInfo()
	verifyStringValue(info.Type, "redirect", t)
	verifyStringValue(info.URL, "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dpulsepoint%26uid%3D%25%25VGUID%25%25", t)
}

func verifyStringValue(value string, expected string, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%s expected, got %s", expected, value))
	}
}
