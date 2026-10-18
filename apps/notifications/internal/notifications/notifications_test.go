package notifications

import (
	"context"
	"testing"
)

type fakeStore struct {
	inserted []Notification
	seen     map[string]bool
}

func (f *fakeStore) List(context.Context, string, int, int) (Page, error) { return Page{}, nil }
func (f *fakeStore) Insert(_ context.Context, _, eventID string, n Notification) (bool, error) {
	if f.seen == nil {
		f.seen = map[string]bool{}
	}
	if eventID != "" && f.seen[eventID] {
		return false, nil
	}
	f.seen[eventID] = true
	f.inserted = append(f.inserted, n)
	return true, nil
}
func (f *fakeStore) MarkRead(context.Context, string, string) error { return nil }
func (f *fakeStore) MarkAllRead(context.Context, string) error      { return nil }

func TestHandleEventValidates(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{}
	s := New(store, nil, nil, nil)
	ok := []byte(`{"title":" Готово ","body":"Текст","link":"/dashboard/tasks"}`)
	for _, c := range []struct {
		typ, user string
		data      []byte
	}{
		{"notification.requested", "u", ok},
		{"other", "u", ok},                                                               // чужой тип
		{"notification.requested", "", ok},                                               // без получателя
		{"notification.requested", "u", []byte(`{"title":""}`)},                          // пустой заголовок
		{"notification.requested", "u", []byte(`not json`)},                              // мусор
		{"notification.requested", "u", []byte(`{"title":"x","link":"//evil.example"}`)}, // внешняя ссылка
	} {
		if err := s.HandleEvent(ctx, "", c.typ, c.user, c.data); err != nil {
			t.Fatal(err)
		}
	}
	if len(store.inserted) != 2 || store.inserted[0].Title != "Готово" || store.inserted[1].Link != "" {
		t.Fatalf("%+v", store.inserted)
	}
}

type fakeMailer struct{ sent []string }

func (m *fakeMailer) Send(_ context.Context, to, subject, _ string) error {
	m.sent = append(m.sent, to+":"+subject)
	return nil
}

func TestDeliveryLiveAndEmailOncePerEvent(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()
	mail := &fakeMailer{}
	s := New(&fakeStore{}, hub, mail, nil)
	live, stop := hub.Subscribe("u")
	defer stop()
	data := []byte(`{"title":"Привет","email":"u@example.test"}`)
	const id = "7f3b8a52-1c1e-4c8e-9a43-3a1d2f0b9c01"
	for i := 0; i < 2; i++ { // повторная доставка того же события
		if err := s.HandleEvent(ctx, id, "notification.requested", "u", data); err != nil {
			t.Fatal(err)
		}
	}
	if len(mail.sent) != 1 || mail.sent[0] != "u@example.test:Привет" {
		t.Fatalf("письма: %v", mail.sent)
	}
	select {
	case n := <-live:
		if n.Title != "Привет" {
			t.Fatalf("%+v", n)
		}
	default:
		t.Fatal("live-событие не доставлено подписчику")
	}
	select {
	case <-live:
		t.Fatal("повторная доставка не должна дублировать live-событие")
	default:
	}
}
