package api

import (
	"net/http/httptest"
	"testing"
)

func TestAutomationOriginAllowlist(t *testing.T) {
	for _, tc := range []struct {
		origin string
		want   bool
	}{
		{"http://localhost:3000", true},
		{"http://127.0.0.1:5173", true},
		{"https://localhost:3000", true},
		{"https://example.com", false},
	} {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Origin", tc.origin)
		if got := upgrader.CheckOrigin(r); got != tc.want {
			t.Errorf("origin %q: got %v want %v", tc.origin, got, tc.want)
		}
	}
}
