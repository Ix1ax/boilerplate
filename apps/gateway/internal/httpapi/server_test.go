package httpapi

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"example.com/hackathon/apps/gateway/internal/auth"
	"github.com/redis/go-redis/v9"
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

func TestRedisFailurePreservesBrowserSession(t *testing.T) {
	cache := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := cache.Close(); err != nil {
		t.Fatal(err)
	}
	sessions := auth.NewSessions(cache, nil, "https://id.example.test", "", "web", "https://app.example.test", true)
	router := New(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil, sessions)
	for _, path := range []string{"/api/v1/me", "/auth/logout"} {
		method := "GET"
		if path == "/auth/logout" {
			method = "POST"
		}
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Origin", "https://app.example.test")
		req.AddCookie(&http.Cookie{Name: "starter_session", Value: "existing-session"})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != 503 {
			t.Fatalf("%s: ожидался 503, получен %d", path, response.Code)
		}
		if len(response.Result().Cookies()) != 0 {
			t.Fatalf("%s: сбой Redis изменил cookie", path)
		}
	}
}
