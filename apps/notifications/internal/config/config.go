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
	// MAX: без токена канал выключен. Токен выдаётся после верификации организации и модерации бота.
	MaxBotToken, MaxBotUsername, MaxAPIURL string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"), AMQPURL: os.Getenv("AMQP_URL"), SMTPAddr: os.Getenv("SMTP_ADDR"), MailFrom: Value("MAIL_FROM", "noreply@example.test"),
		MaxBotToken: os.Getenv("MAX_BOT_TOKEN"), MaxBotUsername: os.Getenv("MAX_BOT_USERNAME"), MaxAPIURL: os.Getenv("MAX_API_URL")}
	for name, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "INTERNAL_TOKEN": c.InternalToken} {
		if v == "" {
			return c, fmt.Errorf("Не задана переменная %s; выполните make setup", name)
		}
	}
	if c.MaxBotToken != "" && c.MaxBotUsername == "" {
		return c, fmt.Errorf("Для MAX укажите MAX_BOT_USERNAME — имя бота для ссылки привязки")
	}
	return c, nil
}
func Value(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
