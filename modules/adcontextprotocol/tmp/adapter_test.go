package tmp

import (
	"testing"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/openrtb/v20/openrtb2"
)

func TestDeriveInputs_Site(t *testing.T) {
	cfg := &Config{}
	req := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "example.com", Page: "https://example.com/article"},
		Imp:  []openrtb2.Imp{{TagID: "slot-1"}},
		Device: &openrtb2.Device{
			Geo: &openrtb2.Geo{Country: "US", Region: "CA", Metro: "807"},
		},
	}
	in := deriveInputs(cfg, req)

	if in.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", in.Domain, "example.com")
	}
	if in.PlacementID != "slot-1" {
		t.Errorf("PlacementID = %q, want %q", in.PlacementID, "slot-1")
	}
	if in.PropertyType != tmproto.PropertyTypeWebsite {
		t.Errorf("PropertyType = %q, want website", in.PropertyType)
	}
	if len(in.ArtifactRefs) != 1 || in.ArtifactRefs[0].Type != tmproto.ArtifactRefTypeURL {
		t.Errorf("ArtifactRefs = %+v, want single url ref", in.ArtifactRefs)
	}
	if in.Country != "US" {
		t.Errorf("Country = %q, want %q", in.Country, "US")
	}
	if in.Geo["metro"] != "807" {
		t.Errorf("Geo[metro] = %v, want 807", in.Geo["metro"])
	}
}

func TestDeriveInputs_App(t *testing.T) {
	cfg := &Config{}
	req := &openrtb2.BidRequest{
		App: &openrtb2.App{Bundle: "com.example.app"},
	}
	in := deriveInputs(cfg, req)
	if in.Bundle != "com.example.app" {
		t.Errorf("Bundle = %q, want com.example.app", in.Bundle)
	}
	if in.PropertyType != tmproto.PropertyTypeMobileApp {
		t.Errorf("PropertyType = %q, want mobile_app", in.PropertyType)
	}
}

func TestDeriveInputs_IdentityCap(t *testing.T) {
	cfg := &Config{}
	req := &openrtb2.BidRequest{
		User: &openrtb2.User{
			EIDs: []openrtb2.EID{
				{Source: "id5-sync.com", UIDs: []openrtb2.UID{{ID: "id5-x"}}},
				{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp-x"}}},
				{Source: "uidapi.com", UIDs: []openrtb2.UID{{ID: "uid2-x"}}},
				{Source: "adserver.org", UIDs: []openrtb2.UID{{ID: "pair-x"}}},
				{Source: "unknown", UIDs: []openrtb2.UID{{ID: "u"}}},
			},
		},
	}
	in := deriveInputs(cfg, req)
	if len(in.Identities) != 3 {
		t.Fatalf("Identities length = %d, want 3", len(in.Identities))
	}
	// Priority: liveramp, uidapi, id5.
	if in.Identities[0].UIDType != tmproto.UIDTypeRampID {
		t.Errorf("first identity uid_type = %q, want rampid", in.Identities[0].UIDType)
	}
	if in.Identities[1].UIDType != tmproto.UIDTypeUID2 {
		t.Errorf("second identity uid_type = %q, want uid2", in.Identities[1].UIDType)
	}
	if in.Identities[2].UIDType != tmproto.UIDTypeID5 {
		t.Errorf("third identity uid_type = %q, want id5", in.Identities[2].UIDType)
	}
}

func TestDeriveInputs_ConsentGPP(t *testing.T) {
	cfg := &Config{}
	req := &openrtb2.BidRequest{
		Regs: &openrtb2.Regs{GPP: "DBABMA~CPXxRfAPXxRfAAfKABENB-CgAAAAAAAAAAYgAAAAAAAA"},
	}
	in := deriveInputs(cfg, req)
	if in.Consent["gpp"] == nil {
		t.Errorf("Consent.gpp missing, got %+v", in.Consent)
	}
}

func TestMapEIDToUIDType(t *testing.T) {
	cases := map[string]tmproto.UIDType{
		"liveramp.com": tmproto.UIDTypeRampID,
		"uidapi.com":   tmproto.UIDTypeUID2,
		"id5-sync.com": tmproto.UIDTypeID5,
		"euid.eu":      tmproto.UIDTypeEUID,
		"adserver.org": tmproto.UIDTypePairID,
		"unknown":      "",
		"":             "",
	}
	for src, want := range cases {
		got := mapEIDToUIDType(src)
		if got != want {
			t.Errorf("mapEIDToUIDType(%q) = %q, want %q", src, got, want)
		}
	}
}
