// Package auth — вход и проверка пользователя: серверные сессии по cookie (токены Keycloak
// хранятся в Redis, браузеру отдаётся только случайный идентификатор), проверка JWT и роли.
package auth

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

// Identity — пользователь из проверенного токена. Subject (sub в Keycloak) — постоянный
// идентификатор: именно он уходит в сервисы как X-User-ID и служит владельцем записей.
type Identity struct {
	Subject     string `json:"sub"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	// Provider — способ входа: sso («Столото ID», Keycloak) или local (почта и пароль).
	// В токене Keycloak такого поля нет, его выставляет Middleware.
	Provider          string    `json:"provider,omitempty"`
	PasswordChangedAt time.Time `json:"password_changed_at,omitempty"`
}

const (
	ProviderSSO   = "sso"
	ProviderLocal = "local"
)

// Middleware пускает дальше только вошедшего пользователя. Токен берётся из заголовка
// Authorization (внешние клиенты, тесты) или из серверной сессии по cookie (браузер).
// Сессия обычной регистрации токена не содержит: в ней сам пользователь, проверенный identity.
// После проверки Identity лежит в контексте запроса — см. Current.
func Middleware(verifier Verifier, sessions ...*Sessions) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Явный Bearer принадлежит внешнему клиенту: не подменяем его браузерной сессией.
		header := c.GetHeader("Authorization")
		if header == "" && len(sessions) > 0 && sessions[0] != nil {
			local, token, err := sessions[0].Authenticate(c)
			// Недоступность Redis, Keycloak или identity не означает выход: клиент сможет повторить запрос.
			if err != nil && !errors.Is(err, redis.Nil) {
				sessionError(c, 503, "session_unavailable", "Сессия временно недоступна. Повторите запрос")
				return
			}
			if local != nil {
				identity := *local
				identity.Provider = ProviderLocal
				c.Set("identity", identity)
				c.Next()
				return
			}
			if token != "" {
				header = "Bearer " + token
			}
		}
		parts := strings.Fields(header)
		if verifier == nil || len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			unauthorized(c)
			return
		}
		token, err := verifier.Verify(c.Request.Context(), parts[1])
		if err != nil {
			unauthorized(c)
			return
		}
		var identity Identity
		if token.Claims(&identity) != nil || identity.Subject == "" {
			unauthorized(c)
			return
		}
		identity.Provider = ProviderSSO
		c.Set("identity", identity)
		c.Next()
	}
}

// Current — пользователь текущего запроса. Работает только на маршрутах за Middleware.
func Current(c *gin.Context) Identity { return c.MustGet("identity").(Identity) }

// RequireRole пропускает только пользователей с указанной ролью Keycloak (например admin), иначе 403.
func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, r := range Current(c).RealmAccess.Roles {
			if r == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "forbidden", "message": "Недостаточно прав для этого действия"}})
	}
}
func unauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(401, gin.H{"error": gin.H{"code": "unauthorized", "message": "Войдите, чтобы продолжить"}})
}
