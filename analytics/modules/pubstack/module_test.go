package pubstack

import (
	"net/http"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
)

func deps() analyticsdeps.Deps {
	return analyticsdeps.Deps{
		HTTPClient: &http.Client{},
		Clock:      clock.New(),
	}
}

func TestBuilder_PubstackSuccess(t *testing.T) {
	m, err := Builder(map[string]interface{}{
		"enabled":   true,
		"scopeId":   "scopeX",
		"intakeUrl": "https://example.com/i",
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatalf("expected module, got nil")
	}
}

func TestBuilder_PubstackMissingFields(t *testing.T) {
	// missing scopeId
	m1, err := Builder(map[string]interface{}{
		"enabled":   true,
		"intakeUrl": "https://example.com/i",
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m1 != nil {
		t.Fatalf("expected nil without scopeId")
	}
	// missing intakeUrl
	m2, err := Builder(map[string]interface{}{
		"enabled": true,
		"scopeId": "scopeX",
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m2 != nil {
		t.Fatalf("expected nil without intakeUrl")
	}
	// disabled
	m3, err := Builder(map[string]interface{}{
		"enabled":   false,
		"scopeId":   "scopeX",
		"intakeUrl": "https://example.com/i",
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m3 != nil {
		t.Fatalf("expected nil when disabled")
	}
}
