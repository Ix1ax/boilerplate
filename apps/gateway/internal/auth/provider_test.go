package auth

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestInternalTransportPreservesIssuerAndQuery(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/realms/hackathon/protocol/openid-connect/certs" || r.URL.Query().Get("x") != "1" {
			t.Errorf("изменился путь: %s", r.URL)
		}
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	public, _ := url.Parse("https://auth.example/realms/hackathon")
	internal, _ := url.Parse(upstream.URL + "/realms/hackathon")
	client := http.Client{Transport: internalTransport{public: public, internal: internal, base: http.DefaultTransport}}
	request, _ := http.NewRequest("GET", public.String()+"/protocol/openid-connect/certs?x=1", nil)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if request.URL.Host != "auth.example" {
		t.Fatal("исходный запрос изменён")
	}
}
