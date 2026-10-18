package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr, DatabaseURL, InternalToken string
	// AMQPURL необязателен: без брокера API работает, но новые уведомления не приходят.
	AMQPURL string
	// SMTPAddr необязателен: без него письма не отправляются (локально — Mailpit).
	SMTPAddr, MailFrom string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"), AMQPURL: os.Getenv("AMQP_URL"), SMTPAddr: os.Getenv("SMTP_ADDR"), MailFrom: Value("MAIL_FROM", "noreply@example.test")}
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
