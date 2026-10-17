package auth

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestSafeNext(t *testing.T) {
	for _, value := range []string{"https://other.test/", "//other.test/", "/\\other.test", "/auth/logout", "javascript:alert(1)", "/%0d%0aLocation:x", ""} {
		if got := safeNext(value); got != "/dashboard/tasks" {
			t.Errorf("пропущен внешний или недопустимый адрес %q: %q", value, got)
		}
	}
	if got := safeNext("/dashboard/files?offset=20#list"); got != "/dashboard/files?offset=20#list" {
		t.Fatal(got)
	}
}

func TestCookieCSRF(t *testing.T) {
	s := &Sessions{origin: "https://app.example.test"}
	router := gin.New()
	router.Use(s.CSRF())
	router.POST("/api/v1/tasks", func(c *gin.Context) { c.Status(204) })
	for _, tc := range []struct {
		origin, bearer string
		status         int
	}{
		{"", "", 403}, {"https://other.test", "", 403}, {s.origin, "", 204}, {"", "Bearer test", 204},
	} {
		req := httptest.NewRequest("POST", "/api/v1/tasks", nil)
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("Authorization", tc.bearer)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != tc.status {
			t.Errorf("Origin=%q: %d", tc.origin, response.Code)
		}
	}
}
