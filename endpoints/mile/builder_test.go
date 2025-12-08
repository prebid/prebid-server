package mile

import (
	"encoding/json"
	"testing"
)

func TestBuildOpenRTBRequest(t *testing.T) {
	site := &SiteConfig{
		SiteID:      "FKKJK",
		PublisherID: "12345",
		Bidders:     []string{"appnexus", "rubicon"},
		Placements: map[string]PlacementConfig{
			"p1": {
				PlacementID:  "p1",
				AdUnit:       "banner_300x250",
				Sizes:        [][]int{{300, 250}},
				Floor:        0.25,
				BidderParams: map[string]json.RawMessage{"appnexus": json.RawMessage(`{"placementId":123}`)},
			},
		},
		SiteConfig: map[string]any{"page": "https://example.com"},
	}

	req := MileRequest{
		SiteID:      "FKKJK",
		PublisherID: "12345",
		PlacementID: "p1",
		CustomData:  []CustomData{{Targeting: map[string]any{"k": "v"}}},
	}

	got, err := buildOpenRTBRequest(req, site)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Site == nil || got.Site.Page != "https://example.com" {
		t.Fatalf("expected site page to be set")
	}

	if len(got.Imp) != 1 {
		t.Fatalf("expected 1 imp, got %d", len(got.Imp))
	}

	imp := got.Imp[0]
	if imp.TagID != "banner_300x250" {
		t.Errorf("expected TagID banner_300x250 got %s", imp.TagID)
	}

	if imp.Banner == nil || len(imp.Banner.Format) != 1 {
		t.Errorf("unexpected banner formats: %+v", imp.Banner)
	} else {
		if imp.Banner.Format[0].W != 300 || imp.Banner.Format[0].H != 250 {
			t.Errorf("unexpected banner size: %+v", imp.Banner.Format[0])
		}
	}

	ext, err := unmarshalExt(imp.Ext)
	if err != nil {
		t.Fatalf("failed to decode imp ext: %v", err)
	}

	prebid, ok := ext["prebid"].(map[string]any)
	if !ok {
		t.Fatalf("expected prebid ext")
	}

	bidder, ok := prebid["bidder"].(map[string]any)
	if !ok {
		t.Fatalf("expected bidder map")
	}

	if _, ok := bidder["appnexus"]; !ok {
		t.Errorf("expected appnexus bidder config")
	}
}

func TestBuildOpenRTBRequestErrors(t *testing.T) {
	site := &SiteConfig{Placements: map[string]PlacementConfig{}}
	if _, err := buildOpenRTBRequest(MileRequest{PlacementID: "p1"}, site); err == nil {
		t.Fatal("expected missing placement error")
	}

	site.Placements["p1"] = PlacementConfig{}
	if _, err := buildOpenRTBRequest(MileRequest{PlacementID: "p1"}, site); err != errNoBidders {
		t.Fatalf("expected errNoBidders, got %v", err)
	}
}
