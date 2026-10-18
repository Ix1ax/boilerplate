package ollama

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fake(t *testing.T, h http.HandlerFunc) *Client {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return New(srv.URL, "qwen2.5:1.5b", 5*time.Second)
}

func TestStreamParsesNDJSON(t *testing.T) {
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"response":"При","done":false}` + "\n" + `{"response":"вет","done":false}` + "\n" + `{"response":"","done":true}` + "\n"))
	})
	var got strings.Builder
	if err := c.Stream(context.Background(), Request{Prompt: "x"}, func(s string) error { got.WriteString(s); return nil }); err != nil || got.String() != "Привет" {
		t.Fatalf("%q %v", got.String(), err)
	}
}

func TestGenerateAndJSONFormat(t *testing.T) {
	var body string
	c := fake(t, func(w http.ResponseWriter, r *http.Request) {
		b := make([]byte, 512)
		n, _ := r.Body.Read(b)
		body = string(b[:n])
		_, _ = w.Write([]byte(`{"response":"{\"ok\":true}","done":true}`))
	})
	out, err := c.Generate(context.Background(), Request{Prompt: "p", JSON: true})
	if err != nil || out != `{"ok":true}` || !strings.Contains(body, `"format":"json"`) {
		t.Fatalf("%q %v %s", out, err, body)
	}
}

func TestUnavailableAndStatus(t *testing.T) {
	c := New("http://127.0.0.1:1", "m", time.Second)
	if _, err := c.Generate(context.Background(), Request{Prompt: "p"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ожидалась ErrUnavailable: %v", err)
	}
	if c.Status(context.Background()).Available {
		t.Fatal("Ollama не должна быть доступна")
	}
	ok := fake(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen2.5:1.5b"}]}`))
	})
	if st := ok.Status(context.Background()); !st.Available || !st.ModelInstalled {
		t.Fatalf("%+v", st)
	}
}
