package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	srv := httptest.NewServer(New(slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer srv.Close()

	tests := []struct {
		path     string
		status   int
		contains string
	}{
		{"/", http.StatusOK, "<h1>unjbr</h1>"},
		{"/healthz", http.StatusOK, "ok"},
		{"/static/style.css", http.StatusOK, "font-family"},
		{"/missing", http.StatusNotFound, ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tt.path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.status)
			}
			if !strings.Contains(string(body), tt.contains) {
				t.Errorf("body does not contain %q:\n%s", tt.contains, body)
			}
		})
	}
}
