package apidocs

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsServed(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	Register(r, nil, "https://id.example.test/realms/hackathon")
	for path, want := range map[string]string{
		"/api/docs":                             "SwaggerUIBundle",
		"/api/openapi.yaml":                     "openapi:",
		"/api/docs/assets/swagger-ui.css":       "swagger-ui",
		"/api/docs/assets/swagger-ui-bundle.js": "",
		"/api/openapi/gateway.yaml":             "Gateway",
		"/api/docs/specs.json":                  "Все сервисы",
		"/api/docs/oauth2-redirect.html":        "oauth2",
	} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), want) {
			t.Errorf("%s: %d", path, w.Code)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/api/docs/assets/nope.js", nil))
	if w.Code != 404 {
		t.Errorf("несуществующий файл: %d", w.Code)
	}
}

// Копия спецификации в gateway не должна отставать от источника правды в packages/contracts.
func TestSpecIsInSync(t *testing.T) {
	source, err := os.ReadFile("../../../../packages/contracts/openapi.yaml")
	if err != nil {
		t.Skip("исходная спецификация недоступна (сборка вне монорепы)")
	}
	embedded, _ := files.ReadFile("openapi.yaml")
	if string(source) != string(embedded) {
		t.Fatal("openapi.yaml устарел: выполните make docs-sync")
	}
}

// Спецификация сервиса берётся у него на лету; незапущенный сервис даёт понятную 502.
func TestServiceSpecProxy(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/openapi.yaml" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("openapi: 3.0.3\ninfo: {title: tasks}\nurl: {{OIDC_ISSUER}}/auth\n"))
	}))
	defer upstream.Close()
	tasksURL, _ := url.Parse(upstream.URL)
	downURL, _ := url.Parse("http://127.0.0.1:1")
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	Register(r, map[string]*url.URL{"tasks": tasksURL, "files": downURL, "audit": nil}, "https://id.example.test/realms/hackathon")
	srv := httptest.NewServer(r)
	defer srv.Close()
	get := func(path string) (int, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	// Адрес Keycloak подставляется и в спецификации, взятые у сервисов.
	if code, body := get("/api/openapi/tasks.yaml"); code != 200 || !strings.Contains(body, "title: tasks") || !strings.Contains(body, "https://id.example.test/realms/hackathon/auth") {
		t.Fatalf("%d %s", code, body)
	}
	if code, _ := get("/api/openapi/files.yaml"); code != 502 {
		t.Fatalf("незапущенный сервис: %d", code)
	}
	for _, path := range []string{"/api/openapi/audit.yaml", "/api/openapi/unknown.yaml", "/api/openapi/tasks.json"} {
		if code, _ := get(path); code != 404 {
			t.Errorf("%s: %d", path, code)
		}
	}
	_, list := get("/api/docs/specs.json")
	if !strings.Contains(list, "tasks.yaml") || strings.Contains(list, "audit") {
		t.Fatalf("список спецификаций: %s", list)
	}
}

// В отдаваемых спецификациях не остаётся заглушки адреса Keycloak.
func TestIssuerSubstituted(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	Register(r, nil, "https://id.example.test/realms/hackathon")
	for _, path := range []string{"/api/openapi.yaml", "/api/openapi/gateway.yaml"} {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		body := w.Body.String()
		if strings.Contains(body, "{{OIDC_ISSUER}}") || !strings.Contains(body, "https://id.example.test/realms/hackathon/protocol/openid-connect/auth") {
			t.Errorf("%s: адрес Keycloak не подставлен", path)
		}
	}
}
