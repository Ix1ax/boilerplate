package apidocs

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsServed(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	Register(r)
	for path, want := range map[string]string{
		"/api/docs":                             "SwaggerUIBundle",
		"/api/openapi.yaml":                     "openapi:",
		"/api/docs/assets/swagger-ui.css":       "swagger-ui",
		"/api/docs/assets/swagger-ui-bundle.js": "",
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
