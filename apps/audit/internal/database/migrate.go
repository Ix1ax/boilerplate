package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// EnsureDatabase создаёт базу сервиса, если её нет: у каждого сервиса своя база,
// и заранее (в init-скриптах Postgres) она может не существовать, например на старом томе.
func EnsureDatabase(ctx context.Context, dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return err
	}
	name := strings.TrimPrefix(u.Path, "/")
	admin := *u
	admin.Path = "/postgres"
	conn, err := pgx.Connect(ctx, admin.String())
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	var exists bool
	if err = conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname=$1)", name).Scan(&exists); err != nil || exists {
		return err
	}
	_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	return err
}

func Migrate(ctx context.Context, dsn, command string) error {
	if command == "up" {
		if err := EnsureDatabase(ctx, dsn); err != nil {
			return fmt.Errorf("создание базы: %w", err)
		}
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	goose.SetBaseFS(migrations)
	if err = goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.RunContext(ctx, command, db, "migrations")
}
