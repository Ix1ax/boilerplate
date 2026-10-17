//go:build integration

package files

import (
	"context"
	"example.com/hackathon/apps/api/internal/database"
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
	page, err := List(ctx, db, owner, 1, 1)
	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Limit != 1 || page.Offset != 1 {
		t.Fatalf("пагинация: %+v %v", page, err)
	}
	otherPage, err := List(ctx, db, other, 20, 0)
	if err != nil || otherPage.Total != 1 || otherPage.Items[0].ID == page.Items[0].ID {
		t.Fatalf("изоляция владельцев: %+v %v", otherPage, err)
	}
	empty, err := List(ctx, db, uuid.NewString(), 20, 0)
	if err != nil || empty.Total != 0 || empty.Items == nil || len(empty.Items) != 0 {
		t.Fatalf("пустой список: %+v %v", empty, err)
	}
}
