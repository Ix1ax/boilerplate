package llm

import (
	"bufio"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/apps/llm/internal/ollama"
	"github.com/gin-gonic/gin"
)

type fakeModel struct {
	block chan struct{}
	err   error
}

func (f *fakeModel) Model() string { return "fake" }
func (f *fakeModel) Generate(ctx context.Context, r ollama.Request) (string, error) {
	if f.block != nil {
		<-f.block
	}
	return "ответ", f.err
}
func (f *fakeModel) Stream(ctx context.Context, r ollama.Request, on func(string) error) error {
	if f.err != nil {
		return f.err
	}
	for _, p := range []string{"при", "вет"} {
		if err := on(p); err != nil {
			return err
		}
	}
	return nil
}
func (f *fakeModel) Status(context.Context) ollama.Status { return ollama.Status{} }

func TestValidation(t *testing.T) {
	s := New(&fakeModel{}, 1, 10)
	bad := 3.0
	for _, r := range []ollama.Request{{Prompt: "  "}, {Prompt: strings.Repeat("я", 11)}, {Prompt: "ok", Temperature: &bad}} {
		if _, err := s.Generate(context.Background(), r); !errors.Is(err, ErrPrompt) {
			t.Errorf("%+v: %v", r, err)
		}
	}
}

func TestBusyWhenSlotsTaken(t *testing.T) {
	m := &fakeModel{block: make(chan struct{})}
	s := New(m, 1, 100)
	done := make(chan struct{})
	go func() { _, _ = s.Generate(context.Background(), ollama.Request{Prompt: "a"}); close(done) }()
	deadline := time.Now().Add(time.Second)
	for len(s.slots) == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if _, err := s.Generate(context.Background(), ollama.Request{Prompt: "b"}); !errors.Is(err, ErrBusy) {
		t.Fatalf("ожидалась ErrBusy: %v", err)
	}
	close(m.block)
	<-done
}

func server(m Completer) *httptest.Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	Register(r.Group("/api/v1"), New(m, 2, 100), time.Second)
	return httptest.NewServer(r)
}

func TestStreamSSE(t *testing.T) {
	srv := server(&fakeModel{})
	defer srv.Close()
	resp, err := http.Post(srv.URL+"/api/v1/llm/stream", "application/json", strings.NewReader(`{"prompt":"hi"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var lines []string
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		if sc.Text() != "" {
			lines = append(lines, sc.Text())
		}
	}
	got := strings.Join(lines, "|")
	if resp.Header.Get("Content-Type") != "text/event-stream" || !strings.Contains(got, `data: {"t":"при"}`) || !strings.HasSuffix(got, "event: done|data: {}") {
		t.Fatalf("%s %s", resp.Header.Get("Content-Type"), got)
	}
}

func TestUnavailableMapsTo503BeforeStream(t *testing.T) {
	srv := server(&fakeModel{err: ollama.ErrUnavailable})
	defer srv.Close()
	for _, p := range []string{"stream", "generate"} {
		resp, err := http.Post(srv.URL+"/api/v1/llm/"+p, "application/json", strings.NewReader(`{"prompt":"hi"}`))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 503 {
			t.Errorf("%s: %d", p, resp.StatusCode)
		}
	}
}
