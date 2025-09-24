package filelogger

import (
	"testing"

	"github.com/prebid/prebid-server/v3/analytics/analyticsdeps"
)

func TestBuilder_FileLoggerSuccess(t *testing.T) {
	m, err := Builder(map[string]interface{}{
		"enabled":  true,
		"filename": "abc.log",
	}, analyticsdeps.Deps{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatalf("expected module, got nil")
	}
}

func TestBuilder_FileLoggerInvalid(t *testing.T) {
	// missing filename
	m, err := Builder(map[string]interface{}{
		"enabled": true,
	}, analyticsdeps.Deps{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Fatalf("expected nil module for missing filename")
	}
	// disabled
	m2, err := Builder(map[string]interface{}{
		"enabled":  false,
		"filename": "x.log",
	}, analyticsdeps.Deps{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m2 != nil {
		t.Fatalf("expected nil when disabled")
	}
}
