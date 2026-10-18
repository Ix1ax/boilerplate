package tasks

import (
	"context"
	"errors"
	"example.com/hackathon/apps/tasks/internal/platform"
	"strings"
	"testing"
)

type fakeStore struct {
	inserted []Task
	owner    string
	from     string
}

func (f *fakeStore) List(context.Context, string, platform.ListQuery) (Page, error) {
	return Page{}, nil
}
func (f *fakeStore) Insert(_ context.Context, owner string, t Task) error {
	f.owner = owner
	f.inserted = append(f.inserted, t)
	return nil
}
func (f *fakeStore) Get(context.Context, string, string) (Task, error) { return Task{}, ErrNotFound }
func (f *fakeStore) UpdateStatus(_ context.Context, _, _, to, comment, _ string, check func(string) error) (Task, error) {
	if err := check(f.from); err != nil {
		return Task{}, err
	}
	return Task{Status: to}, nil
}
func (f *fakeStore) Delete(context.Context, string, string) error { return nil }
func (f *fakeStore) History(context.Context, string) ([]platform.Transition, error) {
	return nil, nil
}
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
	if err != nil || task.ID == "" || task.CreatedAt.IsZero() || task.Status != "draft" || store.owner != "owner" || task.Title != strings.Repeat("Я", 200) {
		t.Fatalf("Сохранена неверная задача: %+v %v", task, err)
	}
}

func TestUpdateChecksStatusWorkflow(t *testing.T) {
	service := New(&fakeStore{from: "draft"})
	if _, err := service.Update(context.Background(), "owner", "id", "done", ""); !errors.Is(err, platform.ErrTransition) {
		t.Fatalf("Недопустимый переход draft→done принят: %v", err)
	}
	service = New(&fakeStore{from: "todo"})
	if _, err := service.Update(context.Background(), "owner", "id", "archived", ""); !errors.Is(err, platform.ErrCommentRequired) {
		t.Fatalf("Архивация без комментария принята: %v", err)
	}
	task, err := service.Update(context.Background(), "owner", "id", "archived", "больше не нужна")
	if err != nil || task.Status != "archived" {
		t.Fatalf("Допустимый переход с комментарием не прошёл: %+v %v", task, err)
	}
}
