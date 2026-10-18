package config

import "testing"

func TestBrowserSessionProductionSettings(t *testing.T) {
	for key, value := range map[string]string{
		"REDIS_PASSWORD": "test", "TASKS_URL": "http://tasks:8080", "FILES_URL": "http://files:8080", "INTERNAL_TOKEN": "test", "METRICS_TOKEN": "m", "OIDC_ISSUER": "https://auth.example.test/realms/test", "OIDC_INTERNAL_URL": "", "APP_ENV": "production",
	} {
		t.Setenv(key, value)
	}
	for _, tc := range []struct {
		origin, secure string
		valid          bool
	}{
		{"https://app.example.test", "true", true},
		{"http://app.example.test", "true", false},
		{"https://app.example.test", "false", false},
		{"https://app.example.test/path", "true", false},
		{"https://user:password@app.example.test", "true", false},
		{"https://app.example.test?query=yes", "true", false},
	} {
		t.Setenv("APP_ORIGIN", tc.origin)
		t.Setenv("SESSION_COOKIE_SECURE", tc.secure)
		_, err := Load()
		if (err == nil) != tc.valid {
			t.Errorf("origin=%q secure=%q: %v", tc.origin, tc.secure, err)
		}
	}
}

func TestProductionRequiresMetricsToken(t *testing.T) {
	for key, value := range map[string]string{
		"REDIS_PASSWORD": "test", "TASKS_URL": "http://tasks:8080", "FILES_URL": "http://files:8080", "INTERNAL_TOKEN": "test",
		"OIDC_ISSUER": "https://auth.example.test/realms/test", "APP_ENV": "production", "APP_ORIGIN": "https://app.example.test", "SESSION_COOKIE_SECURE": "true",
	} {
		t.Setenv(key, value)
	}
	if _, err := Load(); err == nil {
		t.Fatal("production без METRICS_TOKEN должен быть отклонён")
	}
	t.Setenv("METRICS_ENABLED", "false")
	if _, err := Load(); err != nil {
		t.Fatalf("метрики выключены: %v", err)
	}
}
