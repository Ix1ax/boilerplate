// Сервис events — домен кейса: мероприятия, организации с допуском, записи, обсуждение.
// Запросы принимает только от gateway (общий INTERNAL_TOKEN).
// Запуск: `events` — HTTP-сервер; `events migrate [up|down|status]` — миграции его базы
// (в Docker это контейнер events-migrate, он отрабатывает до старта сервиса).
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

	"example.com/hackathon/server/events/api"
	"example.com/hackathon/server/events/internal/broker"
	"example.com/hackathon/server/events/internal/config"
	"example.com/hackathon/server/events/internal/database"
	"example.com/hackathon/server/events/internal/events"
	"example.com/hackathon/server/events/internal/platform"
	"example.com/hackathon/server/events/internal/weather"
	"github.com/gin-gonic/gin"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		migrate()
		return
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Сервис events остановлен", "error", err)
		os.Exit(1)
	}
}

// migrate применяет миграции только этого сервиса: events migrate [up|down|status].
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
	var stories *events.StoryS3
	if c.StoriesEndpoint != "" {
		stories = events.NewStoryS3(c.StoriesEndpoint, c.StoriesRegion, c.StoriesAccessKey, c.StoriesSecretKey, c.StoriesBucket)
		if err := stories.EnsureBucket(startup); err != nil {
			return err
		}
	}
	check := func(ctx context.Context) error {
		if err := db.Ping(ctx); err != nil {
			return errors.New("Postgres недоступен")
		}
		if stories != nil {
			return stories.Check(ctx)
		}
		return nil
	}
	if err := check(startup); err != nil {
		return err
	}
	r := platform.NewEngine(logger, 11<<20, check)
	// Своя спецификация: её читает Swagger в gateway. Маршрут без проверки пользователя —
	// сервис доступен только во внутренней сети.
	r.GET("/openapi.yaml", func(c *gin.Context) { c.Data(http.StatusOK, "application/yaml; charset=utf-8", api.Spec) })
	group := r.Group("/api/v1", platform.TrustGateway(c.InternalToken))
	repo := events.NewRepository(db)
	service := events.New(repo)
	if stories != nil {
		service.StoryMedia = stories
	}
	switch c.WeatherURL {
	case "off":
	case "":
		service.Weather = weather.New(weather.DefaultURL)
	default:
		service.Weather = weather.New(c.WeatherURL)
	}
	events.Register(group, service)
	events.RegisterAdmin(group, service)
	go platform.Every(ctx, db, logger, "outbox-cleanup", time.Hour, repo.CleanupOutbox)
	go platform.Every(ctx, db, logger, "story-media-cleanup", 10*time.Minute, func(ctx context.Context) error { return repo.CleanupStoryMedia(ctx, service.StoryMedia) })
	// Срок сбора состава проверяем раз в минуту: отмена должна прийти вовремя, а не через час.
	go platform.Every(ctx, db, logger, "quorum-deadlines", time.Minute, repo.FailQuorums)
	go platform.Every(ctx, db, logger, "demand-proposals", time.Minute, service.RefreshDemandProposals)
	go platform.Every(ctx, db, logger, "event-reminders", 5*time.Minute, service.SendReminders)
	// Брокер только публикует уведомления и realtime-сигналы из outbox; без него данные
	// сохраняются, а события уйдут, когда он станет доступен.
	brokerDone := make(chan struct{})
	if c.AMQPURL != "" {
		b := broker.New(c.AMQPURL, broker.Options{Queue: "events.service", Outbox: repo}, func(context.Context, broker.Message) error { return nil }, logger)
		go func() { b.Run(ctx); close(brokerDone) }()
	} else {
		close(brokerDone)
	}
	server := &http.Server{Addr: c.Addr, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failed := make(chan error, 1)
	go func() {
		logger.Info("Сервис events принимает запросы", "addr", c.Addr)
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
