//go:build integration

package accounts

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRepositoryAgainstPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo := NewRepository(isolatedDB(t, ctx))
	s := New(repo, NewHasher(4))
	a, err := s.Register(ctx, RegisterInput{Email: "Olga@Example.ru", Password: "ЙогаНаРассвете9", Name: "Ольга"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Register(ctx, RegisterInput{Email: "olga@example.ru ", Password: "ЙогаНаРассвете9", Name: "Ольга"}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("уникальность почты без учёта регистра: %v", err)
	}
	if _, err = s.Login(ctx, "olga@example.ru", "ЙогаНаРассвете9"); err != nil {
		t.Fatal(err)
	}
	// Параллельные неверные попытки считаются атомарно: порог не обходится гонкой.
	var wg sync.WaitGroup
	for range maxFailures {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = s.Login(ctx, "olga@example.ru", "неверный123") }()
	}
	wg.Wait()
	var locked LockedError
	if _, err = s.Login(ctx, "olga@example.ru", "ЙогаНаРассвете9"); !errors.As(err, &locked) {
		t.Fatalf("после %d ошибок вход должен быть закрыт: %v", maxFailures, err)
	}
	if _, err = repo.db.Exec(ctx, "UPDATE accounts SET locked_until=now()-interval '1 second' WHERE id=$1", a.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.Login(ctx, "olga@example.ru", "ЙогаНаРассвете9")
	if err != nil || got.LastLoginAt == nil {
		t.Fatalf("вход после истечения блокировки: %+v, %v", got, err)
	}
	if _, err = s.Block(ctx, a.ID, "Спам в обсуждениях", "Модератор (admin-id)"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Block(ctx, a.ID, "Ещё раз", "Модератор"); !errors.Is(err, ErrTransition) {
		t.Fatalf("повторная блокировка: %v", err)
	}
	page, err := s.List(ctx, "оль", "blocked", 20, 0)
	if err != nil || page.Total != 1 || page.Items[0].BlockReason != "Спам в обсуждениях" {
		t.Fatalf("поиск заблокированных: %+v, %v", page, err)
	}
	if page, _ = s.List(ctx, "%", "", 20, 0); page.Total != 0 {
		t.Fatalf("символ LIKE не экранирован: %+v", page)
	}
	if _, err = s.Unblock(ctx, a.ID, "Модератор"); err != nil {
		t.Fatal(err)
	}
	if err = s.ChangePassword(ctx, a.ID, "ЙогаНаРассвете9", "ЙогаНаЗакате10"); err != nil {
		t.Fatal(err)
	}
	d, err := s.Detail(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	var actions []string
	for _, h := range d.History {
		actions = append(actions, h.Action)
	}
	if want := "registered blocked unblocked password_changed"; strings.Join(actions, " ") != want {
		t.Fatalf("история: %q, ожидалось %q", strings.Join(actions, " "), want)
	}
}
