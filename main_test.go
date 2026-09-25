package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func TestDashboardCleanURLsServeGeneratedHTML(t *testing.T) {
	dashboardFiles := fstest.MapFS{
		"settings.html": {Data: []byte("settings page")},
		"app.js":        {Data: []byte("app asset")},
	}
	handler := http.StripPrefix("/dashboard", http.FileServer(http.FS(cleanURLFS{dashboardFiles})))

	tests := []struct {
		path string
		body string
	}{
		{path: "/dashboard/settings", body: "settings page"},
		{path: "/dashboard/app.js", body: "app asset"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
			}
			if response.Body.String() != test.body {
				t.Fatalf("expected body %q, got %q", test.body, response.Body.String())
			}
		})
	}
}
