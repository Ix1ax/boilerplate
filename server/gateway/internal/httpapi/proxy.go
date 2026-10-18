package httpapi

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"example.com/hackathon/server/gateway/internal/auth"
	"github.com/gin-gonic/gin"
)

// Заголовки внутреннего контракта gateway → сервисы. Сервисы объявляют те же имена у себя:
// общего пакета нет намеренно, чтобы модули собирались независимо.
const (
	HeaderToken  = "X-Internal-Token"
	HeaderUserID = "X-User-ID"
	// HeaderRoles — роли пользователя из проверенного JWT через запятую: сервисы сами решают,
	// какие действия доступны (например, админская выгрузка), не разбирая токен.
	HeaderRoles = "X-User-Roles"
	// HeaderEmail — адрес из JWT: нужен сервисам, чтобы отправить письмо (уведомления).
	HeaderEmail = "X-User-Email"
)

// Upstreams — адреса внутренних сервисов, за которыми стоит gateway.
type Upstreams struct{ Tasks, Files, LLM, Notifications, Audit, Realtime *url.URL }

// slowUpstream — для долгих ответов (LLM, SSE): иначе gateway оборвёт их по таймаутам.
const slowUpstream = 3 * time.Minute

// proxy пересылает запрос во внутренний сервис. Пользовательские заголовки аутентификации
// удаляются: сервис видит только личность, которую проверил gateway.
func proxy(target *url.URL, token string, slow ...time.Duration) gin.HandlerFunc {
	headerTimeout := 25 * time.Second
	if len(slow) > 0 {
		headerTimeout = slow[0]
	}
	rp := &httputil.ReverseProxy{
		Rewrite:       func(r *httputil.ProxyRequest) { r.SetURL(target) },
		FlushInterval: -1,
		Transport:     &http.Transport{MaxIdleConnsPerHost: 16, IdleConnTimeout: 60 * time.Second, ResponseHeaderTimeout: headerTimeout},
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
		identity := auth.Current(c)
		c.Request.Header.Set(HeaderUserID, identity.Subject)
		c.Request.Header.Set(HeaderRoles, strings.Join(identity.RealmAccess.Roles, ","))
		c.Request.Header.Set(HeaderEmail, identity.Email)
		c.Request.Header.Set("X-Request-ID", c.GetString("request_id"))
		if len(slow) > 0 {
			_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(slow[0] + 10*time.Second))
		}
		rp.ServeHTTP(c.Writer, c.Request)
	}
}
