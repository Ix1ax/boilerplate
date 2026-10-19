package auth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func demoRouter() (*gin.Engine, []DemoAccount) {
	s := &Sessions{origin: "https://app.example.test"}
	accounts := []DemoAccount{{Key: "anna", Name: "Анна", Role: "Пользователь", email: "anna@example.test", pass: "secret-pass"}}
	r := gin.New()
	s.RegisterDemo(r, accounts)
	return r, accounts
}

func TestDemoListHidesCredentials(t *testing.T) {
	r, _ := demoRouter()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/auth/demo", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"key":"anna"`) {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), "secret-pass") || strings.Contains(w.Body.String(), "anna@example.test") {
		t.Fatal("в списке демо-аккаунтов утекли почта или пароль")
	}
}

func TestDemoLoginRejectsBadRequests(t *testing.T) {
	r, _ := demoRouter()
	for _, tc := range []struct {
		name, origin, key string
		status            int
	}{
		{"без Origin", "", "anna", 403},
		{"чужой Origin", "https://evil.test", "anna", 403},
		{"неизвестный аккаунт", "https://app.example.test", "nobody", 404},
	} {
		req := httptest.NewRequest("POST", "/auth/demo/"+tc.key, nil)
		req.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status {
			t.Errorf("%s: %d", tc.name, w.Code)
		}
	}
}

func TestDemoAccountsFromEnvSkipsIncomplete(t *testing.T) {
	t.Setenv("DEMO_EMAIL", "anna@example.test")
	t.Setenv("DEMO_PASSWORD", "x")
	t.Setenv("ADMIN_DEMO_EMAIL", "admin@example.test")
	t.Setenv("ADMIN_DEMO_PASSWORD", "")
	for _, name := range []string{"SECOND_DEMO_EMAIL", "ORG_DEMO_EMAIL", "ORG_PENDING_DEMO_EMAIL", "ORG_REJECTED_DEMO_EMAIL"} {
		t.Setenv(name, "")
	}
	got := DemoAccountsFromEnv()
	if len(got) != 1 || got[0].Key != "anna" {
		t.Fatalf("%+v", got)
	}
}
