package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerServesPWAAssets(t *testing.T) {
	handler := Handler()
	paths := []string{
		"/manifest.webmanifest",
		"/sw.js",
		"/icons/icon-192.png",
		"/icons/icon-512.png",
		"/icons/icon-maskable-512.png",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("GET %s returned %d", path, response.Code)
			}
			if response.Body.Len() == 0 {
				t.Fatalf("GET %s returned an empty body", path)
			}
		})
	}
}

func TestManifestIsValidJSON(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/manifest.webmanifest", nil)
	response := httptest.NewRecorder()
	Handler().ServeHTTP(response, request)

	var manifest map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &manifest); err != nil {
		t.Fatalf("decoding manifest: %v", err)
	}
	if manifest["display"] != "standalone" {
		t.Fatalf("display = %v, want standalone", manifest["display"])
	}
}
