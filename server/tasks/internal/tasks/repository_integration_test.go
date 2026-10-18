//go:build integration

package tasks

import (
	"context"
	"errors"
	"example.com/hackathon/server/tasks/internal/database"
	"example.com/hackathon/server/tasks/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

var defaultOrder = platform.ListQuery{OrderBy: " ORDER BY t.created_at DESC"}

func TestRepositoryOwnershipAndPagination(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("Укажите DATABASE_URL для интеграционного теста")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.Migrate(ctx, dsn, "up"); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner := uuid.NewString()
	other := uuid.NewString()
	defer db.Exec(context.Background(), "DELETE FROM tasks WHERE owner_id=$1", owner)
	s := New(NewRepository(db))
	first, err := s.Create(ctx, owner, "first")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Create(ctx, owner, "second"); err != nil {
		t.Fatal(err)
	}
	page, err := s.List(ctx, owner, platform.ListQuery{Limit: 1, Offset: 1, OrderBy: defaultOrder.OrderBy})
	if err != nil || len(page.Items) != 1 || page.Total != 2 || page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("Неверная пагинация: %+v %v", page, err)
	}
	empty, err := s.List(ctx, other, platform.ListQuery{Limit: 20, OrderBy: defaultOrder.OrderBy})
	if err != nil || empty.Total != 0 || empty.Items == nil {
		t.Fatalf("Нарушена изоляция владельцев: %+v %v", empty, err)
	}
	if _, err = s.Update(ctx, other, first.ID, "todo", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Удалось изменить чужую запись: %v", err)
	}
	if err = s.Delete(ctx, other, first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Удалось удалить чужую запись: %v", err)
	}
	updated, err := s.Update(ctx, owner, first.ID, "todo", "")
	if err != nil || updated.Status != "todo" {
		t.Fatalf("Не удалось изменить запись: %+v %v", updated, err)
	}
	history, err := s.History(ctx, owner, first.ID)
	if err != nil || len(history) != 1 || history[0].From != "draft" || history[0].To != "todo" {
		t.Fatalf("Неверная история переходов: %+v %v", history, err)
	}
	if err = s.Delete(ctx, owner, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Update(ctx, owner, first.ID, "todo", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Удалённая задача осталась доступна: %v", err)
	}
}

func TestRepositoryListFiltersAndSearches(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("Укажите DATABASE_URL для интеграционного теста")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner := uuid.NewString()
	defer db.Exec(context.Background(), "DELETE FROM tasks WHERE owner_id=$1", owner)
	s := New(NewRepository(db))
	task, err := s.Create(ctx, owner, "уникальный заголовок")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Create(ctx, owner, "другая задача"); err != nil {
		t.Fatal(err)
	}
	found, err := s.List(ctx, owner, platform.ListQuery{Limit: 20, Where: " AND t.status = $2", Args: []any{"draft"}, OrderBy: defaultOrder.OrderBy})
	if err != nil || found.Total != 2 {
		t.Fatalf("Фильтр по статусу: %+v %v", found, err)
	}
	searched, err := s.List(ctx, owner, platform.ListQuery{Limit: 20, Where: " AND t.title ILIKE $2", Args: []any{"%уникальный%"}, OrderBy: defaultOrder.OrderBy})
	if err != nil || searched.Total != 1 || searched.Items[0].ID != task.ID {
		t.Fatalf("Поиск по названию: %+v %v", searched, err)
	}
}

func TestStatsSnapshot(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := NewRepository(db)
	before, err := repo.Stats(ctx)
	if err != nil || len(before.ByDay) != 14 {
		t.Fatalf("%+v %v", before, err)
	}
	owner := uuid.NewString()
	defer db.Exec(context.Background(), "DELETE FROM tasks WHERE owner_id=$1", owner)
	if _, err = New(repo).Create(ctx, owner, "для статистики"); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.Stats(ctx)
	if after.Total != before.Total+1 || after.ByDay[13].Count != before.ByDay[13].Count+1 || after.Moderation.Unchecked < 1 {
		t.Fatalf("до %+v после %+v", before, after)
	}
}
