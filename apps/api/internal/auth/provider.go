package auth

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// В браузере issuer публичный, а контейнер API получает discovery и JWKS по внутренней сети.
// Проверка issuer токена при этом остаётся включённой.
type Verifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

type internalTransport struct {
	public, internal *url.URL
	base             http.RoundTripper
}

func (t internalTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Scheme != t.public.Scheme || request.URL.Host != t.public.Host || !strings.HasPrefix(request.URL.Path, t.public.Path+"/") {
		return t.base.RoundTrip(request)
	}
	cloned := request.Clone(request.Context())
	u := *request.URL
	u.Scheme = t.internal.Scheme
	u.Host = t.internal.Host
	u.Path = t.internal.Path + strings.TrimPrefix(u.Path, t.public.Path)
	cloned.URL = &u
	// Keycloak использует этот Host для маршрутизации; issuer задаётся его KC_HOSTNAME.
	cloned.Host = u.Host
	return t.base.RoundTrip(cloned)
}
func NewVerifier(ctx context.Context, issuer, clientID, internalURL string) (Verifier, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	if internalURL != "" {
		public, err := url.Parse(issuer)
		if err != nil {
			return nil, err
		}
		internal, err := url.Parse(internalURL)
		if err != nil {
			return nil, err
		}
		client.Transport = internalTransport{public: public, internal: internal, base: http.DefaultTransport}
	}
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, client), issuer)
	if err != nil {
		return nil, err
	}
	return provider.Verifier(&oidc.Config{ClientID: clientID}), nil
}
