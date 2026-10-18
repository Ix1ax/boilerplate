//go:build integration

package notifications

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"example.com/hackathon/server/notifications/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMaxLinks(t *testing.T) {
	ctx := context.Background()
	dsn := os.Getenv("DATABASE_URL")
	if err := database.Migrate(ctx, dsn, "up"); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	r := NewRepository(db)
	u1, u2, maxID := uuid.NewString(), uuid.NewString(), time.Now().UnixNano()
	defer db.Exec(context.Background(), "DELETE FROM max_links WHERE user_id IN ($1,$2)", u1, u2)
	if err = r.CreateLinkCode(ctx, u1, "code-"+u1, time.Minute); err != nil {
		t.Fatal(err)
	}
	if user, err := r.ConsumeLinkCode(ctx, "code-"+u1); err != nil || user != u1 {
		t.Fatalf("%q %v", user, err)
	}
	if _, err = r.ConsumeLinkCode(ctx, "code-"+u1); !errors.Is(err, ErrLinkCode) {
		t.Fatalf("код погашен, но принят повторно: %v", err)
	}
	_ = r.CreateLinkCode(ctx, u2, "old-"+u2, -time.Minute)
	if _, err = r.ConsumeLinkCode(ctx, "old-"+u2); !errors.Is(err, ErrLinkCode) {
		t.Fatalf("просроченный код принят: %v", err)
	}
	// Один MAX-аккаунт привязан только к одному пользователю: повторная привязка переносит его.
	if err = r.Link(ctx, u1, maxID); err != nil {
		t.Fatal(err)
	}
	if err = r.Link(ctx, u2, maxID); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := r.MaxUserID(ctx, u1); ok {
		t.Fatal("старая привязка не снята")
	}
	if id, ok, _ := r.MaxUserID(ctx, u2); !ok || id != maxID {
		t.Fatal("новая привязка не сохранена")
	}
	if ok, _ := r.UnlinkMax(ctx, maxID); !ok {
		t.Fatal("отвязка по /stop не сработала")
	}
}
