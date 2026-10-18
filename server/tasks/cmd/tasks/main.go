package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/hackathon/server/tasks/internal/broker"
	"example.com/hackathon/server/tasks/internal/config"
	"example.com/hackathon/server/tasks/internal/copilot"
	"example.com/hackathon/server/tasks/internal/database"
	"example.com/hackathon/server/tasks/internal/platform"
	"example.com/hackathon/server/tasks/internal/tasks"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrate()
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Сервис tasks остановлен", "error", err)
		os.Exit(1)
	}
}

// migrate применяет миграции только этого сервиса: tasks migrate [up|down|status].
func migrate() {
	command := "up"
	if len(os.Args) > 2 {
		command = os.Args[2]
	}
	if command != "up" && command != "down" && command != "status" {
		log.Fatal("Укажите команду up, down или status")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Укажите DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := database.Migrate(ctx, dsn, command); err != nil {
		log.Fatal(err)
	}
}

func run(logger *slog.Logger) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, err := platform.OpenDB(startup, c.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	check := func(ctx context.Context) error {
		if err := db.Ping(ctx); err != nil {
			return errors.New("Postgres недоступен")
		}
		return nil
	}
	if err := check(startup); err != nil {
		return err
	}
	r := platform.NewEngine(logger, 1<<20, check)
	api := r.Group("/api/v1", platform.TrustGateway(c.InternalToken))
	repo := tasks.NewRepository(db)
	service := tasks.New(repo)
	tasks.Register(api, service)
	tasks.RegisterAdmin(api, repo)
	var advisor copilot.Advisor
	if c.LLMServiceURL != "" {
		advisor = copilot.NewHTTPAdvisor(c.LLMServiceURL, c.InternalToken)
	}
	moderation := copilot.New(service, advisor, repo)
	copilot.Register(api, moderation)
	// Образец фонового задания: при нескольких репликах выполняется одной (advisory lock).
	go platform.Every(ctx, db, logger, "outbox-cleanup", time.Hour, repo.CleanupOutbox)
	// Брокер необязателен для CRUD: события лежат в outbox и уйдут, когда он станет доступен.
	brokerDone := make(chan struct{})
	if c.AMQPURL != "" {
		b := broker.New(c.AMQPURL, broker.Options{Queue: "tasks.moderation", Keys: []string{"task.created"}, Outbox: repo}, func(ctx context.Context, m broker.Message) error {
			return moderation.HandleEvent(ctx, m.Type, m.UserID, m.Data)
		}, logger)
		go func() { b.Run(ctx); close(brokerDone) }()
	} else {
		close(brokerDone)
	}
	server := &http.Server{Addr: c.Addr, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failed := make(chan error, 1)
	go func() {
		logger.Info("Сервис tasks принимает запросы", "addr", c.Addr)
		failed <- server.ListenAndServe()
	}()
	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	stop()
	<-brokerDone
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdown)
}
