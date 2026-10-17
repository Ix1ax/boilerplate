package httpapi

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"example.com/hackathon/apps/gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// Заголовки внутреннего контракта gateway → сервисы. Сервисы объявляют те же имена у себя:
// общего пакета нет намеренно, чтобы модули собирались независимо.
const (
	HeaderToken  = "X-Internal-Token"
	HeaderUserID = "X-User-ID"
)

// Upstreams — адреса внутренних сервисов, за которыми стоит gateway.
type Upstreams struct{ Tasks, Files *url.URL }

// proxy пересылает запрос во внутренний сервис. Пользовательские заголовки аутентификации
// удаляются: сервис видит только личность, которую проверил gateway.
func proxy(target *url.URL, token string) gin.HandlerFunc {
	rp := &httputil.ReverseProxy{
		Rewrite:       func(r *httputil.ProxyRequest) { r.SetURL(target) },
		FlushInterval: -1,
		Transport:     &http.Transport{MaxIdleConnsPerHost: 16, IdleConnTimeout: 60 * time.Second, ResponseHeaderTimeout: 25 * time.Second},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte(`{"error":{"code":"upstream_unavailable","message":"Сервис временно недоступен"}}`))
		},
	}
	return func(c *gin.Context) {
		// Сначала удаляем присланное клиентом, затем ставим значения gateway.
		for _, h := range []string{"Authorization", "Cookie"} {
			c.Request.Header.Del(h)
		}
		c.Request.Header.Set(HeaderToken, token)
		c.Request.Header.Set(HeaderUserID, auth.Current(c).Subject)
		c.Request.Header.Set("X-Request-ID", c.GetString("request_id"))
		rp.ServeHTTP(c.Writer, c.Request)
	}
}
