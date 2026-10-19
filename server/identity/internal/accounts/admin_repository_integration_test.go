//go:build integration

package accounts

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAdminRepositoryAgainstPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	repo := NewRepository(isolatedDB(t, ctx))
	s := NewAdmins(repo, NewHasher(4))
	if created, err := s.Bootstrap(ctx, "boss@example.ru", "ГлавныйПароль2026", "Главный Админ"); err != nil || !created {
		t.Fatalf("бутстрап: %v, %v", created, err)
	}
	// Второй суперадминистратор невозможен даже с другой почтой: частичный уникальный индекс.
	if created, err := s.Bootstrap(ctx, "other@example.ru", "ДругойПароль2027", "Другой"); err != nil || created {
		t.Fatalf("второй суперадминистратор: %v, %v", created, err)
	}
	boss, err := s.Login(ctx, "boss@example.ru", "ГлавныйПароль2026")
	if err != nil || boss.Role != RoleSuperadmin {
		t.Fatalf("вход: %+v, %v", boss, err)
	}
	mod, err := s.Create(ctx, AdminInput{Email: "Mod@Example.ru", Name: "Модератор", Password: "Модерация2026"}, boss.ID, "boss")
	if err != nil || mod.CreatedBy == nil || *mod.CreatedBy != boss.ID {
		t.Fatalf("создание: %+v, %v", mod, err)
	}
	if _, err = s.Create(ctx, AdminInput{Email: "mod@example.ru", Name: "Дубль", Password: "Модерация2026"}, boss.ID, "boss"); !errors.Is(err, ErrAdminEmailTaken) {
		t.Fatalf("дубль почты: %v", err)
	}
	if _, err = repo.SetAdminStatus(ctx, boss.ID, "blocked", "x", "boss"); !errors.Is(err, ErrSuperadminProtected) {
		t.Fatalf("блокировка суперадминистратора: %v", err)
	}
	before := mod.PasswordChangedAt
	if mod, err = s.ResetPassword(ctx, mod.ID, "СброшенныйПароль1", "boss"); err != nil || !mod.PasswordChangedAt.After(before) {
		t.Fatalf("сброс пароля должен сменить версию: %v", err)
	}
	if _, err = s.Delete(ctx, mod.ID, "boss"); err != nil {
		t.Fatal(err)
	}
	// После удаления почта свободна, а удалённый не входит и не виден в списке по умолчанию.
	if _, err = s.Create(ctx, AdminInput{Email: "mod@example.ru", Name: "Новый модератор", Password: "Модерация2027"}, boss.ID, "boss"); err != nil {
		t.Fatalf("почта удалённого не освободилась: %v", err)
	}
	page, err := s.List(ctx, "", "", 20, 0)
	if err != nil || page.Total != 2 || page.Items[0].Role != RoleSuperadmin {
		t.Fatalf("список: %+v, %v", page, err)
	}
	deleted, err := s.List(ctx, "mod", "deleted", 20, 0)
	if err != nil || deleted.Total != 1 {
		t.Fatalf("удалённые: %+v, %v", deleted, err)
	}
	d, err := s.Detail(ctx, mod.ID)
	if err != nil || len(d.History) != 3 || d.History[2].Action != "deleted" {
		t.Fatalf("история: %+v, %v", d.History, err)
	}
}
