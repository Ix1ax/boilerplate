package realtime

import "sync"

// Hub — подписки клиентов этой реплики на каналы. Межрепликовую доставку делает Bus.
type Hub struct {
	mu   sync.RWMutex
	subs map[string]map[*client]struct{}
}

func NewHub() *Hub { return &Hub{subs: map[string]map[*client]struct{}{}} }

func (h *Hub) subscribe(c *client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[channel] == nil {
		h.subs[channel] = map[*client]struct{}{}
	}
	h.subs[channel][c] = struct{}{}
}

func (h *Hub) unsubscribe(c *client, channel string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.subs[channel], c)
	if len(h.subs[channel]) == 0 {
		delete(h.subs, channel)
	}
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for channel, set := range h.subs {
		delete(set, c)
		if len(set) == 0 {
			delete(h.subs, channel)
		}
	}
}

// Deliver отправляет сообщение подписчикам канала на этой реплике (без блокировки отправителя).
func (h *Hub) Deliver(m Message) {
	frame := outbound("message", m)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.subs[m.Channel] {
		c.enqueue(frame)
	}
}

// Subscribers — число подписчиков канала на этой реплике (для тестов и метрик).
func (h *Hub) Subscribers(channel string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs[channel])
}
