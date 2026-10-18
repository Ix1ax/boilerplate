package notifications

import (
	"context"
	"testing"
)

type fakeStore struct{ inserted []Notification }

func (f *fakeStore) List(context.Context, string, int, int) (Page, error) { return Page{}, nil }
func (f *fakeStore) Insert(_ context.Context, _, _ string, n Notification) error {
	f.inserted = append(f.inserted, n)
	return nil
}
func (f *fakeStore) MarkRead(context.Context, string, string) error { return nil }
func (f *fakeStore) MarkAllRead(context.Context, string) error      { return nil }

func TestHandleEventValidates(t *testing.T) {
	ctx := context.Background()
	store := &fakeStore{}
	s := New(store)
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
