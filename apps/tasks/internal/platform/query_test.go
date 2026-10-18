package platform

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func parse(t *testing.T, url string) (ListQuery, error) {
	gin.SetMode(gin.ReleaseMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", url, nil)
	return ParseList(c, ListSpec{
		Filters:     map[string]string{"status": "t.status"},
		Sorts:       map[string]string{"created": "t.created_at", "title": "t.title"},
		DefaultSort: "-created",
		Search:      []string{"t.title"},
	}, 2)
}

func TestParseList(t *testing.T) {
	q, err := parse(t, "/x?status=a,b&q=100%25_&sort=title&limit=5")
	if err != nil {
		t.Fatal(err)
	}
	if q.Where != " AND t.status = ANY($2) AND (t.title ILIKE $3)" || q.OrderBy != " ORDER BY t.title ASC" || q.Limit != 5 {
		t.Fatalf("%+v", q)
	}
	if q.Args[1] != `%100\%\_%` {
		t.Fatalf("спецсимволы LIKE не экранированы: %v", q.Args[1])
	}
	if q, _ = parse(t, "/x"); q.OrderBy != " ORDER BY t.created_at DESC" || q.Where != "" {
		t.Fatalf("%+v", q)
	}
	for _, bad := range []string{"/x?sort=password", "/x?sort=created%3Bdrop", "/x?limit=0", "/x?q=" + strings.Repeat("я", 101)} {
		if _, err := parse(t, bad); err == nil {
			t.Errorf("%s: ожидалась ошибка", bad)
		}
	}
	// Неизвестные параметры игнорируются, а не попадают в SQL.
	if q, _ = parse(t, "/x?owner_id=1;drop"); q.Where != "" {
		t.Fatalf("%+v", q)
	}
}
