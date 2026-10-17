//go:build integration

package files

import (
	"context"
	"errors"
	"example.com/hackathon/apps/files/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"time"
)

func TestFileListOwnershipAndPagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dsn := os.Getenv("DATABASE_URL")
	if err := database.Migrate(ctx, dsn, "up"); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner, other := uuid.NewString(), uuid.NewString()
	defer db.Exec(context.Background(), "DELETE FROM files WHERE owner_id=$1 OR owner_id=$2", owner, other)
	for _, subject := range []string{owner, owner, other} {
		id := uuid.NewString()
		_, err = db.Exec(ctx, "INSERT INTO files(id,owner_id,object_key,name,size,content_type) VALUES($1,$2,$3,'test.txt',4,'text/plain')", id, subject, subject+"/"+id)
		if err != nil {
			t.Fatal(err)
		}
	}
	repo := NewRepository(db)
	page, err := repo.List(ctx, owner, 1, 1)
	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("пагинация: %+v %v", page, err)
	}
	otherPage, err := repo.List(ctx, other, 20, 0)
	if err != nil || otherPage.Total != 1 || otherPage.Items[0].ID == page.Items[0].ID {
		t.Fatalf("изоляция владельцев: %+v %v", otherPage, err)
	}
	empty, err := repo.List(ctx, uuid.NewString(), 20, 0)
	if err != nil || empty.Total != 0 || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("пустой список: %+v %v", empty, err)
	}
}

func TestFileGetAndDeleteOwnership(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	owner, other, id := uuid.NewString(), uuid.NewString(), uuid.NewString()
	defer db.Exec(context.Background(), "DELETE FROM files WHERE id=$1", id)
	repo := NewRepository(db)
	if err := repo.Insert(ctx, owner, "private-key", File{ID: id, Name: "test.txt", Size: 4, ContentType: "text/plain", CreatedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := repo.Get(ctx, other, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("чужой файл доступен: %v", err)
	}
	if err := repo.Delete(ctx, other, id); err != nil {
		t.Fatal(err)
	}
	if key, _, _, err := repo.Get(ctx, owner, id); err != nil || key != "private-key" {
		t.Fatalf("чужой запрос удалил файл: %v", err)
	}
	for range 2 {
		if err := repo.Delete(ctx, owner, id); err != nil {
			t.Fatalf("повтор удаления небезопасен: %v", err)
		}
	}
	if _, _, _, err := repo.Get(ctx, owner, id); !errors.Is(err, ErrNotFound) {
		t.Fatalf("файл остался после удаления: %v", err)
	}
}
