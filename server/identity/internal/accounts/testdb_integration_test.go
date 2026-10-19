//go:build integration

package accounts

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"example.com/hackathon/server/identity/internal/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// isolatedDB создаёт для теста отдельную базу на том же сервере, что DATABASE_URL, и удаляет её после.
// Тесты создают учётные записи: рабочая база остаётся нетронутой.
func isolatedDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("Укажите DATABASE_URL для интеграционного теста")
	}
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := "identity_it_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close()
		t.Fatalf("не удалось создать тестовую базу (нужно право CREATEDB): %v", err)
	}
	t.Cleanup(func() {
		// Контекст теста к этому моменту может быть отменён, поэтому удаляем с фоновым.
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		admin.Close()
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	if err = database.Migrate(ctx, u.String(), "up"); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}
