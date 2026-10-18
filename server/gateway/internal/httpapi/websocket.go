package httpapi

import "github.com/gin-gonic/gin"

// websocketGuard защищает от перехвата сокета с чужого сайта (CSWSH): браузер шлёт cookie сессии
// и на кросс-доменный WebSocket, а CSRF-проверка gateway касается только изменяющих методов.
// Поэтому для cookie-подключений требуем точный Origin приложения; Bearer-клиенты не затронуты.
func websocketGuard(appOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Authorization") == "" && c.GetHeader("Origin") != appOrigin {
			c.AbortWithStatusJSON(403, gin.H{"error": gin.H{"code": "invalid_origin", "message": "Подключение с этого адреса запрещено"}})
			return
		}
		c.Next()
	}
}
