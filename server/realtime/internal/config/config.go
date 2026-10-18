// Package config читает настройки сервиса из переменных окружения (они описаны в .env.example)
// и сразу проверяет обязательные, чтобы сервис падал при старте, а не на первом запросе.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr, InternalToken string
	// AMQPURL необязателен: без брокера сообщения доходят только до клиентов этой реплики.
	AMQPURL string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), InternalToken: os.Getenv("INTERNAL_TOKEN"), AMQPURL: os.Getenv("AMQP_URL")}
	if c.InternalToken == "" {
		return c, fmt.Errorf("Не задана переменная INTERNAL_TOKEN; выполните make setup")
	}
	return c, nil
}
func Value(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
