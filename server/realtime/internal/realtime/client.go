package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Ограничения соединения: защищают сервис от слишком частых или крупных сообщений.
const (
	maxFrame    = 16 << 10         // байт на входящее сообщение
	maxChannels = 50               // подписок на соединение
	idleTimeout = 70 * time.Second // клиент шлёт ping каждые 25 с
	writeWait   = 10 * time.Second
	sendBuffer  = 64 // исходящих кадров в очереди; медленный клиент отключается
	ratePerSec  = 20 // входящих сообщений в секунду (с запасом ×2 на всплеск)
)

// Publisher — куда отправлять сообщения клиентов: брокер (все реплики) или сразу в Hub.
type Publisher interface {
	Publish(ctx context.Context, m Message) error
}

type client struct {
	conn     *websocket.Conn
	user     string
	roles    []string
	hub      *Hub
	pub      Publisher
	policy   Policy
	log      *slog.Logger
	send     chan []byte
	channels map[string]bool
	cancel   context.CancelFunc
}

func (c *client) enqueue(frame []byte) {
	select {
	case c.send <- frame:
	default:
		// Клиент не успевает читать: закрываем, чтобы не копить память. Браузер переподключится.
		c.cancel()
	}
}

// serve обслуживает соединение до разрыва: читает команды и пишет кадры из очереди.
func (c *client) serve(ctx context.Context) {
	ctx, c.cancel = context.WithCancel(ctx)
	defer c.cancel()
	defer c.hub.remove(c)
	c.conn.SetReadLimit(maxFrame)
	go c.writeLoop(ctx)
	tokens, last := float64(ratePerSec*2), time.Now()
	for {
		readCtx, cancel := context.WithTimeout(ctx, idleTimeout)
		_, data, err := c.conn.Read(readCtx)
		cancel()
		if err != nil {
			code := websocket.CloseStatus(err)
			if code == -1 && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				c.log.Debug("WebSocket: чтение прервано", "error", err)
			}
			if errors.Is(err, context.DeadlineExceeded) {
				_ = c.conn.Close(websocket.StatusGoingAway, "нет активности")
			}
			return
		}
		now := time.Now()
		tokens = min(float64(ratePerSec*2), tokens+now.Sub(last).Seconds()*ratePerSec)
		last = now
		if tokens < 1 {
			c.enqueue(errorFrame("rate_limit", "Слишком много сообщений"))
			continue
		}
		tokens--
		c.handle(ctx, data)
	}
}

func (c *client) handle(ctx context.Context, data []byte) {
	var in inbound
	if json.Unmarshal(data, &in) != nil {
		c.enqueue(errorFrame("invalid_message", "Ожидается JSON {type, payload}"))
		return
	}
	channel := strings.TrimSpace(in.Payload.Channel)
	switch in.Type {
	case "ping":
		c.enqueue(outbound("pong", nil))
	case "subscribe":
		if !c.policy.CanSubscribe(c.user, c.roles, channel) {
			c.enqueue(errorFrame("forbidden", "Нет доступа к каналу "+channel))
			return
		}
		if !c.channels[channel] && len(c.channels) >= maxChannels {
			c.enqueue(errorFrame("too_many_channels", "Слишком много подписок"))
			return
		}
		c.channels[channel] = true
		c.hub.subscribe(c, channel)
	case "unsubscribe":
		delete(c.channels, channel)
		c.hub.unsubscribe(c, channel)
	case "publish":
		if !c.policy.CanPublish(c.user, c.roles, channel) {
			c.enqueue(errorFrame("forbidden", "Нельзя писать в канал "+channel))
			return
		}
		if len(in.Payload.Data) == 0 || !json.Valid(in.Payload.Data) {
			c.enqueue(errorFrame("invalid_message", "payload.data должен быть JSON"))
			return
		}
		m := Message{Channel: channel, From: c.user, Data: in.Payload.Data, At: time.Now().UTC()}
		if err := c.pub.Publish(ctx, m); err != nil {
			c.log.Warn("WebSocket: публикация не удалась", "error", err)
			c.enqueue(errorFrame("unavailable", "Сообщение не отправлено, повторите"))
		}
	default:
		c.enqueue(errorFrame("unknown_type", "Неизвестный тип сообщения "+in.Type))
	}
}

func (c *client) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			_ = c.conn.Close(websocket.StatusNormalClosure, "")
			return
		case frame := <-c.send:
			wctx, cancel := context.WithTimeout(ctx, writeWait)
			err := c.conn.Write(wctx, websocket.MessageText, frame)
			cancel()
			if err != nil {
				c.cancel()
				return
			}
		}
	}
}
