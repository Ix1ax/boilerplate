// Package apidocs отдаёт Swagger UI и OpenAPI-описания из самого бинарника: документация
// доступна без интернета и без отдельного контейнера.
//
// В интерфейсе есть список спецификаций: «Все сервисы» (общая, собрана make openapi),
// gateway (встроена) и каждый подключённый сервис — её gateway берёт у сервиса на лету
// (GET <сервис>/openapi.yaml), поэтому показывается ровно та версия, что сейчас запущена.
//
// Вход в Swagger идёт через Keycloak (схема keycloak, клиент hackathon-swagger с PKCE).
// В спецификациях вместо адреса Keycloak стоит {{OIDC_ISSUER}} — gateway подставляет свой
// OIDC_ISSUER при отдаче, поэтому одни и те же файлы работают локально и на сервере.
package apidocs

import (
	"bytes"
	"embed"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"time"

	gatewayapi "example.com/hackathon/server/gateway/api"
	"github.com/gin-gonic/gin"
)

//go:embed index.html openapi.yaml assets
var files embed.FS

type specLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Register добавляет публичные маршруты:
//   - GET /api/docs — Swagger UI;
//   - GET /api/openapi.yaml — общая спецификация (копия packages/contracts/openapi.yaml);
//   - GET /api/openapi/<имя>.yaml — спецификация gateway или сервиса;
//   - GET /api/docs/specs.json — список спецификаций для переключателя в интерфейсе.
//
// services — адреса подключённых сервисов по имени; nil-адрес пропускается.
// issuer — публичный адрес realm Keycloak (OIDC_ISSUER) для входа из Swagger.
func Register(r *gin.Engine, services map[string]*url.URL, issuer string) {
	withIssuer := func(spec []byte) []byte {
		return bytes.ReplaceAll(spec, []byte("{{OIDC_ISSUER}}"), []byte(issuer))
	}
	assets, _ := fs.Sub(files, "assets")
	noCache := func(c *gin.Context) { c.Header("Cache-Control", "no-cache") }
	links := []specLink{{"Все сервисы", "/api/openapi.yaml"}, {"gateway", "/api/openapi/gateway.yaml"}}
	names := make([]string, 0, len(services))
	for name, u := range services {
		if u != nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		links = append(links, specLink{name, "/api/openapi/" + name + ".yaml"})
	}
	list, _ := json.Marshal(links)

	r.GET("/api/docs", noCache, func(c *gin.Context) {
		page, _ := files.ReadFile("index.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	})
	r.GET("/api/docs/specs.json", noCache, func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", list)
	})
	r.GET("/api/openapi.yaml", noCache, func(c *gin.Context) {
		spec, _ := files.ReadFile("openapi.yaml")
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", withIssuer(spec))
	})
	proxies := map[string]*httputil.ReverseProxy{}
	for _, name := range names {
		target := services[name]
		proxies[name] = &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(target)
				pr.Out.URL.Path = "/openapi.yaml"
				pr.Out.URL.RawQuery = ""
			},
			Transport: &http.Transport{ResponseHeaderTimeout: 5 * time.Second},
			ModifyResponse: func(resp *http.Response) error {
				body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
				_ = resp.Body.Close()
				if err != nil {
					return err
				}
				body = withIssuer(body)
				resp.Body = io.NopCloser(bytes.NewReader(body))
				resp.ContentLength = int64(len(body))
				resp.Header.Set("Content-Length", strconv.Itoa(len(body)))
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(`{"error":{"code":"upstream_unavailable","message":"Сервис не запущен — его спецификация недоступна"}}`))
			},
		}
	}
	r.GET("/api/openapi/:file", noCache, func(c *gin.Context) {
		name := c.Param("file")
		if len(name) < 6 || name[len(name)-5:] != ".yaml" {
			c.Status(http.StatusNotFound)
			return
		}
		name = name[:len(name)-5]
		if name == "gateway" {
			c.Data(http.StatusOK, "application/yaml; charset=utf-8", withIssuer(gatewayapi.Spec))
			return
		}
		proxy, ok := proxies[name]
		if !ok {
			c.Status(http.StatusNotFound)
			return
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	})
	// Страница, на которую Keycloak возвращает после входа в Swagger (адрес — в клиенте hackathon-swagger).
	r.GET("/api/docs/oauth2-redirect.html", noCache, func(c *gin.Context) {
		page, _ := files.ReadFile("assets/oauth2-redirect.html")
		c.Data(http.StatusOK, "text/html; charset=utf-8", page)
	})
	r.GET("/api/docs/assets/:name", func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age=3600")
		c.FileFromFS(c.Param("name"), http.FS(assets))
	})
}
