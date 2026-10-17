package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/go-jose/go-jose/v4"
)

func TestTokenValidation(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, nil)
	if err != nil {
		t.Fatal(err)
	}
	verifier := oidc.NewVerifier("https://issuer.test", &oidc.StaticKeySet{PublicKeys: []crypto.PublicKey{&key.PublicKey}}, &oidc.Config{ClientID: "hackathon-api"})

	for _, tc := range []struct {
		name   string
		change func(map[string]any)
		status int
	}{
		{"валидный токен", func(map[string]any) {}, 200},
		{"другой audience", func(c map[string]any) { c["aud"] = "other" }, 401},
		{"другой issuer", func(c map[string]any) { c["iss"] = "https://other.test" }, 401},
		{"истёкший токен", func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }, 401},
		{"нет subject", func(c map[string]any) { delete(c, "sub") }, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := map[string]any{"iss": "https://issuer.test", "aud": "hackathon-api", "sub": "user-1", "exp": time.Now().Add(time.Hour).Unix(), "email": "user@example.test", "email_verified": true}
			tc.change(claims)
			payload, err := json.Marshal(claims)
			if err != nil {
				t.Fatal(err)
			}
			signed, err := signer.Sign(payload)
			if err != nil {
				t.Fatal(err)
			}
			token, err := signed.CompactSerialize()
			if err != nil {
				t.Fatal(err)
			}
			checkRequest(t, verifier, "Bearer "+token, tc.status)
		})
	}
	for _, header := range []string{"", "Basic abc", "Bearer", "Bearer invalid", "Bearer a b"} {
		checkRequest(t, verifier, header, 401)
	}
}

func checkRequest(t *testing.T, verifier Verifier, header string, status int) {
	t.Helper()
	r := gin.New()
	r.GET("/", Middleware(verifier), func(c *gin.Context) {
		if _, ok := c.Get("identity"); !ok {
			t.Error("нет проверенной идентичности")
		}
		c.Status(200)
	})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", header)
	response := httptest.NewRecorder()
	r.ServeHTTP(response, req)
	if response.Code != status {
		t.Fatalf("ожидали %d, получили %d", status, response.Code)
	}
}
