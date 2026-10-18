//go:build integration

package platform

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestHistoryInTransaction(t *testing.T) {
	ctx := context.Background()
	db, err := pgxpool.New(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := uuid.NewString()
	defer db.Exec(ctx, "DELETE FROM status_history WHERE entity_id=$1", id)
	tx, err := db.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, tr := range []Transition{{From: "draft", To: "review", Actor: "u"}, {From: "review", To: "draft", Actor: "admin", Comment: "Уточните сроки"}} {
		if err = RecordTransition(ctx, tx, "test", id, tr); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	h, err := History(ctx, db, "test", id)
	if err != nil || len(h) != 2 || h[1].Comment != "Уточните сроки" || h[0].To != "review" {
		t.Fatalf("%+v %v", h, err)
	}
}
