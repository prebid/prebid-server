package agma

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

func TestBuilder_AgmaSuccess(t *testing.T) {
	m, err := Builder(map[string]interface{}{
		"enabled":  true,
		"endpoint": "https://agma.example.com/collect",
		"buffers": map[string]interface{}{
			"eventCount": 25,
			"bufferSize": "64KB",
			"timeout":    "1s",
		},
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatalf("expected module, got nil")
	}
}

func TestBuilder_AgmaInvalidOrDisabled(t *testing.T) {
	// missing endpoint
	m1, err := Builder(map[string]interface{}{
		"enabled": true,
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m1 != nil {
		t.Fatalf("expected nil without endpoint")
	}
	// disabled
	m2, err := Builder(map[string]interface{}{
		"enabled":  false,
		"endpoint": "https://agma.example.com/collect",
	}, deps())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m2 != nil {
		t.Fatalf("expected nil when disabled")
	}
}
