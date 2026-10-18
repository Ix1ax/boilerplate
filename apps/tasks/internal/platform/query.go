package platform

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListSpec описывает, что клиенту разрешено в списке. Имена колонок берутся только отсюда,
// значения из запроса уходят в SQL параметрами — так фильтры не открывают SQL-инъекцию.
type ListSpec struct {
	Filters     map[string]string // параметр запроса → колонка (равенство; «a,b» → IN)
	Sorts       map[string]string // значение sort → колонка; «-created» — по убыванию
	DefaultSort string            // например "-created"
	// Search — колонки для поиска по подстроке без учёта регистра (параметр q).
	// Для больших таблиц добавьте индекс pg_trgm: CREATE INDEX … USING gin (title gin_trgm_ops).
	Search []string
}

// ListQuery — разобранный запрос: готовые WHERE (с ведущим AND) и ORDER BY с аргументами.
type ListQuery struct {
	Where   string
	OrderBy string
	Args    []any
	Limit   int
	Offset  int
}

// ParseList разбирает limit/offset, фильтры, сортировку и поиск. firstArg — номер первого
// плейсхолдера после уже занятых (обычно 2, если $1 — владелец).
func ParseList(c *gin.Context, spec ListSpec, firstArg int) (ListQuery, error) {
	limit, offset, ok := ParsePagination(c.DefaultQuery("limit", "20"), c.DefaultQuery("offset", "0"))
	if !ok {
		return ListQuery{}, fmt.Errorf("limit должен быть от 1 до 100, offset — от 0 до 100000")
	}
	q := ListQuery{Limit: limit, Offset: offset}
	n := firstArg
	arg := func(v any) string {
		q.Args = append(q.Args, v)
		n++
		return "$" + strconv.Itoa(n-1)
	}
	var where []string
	for param, column := range spec.Filters {
		raw := strings.TrimSpace(c.Query(param))
		if raw == "" {
			continue
		}
		values := strings.Split(raw, ",")
		if len(values) > 20 {
			return ListQuery{}, fmt.Errorf("Слишком много значений в фильтре %s", param)
		}
		if len(values) == 1 {
			where = append(where, column+" = "+arg(values[0]))
		} else {
			where = append(where, column+" = ANY("+arg(values)+")")
		}
	}
	if s := strings.TrimSpace(c.Query("q")); s != "" && len(spec.Search) > 0 {
		if len([]rune(s)) > 100 {
			return ListQuery{}, fmt.Errorf("Поисковый запрос длиннее 100 символов")
		}
		p := arg("%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(s) + "%")
		var or []string
		for _, column := range spec.Search {
			or = append(or, column+" ILIKE "+p)
		}
		where = append(where, "("+strings.Join(or, " OR ")+")")
	}
	if len(where) > 0 {
		q.Where = " AND " + strings.Join(where, " AND ")
	}
	sort := c.DefaultQuery("sort", spec.DefaultSort)
	column, ok := spec.Sorts[strings.TrimPrefix(sort, "-")]
	if !ok {
		return ListQuery{}, fmt.Errorf("Недопустимая сортировка %q", sort)
	}
	dir := "ASC"
	if strings.HasPrefix(sort, "-") {
		dir = "DESC"
	}
	q.OrderBy = " ORDER BY " + column + " " + dir
	return q, nil
}
