// Сервис llm — шлюз к локальной модели (Ollama + Qwen): генерация целиком и потоком (SSE).
// Своей базы нет. Без запущенной модели сервис работает и на генерацию отвечает 503 —
// так всё приложение не зависит от того, скачана ли модель (make llm).
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

	"example.com/hackathon/server/llm/api"
	"example.com/hackathon/server/llm/internal/config"
	"example.com/hackathon/server/llm/internal/llm"
	"example.com/hackathon/server/llm/internal/ollama"
	"example.com/hackathon/server/llm/internal/platform"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Сервис llm остановлен", "error", err)
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
	// Готовность не зависит от Ollama: без профиля llm сервис работает и отвечает 503 на генерацию.
	r := platform.NewEngine(logger, 1<<20, func(context.Context) error { return nil })
	// Своя спецификация: её читает Swagger в gateway (список «сервисы»). Маршрут без проверки
	// пользователя — сервис доступен только во внутренней сети.
	r.GET("/openapi.yaml", func(c *gin.Context) { c.Data(http.StatusOK, "application/yaml; charset=utf-8", api.Spec) })
	api := r.Group("/api/v1", platform.TrustGateway(c.InternalToken))
	llm.Register(api, llm.New(ollama.New(c.OllamaURL, c.Model, c.Timeout), c.MaxConcurrent, c.MaxPromptRunes), c.Timeout)
	server := &http.Server{Addr: c.Addr, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: c.Timeout + 15*time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	failed := make(chan error, 1)
	go func() {
		logger.Info("Сервис llm принимает запросы", "addr", c.Addr)
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
