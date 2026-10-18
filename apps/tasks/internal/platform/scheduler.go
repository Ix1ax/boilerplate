package platform

import (
	"context"
	"hash/fnv"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Every запускает job раз в interval, пока ctx не отменён. При нескольких репликах сервиса
// задание выполняет только одна: берётся advisory-блокировка Postgres по имени задания
// (отдельный Redis для этого не нужен). Ошибка задания пишется в лог и не останавливает цикл.
func Every(ctx context.Context, db *pgxpool.Pool, log *slog.Logger, name string, interval time.Duration, job func(context.Context) error) {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	key := int64(h.Sum64())
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		runLocked(ctx, db, log, name, key, interval, job)
	}
}

func runLocked(ctx context.Context, db *pgxpool.Pool, log *slog.Logger, name string, key int64, interval time.Duration, job func(context.Context) error) {
	conn, err := db.Acquire(ctx)
	if err != nil {
		log.Warn("Планировщик: нет соединения с БД", "job", name, "error", err)
		return
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", key).Scan(&locked); err != nil || !locked {
		return // задание уже выполняет другая реплика
	}
	defer conn.Exec(context.WithoutCancel(ctx), "SELECT pg_advisory_unlock($1)", key)
	jobCtx, cancel := context.WithTimeout(ctx, interval)
	defer cancel()
	start := time.Now()
	if err = job(jobCtx); err != nil {
		log.Warn("Фоновое задание завершилось с ошибкой", "job", name, "error", err)
		return
	}
	log.Info("Фоновое задание выполнено", "job", name, "duration_ms", time.Since(start).Milliseconds())
}
