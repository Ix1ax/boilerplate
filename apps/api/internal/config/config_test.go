package config

import "testing"

func TestBrowserSessionProductionSettings(t *testing.T) {
	for key, value := range map[string]string{
		"DATABASE_URL": "postgres://localhost/test", "REDIS_PASSWORD": "test", "S3_ENDPOINT": "http://localhost:8333", "S3_ACCESS_KEY": "test", "S3_SECRET_KEY": "test", "S3_BUCKET": "test", "OIDC_ISSUER": "https://auth.example.test/realms/test", "OIDC_INTERNAL_URL": "", "APP_ENV": "production",
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
