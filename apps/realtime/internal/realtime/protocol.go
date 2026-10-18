// Package realtime — двусторонний канал браузер ↔ сервер поверх WebSocket.
//
// Протокол совпадает с клиентом apps/web/src/shared/ws: сообщения {type, payload}.
// Клиент → сервер: ping, subscribe{channel}, unsubscribe{channel}, publish{channel, data}.
// Сервер → клиент: pong, message{channel, from, data, at}, error{code, message}.
package realtime

import (
	"encoding/json"
	"strings"
	"time"
)

type inbound struct {
	Type    string `json:"type"`
	Payload struct {
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	} `json:"payload"`
}

// Message — сообщение канала; так же оно ходит между репликами через брокер.
type Message struct {
	Channel string          `json:"channel"`
	From    string          `json:"from"`
	Data    json.RawMessage `json:"data"`
	At      time.Time       `json:"at"`
}

func outbound(kind string, payload any) []byte {
	b, _ := json.Marshal(map[string]any{"type": kind, "payload": payload})
	return b
}

func errorFrame(code, message string) []byte {
	return outbound("error", map[string]string{"code": code, "message": message})
}

// Права на каналы — общая заготовка; доменные правила добавляются в Policy:
//
//	user:<id>  — личный канал: подписка только владельцу, писать может только сервер;
//	room:<имя> — комната: читать и писать могут все вошедшие (чат, «кто онлайн»);
//	broadcast  — объявления: читать все, писать только admin.
type Policy struct{}

func (Policy) CanSubscribe(user string, roles []string, channel string) bool {
	switch {
	case strings.HasPrefix(channel, "user:"):
		return channel == "user:"+user
	case strings.HasPrefix(channel, "room:"), channel == "broadcast":
		return validName(channel)
	}
	return false
}

func (Policy) CanPublish(user string, roles []string, channel string) bool {
	switch {
	case strings.HasPrefix(channel, "room:"):
		return validName(channel)
	case channel == "broadcast":
		for _, r := range roles {
			if r == "admin" {
				return true
			}
		}
	}
	return false
}

func validName(channel string) bool {
	if len(channel) > 100 {
		return false
	}
	for _, r := range channel {
		if !(r == ':' || r == '-' || r == '_' || r == '.' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}
