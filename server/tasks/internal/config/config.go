// Package config читает настройки сервиса из переменных окружения (они описаны в .env.example)
// и сразу проверяет обязательные, чтобы сервис падал при старте, а не на первом запросе.
package config

import (
	"fmt"
	"os"
)

// Config — настройки сервиса. Обязательны DATABASE_URL и INTERNAL_TOKEN.
type Config struct {
	Addr, DatabaseURL, InternalToken string
	// LLMServiceURL необязателен: без него копилот работает только по правилам.
	LLMServiceURL string
	// AMQPURL необязателен: без брокера события копятся в outbox, проверка запускается вручную.
	AMQPURL string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"), LLMServiceURL: os.Getenv("LLM_SERVICE_URL"), AMQPURL: os.Getenv("AMQP_URL")}
	for name, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "INTERNAL_TOKEN": c.InternalToken} {
		if v == "" {
			return c, fmt.Errorf("Не задана переменная %s; выполните make setup", name)
		}
	}
	return c, nil
}
func Value(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
