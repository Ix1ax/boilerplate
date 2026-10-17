package httpapi

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
)

func TestHealthAuthAndUnknownRoutes(t *testing.T) {
	router := New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	for _, tc := range []struct {
		path   string
		status int
	}{{"/healthz", 200}, {"/readyz", 503}, {"/api/v1/me", 401}, {"/api/nope", 404}} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", tc.path, nil))
		if response.Code != tc.status {
			t.Errorf("%s: %d", tc.path, response.Code)
		}
		if response.Header().Get("X-Request-ID") == "" || response.Header().Get("Cache-Control") != "no-store" {
			t.Error("В ответе нет обязательных диагностических заголовков")
		}
	}
}
