package main

import (
	"context"
	"errors"
	"example.com/hackathon/server/gateway/internal/audit"
	"example.com/hackathon/server/gateway/internal/auth"
	"example.com/hackathon/server/gateway/internal/config"
	"example.com/hackathon/server/gateway/internal/httpapi"
	"example.com/hackathon/server/gateway/internal/platform"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("API остановлен", "error", err)
		os.Exit(1)
	}
}
func run(log *slog.Logger) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	deps, err := platform.Open(startup, c)
	if err != nil {
		return err
	}
	defer deps.Close()
	tasksURL, _ := url.Parse(c.TasksURL)
	filesURL, _ := url.Parse(c.FilesURL)
	// Необязательные сервисы: пустой адрес — маршрут не подключается.
	optional := func(raw string) *url.URL {
		if raw == "" {
			return nil
		}
		u, _ := url.Parse(raw)
		return u
	}
	llmURL, notificationsURL, auditURL, realtimeURL := optional(c.LLMURL), optional(c.NotificationsURL), optional(c.AuditURL), optional(c.RealtimeURL)
	// Журнал аудита включается вместе с брокером; без AMQP_URL записи не публикуются.
	var recorder audit.Recorder
	if c.AMQPURL != "" {
		publisher := audit.NewPublisher(c.AMQPURL, log)
		go publisher.Run(ctx)
		recorder = publisher
	}
	verifier, err := auth.NewVerifier(startup, c.OIDCIssuer, c.OIDCClientID, c.OIDCInternalURL)
	if err != nil {
		return err
	}
	sessions := auth.NewSessions(deps.Cache, verifier, c.OIDCIssuer, c.OIDCInternalURL, c.OIDCWebClientID, c.AppOrigin, c.SessionCookieSecure)
	server := &http.Server{Addr: c.Addr, Handler: httpapi.NewGateway(log, deps, verifier, httpapi.Upstreams{Tasks: tasksURL, Files: filesURL, LLM: llmURL, Notifications: notificationsURL, Audit: auditURL, Realtime: realtimeURL}, httpapi.Options{InternalToken: c.InternalToken, Metrics: c.MetricsEnabled, MetricsToken: c.MetricsToken, Audit: recorder, AppOrigin: c.AppOrigin}, sessions), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failed := make(chan error, 1)
	go func() {
		log.Info("API принимает запросы", "addr", c.Addr)
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
