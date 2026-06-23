package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRouterServesConsoleBuild(t *testing.T) {
	consoleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(consoleDir, "index.html"), []byte("<html>OpenARD Console</html>"), 0o600); err != nil {
		t.Fatalf("write console index: %v", err)
	}
	assetsDir := filepath.Join(consoleDir, "assets")
	if err := os.Mkdir(assetsDir, 0o700); err != nil {
		t.Fatalf("create assets dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte("console.log('openard')"), 0o600); err != nil {
		t.Fatalf("write console asset: %v", err)
	}

	router := NewRouterWithOptions(nil, Options{ConsoleDir: consoleDir})

	tests := []struct {
		name       string
		path       string
		statusCode int
		body       string
	}{
		{
			name:       "console root",
			path:       "/console",
			statusCode: http.StatusOK,
			body:       "OpenARD Console",
		},
		{
			name:       "console asset",
			path:       "/console/assets/app.js",
			statusCode: http.StatusOK,
			body:       "console.log('openard')",
		},
		{
			name:       "console route fallback",
			path:       "/console/catalog",
			statusCode: http.StatusOK,
			body:       "OpenARD Console",
		},
		{
			name:       "missing asset",
			path:       "/console/assets/missing.js",
			statusCode: http.StatusNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			router.ServeHTTP(response, request)
			if response.Code != test.statusCode {
				t.Fatalf("expected status %d, got %d", test.statusCode, response.Code)
			}
			if test.body != "" && !strings.Contains(response.Body.String(), test.body) {
				t.Fatalf("expected body to contain %q, got %q", test.body, response.Body.String())
			}
		})
	}
}
