//go:build integration

package tasks

import (
	"context"
	"errors"
	"example.com/hackathon/apps/tasks/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

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
	page, err := s.List(ctx, owner, 1, 1)
	if err != nil || len(page.Items) != 1 || page.Total != 2 || page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("Неверная пагинация: %+v %v", page, err)
	}
	empty, err := s.List(ctx, other, 20, 0)
	if err != nil || empty.Total != 0 || empty.Items == nil {
		t.Fatalf("Нарушена изоляция владельцев: %+v %v", empty, err)
	}
	if _, err = s.Update(ctx, other, first.ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Удалось изменить чужую запись: %v", err)
	}
	if err = s.Delete(ctx, other, first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Удалось удалить чужую запись: %v", err)
	}
	updated, err := s.Update(ctx, owner, first.ID, true)
	if err != nil || !updated.Completed {
		t.Fatalf("Не удалось изменить запись: %+v %v", updated, err)
	}
	if err = s.Delete(ctx, owner, first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Update(ctx, owner, first.ID, false); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Удалённая задача осталась доступна: %v", err)
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
