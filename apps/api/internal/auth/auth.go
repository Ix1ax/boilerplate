package auth

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"strings"
)

type Identity struct {
	Subject     string `json:"sub"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	RealmAccess struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
}

func Middleware(verifier Verifier, sessions ...*Sessions) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" && len(sessions) > 0 && sessions[0] != nil {
			token, err := sessions[0].AccessToken(c)
			if err != nil && !errors.Is(err, redis.Nil) {
				sessionError(c, 503, "session_unavailable", "Сессия временно недоступна. Повторите запрос")
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
		c.Set("identity", identity)
		c.Next()
	}
}
func Current(c *gin.Context) Identity { return c.MustGet("identity").(Identity) }
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
