package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusNoContent(t *testing.T) {
	handler := NewStatusEndpoint("")
	w := httptest.NewRecorder()
	handler(w, nil, nil)
	if w.Code != http.StatusNoContent {
		t.Errorf("Bad code for empty content. Expected %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestStatusWithContent(t *testing.T) {
	handler := NewStatusEndpoint("ready")
	w := httptest.NewRecorder()
	handler(w, nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("Bad code for empty content. Expected %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != "ready" {
		t.Errorf("Bad status body. Expected %s, got %s", "ready", w.Body.String())
	}
}
