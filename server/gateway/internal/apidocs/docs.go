// Package apidocs отдаёт Swagger UI и OpenAPI-описание из самого бинарника: документация
// доступна без интернета и без отдельного контейнера. Файлы embed-ятся при сборке.
package apidocs

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed index.html openapi.yaml assets
var files embed.FS

// Register добавляет публичные маршруты /api/docs, /api/docs/assets/* и /api/openapi.yaml.
// Спецификация — копия packages/contracts/openapi.yaml (make docs-sync, CI проверяет совпадение).
func Register(r *gin.Engine) {
	assets, _ := fs.Sub(files, "assets")
	cached := func(c *gin.Context) { c.Header("Cache-Control", "public, max-age=3600") }
	r.GET("/api/docs", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		page, _ := files.ReadFile("index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	})
	r.GET("/api/openapi.yaml", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		spec, _ := files.ReadFile("openapi.yaml")
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", spec)
	})
	r.GET("/api/docs/assets/:name", cached, func(c *gin.Context) {
		c.FileFromFS(c.Param("name"), http.FS(assets))
	})
}
