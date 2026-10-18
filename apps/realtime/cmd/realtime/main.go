package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/hackathon/apps/realtime/internal/config"
	"example.com/hackathon/apps/realtime/internal/platform"
	"example.com/hackathon/apps/realtime/internal/realtime"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Сервис realtime остановлен", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	hub := realtime.NewHub()
	var pub realtime.Publisher = realtime.LocalPublisher(hub)
	if c.AMQPURL != "" {
		bus := realtime.NewBus(c.AMQPURL, hub, logger)
		go bus.Run(ctx)
		pub = bus
	}
	// Готовность не зависит от брокера: без него сообщения доставляются в пределах реплики.
	r := platform.NewEngine(logger, 1<<20, func(context.Context) error { return nil })
	realtime.Register(r.Group("/api/v1", platform.TrustGateway(c.InternalToken)), hub, pub, logger)
	// Таймауты чтения/записи не задаём: соединения WebSocket долгие, простой обрывает сам клиент-хендлер.
	server := &http.Server{Addr: c.Addr, Handler: r, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failed := make(chan error, 1)
	go func() {
		logger.Info("Сервис realtime принимает запросы", "addr", c.Addr)
		failed <- server.ListenAndServe()
	}()
	select {
	case err := <-failed:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return server.Shutdown(shutdown)
}
