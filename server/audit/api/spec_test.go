package api_test

import (
	"regexp"
	"strings"
	"testing"

	"example.com/hackathon/server/audit/api"
	"example.com/hackathon/server/audit/internal/audit"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// register подключает все маршруты сервиса так же, как cmd/audit/main.go (без зависимостей).
func register(api *gin.RouterGroup) {
	audit.Register(api, nil)
}

var httpMethods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true, "head": true, "options": true}

// Маршруты сервиса и openapi.yaml должны совпадать в обе стороны: новый маршрут без описания
// в спецификации (или описание без маршрута) роняет тест — Swagger не отстаёт от кода.
func TestRoutesMatchSpec(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	register(r.Group("/api/v1"))
	routes := map[string]bool{}
	param := regexp.MustCompile(`:([A-Za-z_]+)`)
	for _, rt := range r.Routes() {
		routes[strings.ToLower(rt.Method)+" "+param.ReplaceAllString(rt.Path, "{$1}")] = true
	}
	var spec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}
	if err := yaml.Unmarshal(api.Spec, &spec); err != nil {
		t.Fatal(err)
	}
	described := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			if !httpMethods[method] {
				continue // parameters, summary и прочие ключи уровня пути — не операции
			}
			key := method + " " + path
			described[key] = true
			if !routes[key] {
				t.Errorf("в openapi.yaml есть %s, а маршрута нет", key)
			}
		}
	}
	for key := range routes {
		if !described[key] {
			t.Errorf("маршрут %s не описан в api/openapi.yaml", key)
		}
	}
}
