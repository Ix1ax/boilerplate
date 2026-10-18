//go:build integration

package platform

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Пока задание выполняет одна «реплика», вторая с тем же именем его не запускает.
func TestSchedulerRunsOnlyOneReplica(t *testing.T) {
	db, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	var runs atomic.Int32
	release := make(chan struct{})
	job := func(context.Context) error { runs.Add(1); <-release; return nil }
	ctx := context.Background()
	done := make(chan struct{})
	go func() { runLocked(ctx, db, log, "test-job", 42, time.Minute, job); close(done) }()
	for runs.Load() == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	runLocked(ctx, db, log, "test-job", 42, time.Minute, func(context.Context) error { runs.Add(1); return nil })
	close(release)
	<-done
	if runs.Load() != 1 {
		t.Fatalf("задание выполнено %d раз при удержанной блокировке", runs.Load())
	}
	// После освобождения блокировки задание снова запускается.
	runLocked(ctx, db, log, "test-job", 42, time.Minute, func(context.Context) error { runs.Add(1); return nil })
	if runs.Load() != 2 {
		t.Fatal("блокировка не освобождена после выполнения")
	}
}
