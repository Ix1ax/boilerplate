package tasks

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeStore struct {
	inserted []Task
	owner    string
}

func (f *fakeStore) List(context.Context, string, int, int) (Page, error) { return Page{}, nil }
func (f *fakeStore) Insert(_ context.Context, owner string, t Task) error {
	f.owner = owner
	f.inserted = append(f.inserted, t)
	return nil
}
func (f *fakeStore) Update(context.Context, string, string, bool) (Task, error) { return Task{}, nil }
func (f *fakeStore) Delete(context.Context, string, string) error               { return nil }
func TestCreateValidatesUnicodeTitleBeforePersistence(t *testing.T) {
	store := &fakeStore{}
	service := New(store)
	for _, title := range []string{"", "  ", strings.Repeat("Я", 201), string([]byte{0xff})} {
		if _, err := service.Create(context.Background(), "owner", title); !errors.Is(err, ErrTitle) {
			t.Fatalf("Название %q: %v", title, err)
		}
	}
	if len(store.inserted) != 0 {
		t.Fatal("Задача с недопустимым названием попала в хранилище")
	}
	task, err := service.Create(context.Background(), "owner", "  "+strings.Repeat("Я", 200)+"  ")
	if err != nil || task.ID == "" || task.CreatedAt.IsZero() || task.Completed || store.owner != "owner" || task.Title != strings.Repeat("Я", 200) {
		t.Fatalf("Сохранена неверная задача: %+v %v", task, err)
	}
}
