package analytics

import (
	"testing"
)

// getEnabledAnalytics wywołuje New() i asertywnie rzutuje wynik na EnabledAnalytics.
func getEnabledAnalytics(t *testing.T, cfg map[string]interface{}) EnabledAnalytics {
	t.Helper()
	r := New(cfg)
	ea, ok := r.(EnabledAnalytics)
	if !ok {
		t.Fatalf("expected New() to return EnabledAnalytics, got %T", r)
	}
	return ea
}

func TestNew_WithFileLoggerConfig(t *testing.T) {
	ea := getEnabledAnalytics(t, map[string]interface{}{
		"filelogger": map[string]interface{}{
			"enabled":  true,
			"filename": "test.log",
		},
	})
	if _, ok := ea["filelogger"]; !ok {
		t.Fatalf("expected filelogger module to be initialized")
	}
}

func TestNew_FileLoggerDisabledOrInvalid(t *testing.T) {
	// brak filename => nie powinno być modułu
	ea := getEnabledAnalytics(t, map[string]interface{}{
		"filelogger": map[string]interface{}{
			"enabled": true,
		},
	})
	if _, ok := ea["filelogger"]; ok {
		t.Fatalf("did not expect filelogger with missing filename")
	}
	// disabled
	ea2 := getEnabledAnalytics(t, map[string]interface{}{
		"filelogger": map[string]interface{}{
			"enabled":  false,
			"filename": "x.log",
		},
	})
	if _, ok := ea2["filelogger"]; ok {
		t.Fatalf("did not expect filelogger when disabled")
	}
}

func TestNew_WithPubstackConfig(t *testing.T) {
	ea := getEnabledAnalytics(t, map[string]interface{}{
		"pubstack": map[string]interface{}{
			"enabled":   true,
			"scopeId":   "scope1",
			"intakeUrl": "https://example.com/intake",
			"buffers": map[string]interface{}{
				"eventCount": 10,
				"bufferSize": "64KB",
				"timeout":    "1s",
			},
		},
	})
	if _, ok := ea["pubstack"]; !ok {
		t.Fatalf("expected pubstack module to be initialized")
	}
}

func TestNew_PubstackMissingFields(t *testing.T) {
	// missing scopeId
	ea := getEnabledAnalytics(t, map[string]interface{}{
		"pubstack": map[string]interface{}{
			"enabled":   true,
			"intakeUrl": "https://example.com",
		},
	})
	if _, ok := ea["pubstack"]; ok {
		t.Fatalf("did not expect pubstack without scopeId")
	}
	// missing intakeUrl
	ea2 := getEnabledAnalytics(t, map[string]interface{}{
		"pubstack": map[string]interface{}{
			"enabled": true,
			"scopeId": "s1",
		},
	})
	if _, ok := ea2["pubstack"]; ok {
		t.Fatalf("did not expect pubstack without intakeUrl")
	}
}

func TestNew_WithAgmaConfig(t *testing.T) {
	ea := getEnabledAnalytics(t, map[string]interface{}{
		"agma": map[string]interface{}{
			"enabled":  true,
			"endpoint": "https://agma.example.com/collect",
			"buffers": map[string]interface{}{
				"eventCount": 50,
				"bufferSize": "32KB",
				"timeout":    "2s",
			},
		},
	})
	if _, ok := ea["agma"]; !ok {
		t.Fatalf("expected agma module to be initialized")
	}
}

func TestNew_AgmaMissingOrDisabled(t *testing.T) {
	// missing endpoint
	ea := getEnabledAnalytics(t, map[string]interface{}{
		"agma": map[string]interface{}{
			"enabled": true,
		},
	})
	if _, ok := ea["agma"]; ok {
		t.Fatalf("did not expect agma without endpoint")
	}
	// disabled
	ea2 := getEnabledAnalytics(t, map[string]interface{}{
		"agma": map[string]interface{}{
			"enabled":  false,
			"endpoint": "https://agma.example.com",
		},
	})
	if _, ok := ea2["agma"]; ok {
		t.Fatalf("did not expect agma when disabled")
	}
}
