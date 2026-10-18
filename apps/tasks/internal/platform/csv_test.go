package platform

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSVEscapesFormulasAndAddsBOM(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	out := NewCSV(c, "отчёт.csv", []string{"a", "b"})
	_ = out.Row("=HYPERLINK(\"x\")", "обычный; текст")
	_ = out.Flush()
	body := w.Body.String()
	if !strings.HasPrefix(body, "\uFEFFa;b\n") || !strings.Contains(body, `"'=HYPERLINK(""x"")"`) || !strings.Contains(body, `"обычный; текст"`) {
		t.Fatalf("%q", body)
	}
	if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
		t.Fatal("нет Content-Disposition")
	}
}
