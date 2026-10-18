package realtime

import (
	"context"
	"log/slog"
	"strings"

	"example.com/hackathon/apps/realtime/internal/platform"
	"github.com/coder/websocket"
	"github.com/gin-gonic/gin"
)

// localPublisher — без брокера: сообщение доставляется подписчикам этой реплики.
type localPublisher struct{ hub *Hub }

func (p localPublisher) Publish(_ context.Context, m Message) error {
	p.hub.Deliver(m)
	return nil
}

// LocalPublisher нужен, когда брокер не настроен (одна реплика, тесты).
func LocalPublisher(h *Hub) Publisher { return localPublisher{hub: h} }

// Register добавляет GET /ws. Пользователь и роли приходят от gateway (TrustGateway);
// Origin проверяет gateway, поэтому здесь проверка хоста отключена.
func Register(group *gin.RouterGroup, hub *Hub, pub Publisher, log *slog.Logger) {
	group.GET("/ws", func(c *gin.Context) {
		conn, err := websocket.Accept(c.Writer, c.Request, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return // Accept уже ответил клиенту
		}
		roles := []string{}
		if raw := c.GetHeader(platform.HeaderRoles); raw != "" {
			roles = strings.Split(raw, ",")
		}
		cl := &client{
			conn: conn, user: platform.Owner(c), roles: roles, hub: hub, pub: pub, log: log,
			send: make(chan []byte, sendBuffer), channels: map[string]bool{},
		}
		// Контекст запроса закрывается после Hijack, поэтому соединение живёт на своём.
		cl.serve(context.WithoutCancel(c.Request.Context()))
	})
}
