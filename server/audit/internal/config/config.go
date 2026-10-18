package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr, DatabaseURL, InternalToken string
	// AMQPURL необязателен: без брокера журнал не пополняется.
	AMQPURL string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"), AMQPURL: os.Getenv("AMQP_URL")}
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
