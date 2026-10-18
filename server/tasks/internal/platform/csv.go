package platform

import (
	"encoding/csv"
	"mime"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSVWriter пишет выгрузку прямо в ответ, без накопления в памяти.
// Формат рассчитан на Excel с русской локалью: UTF-8 с BOM и разделитель «;».
type CSVWriter struct{ w *csv.Writer }

// NewCSV выставляет заголовки ответа для скачивания и пишет BOM и строку заголовков таблицы.
func NewCSV(c *gin.Context, filename string, header []string) *CSVWriter {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
	_, _ = c.Writer.WriteString("\uFEFF")
	w := csv.NewWriter(c.Writer)
	w.Comma = ';'
	out := &CSVWriter{w: w}
	_ = out.Row(header...)
	return out
}

// Row экранирует ячейки, похожие на формулы (=, +, -, @): иначе Excel выполнит их при открытии.
func (c *CSVWriter) Row(cells ...string) error {
	for i, v := range cells {
		if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
			cells[i] = "'" + v
		}
	}
	return c.w.Write(cells)
}

func (c *CSVWriter) Flush() error {
	c.w.Flush()
	return c.w.Error()
}
