// Сервис audit — журнал действий: сохраняет записи, которые публикует gateway.
// Запросы принимает только от gateway (общий INTERNAL_TOKEN).
// Запуск: `audit` — HTTP-сервер; `audit migrate [up|down|status]` — миграции его базы
// (в Docker это контейнер audit-migrate, он отрабатывает до старта сервиса).
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

	"example.com/hackathon/server/audit/api"
	"example.com/hackathon/server/audit/internal/audit"
	"example.com/hackathon/server/audit/internal/broker"
	"example.com/hackathon/server/audit/internal/config"
	"example.com/hackathon/server/audit/internal/database"
	"example.com/hackathon/server/audit/internal/platform"
	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrate()
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Сервис audit остановлен", "error", err)
		os.Exit(1)
	}
}

// migrate применяет миграции только этого сервиса: audit migrate [up|down|status].
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
	// Своя спецификация: её читает Swagger в gateway (список «сервисы»). Маршрут без проверки
	// пользователя — сервис доступен только во внутренней сети.
	r.GET("/openapi.yaml", func(c *gin.Context) { c.Data(http.StatusOK, "application/yaml; charset=utf-8", api.Spec) })
	api := r.Group("/api/v1", platform.TrustGateway(c.InternalToken))
	repo := audit.NewRepository(db)
	audit.Register(api, repo)
	service := audit.New(repo)
	// Подписка на audit.recorded: записи публикует gateway о каждом изменяющем запросе.
	brokerDone := make(chan struct{})
	if c.AMQPURL != "" {
		b := broker.New(c.AMQPURL, broker.Options{Queue: "audit.records", Keys: []string{"audit.recorded"}}, func(ctx context.Context, m broker.Message) error {
			return service.HandleEvent(ctx, m.ID, m.Type, m.Data)
		}, logger)
		go func() { b.Run(ctx); close(brokerDone) }()
	} else {
		close(brokerDone)
	}
	server := &http.Server{Addr: c.Addr, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failed := make(chan error, 1)
	go func() {
		logger.Info("Сервис audit принимает запросы", "addr", c.Addr)
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
