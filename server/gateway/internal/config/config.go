// Package config читает настройки gateway из переменных окружения (см. .env.example)
// и проверяет их при старте, включая правила для production (HTTPS, защищённые cookie, токен метрик).
package config

import (
	"fmt"
	"net/url"
	"os"
)

// Config — настройки gateway. Адреса LLM, уведомлений, журнала и realtime необязательны:
// пустой адрес просто отключает соответствующие маршруты.
type Config struct {
	Env, Addr, RedisAddr, RedisPassword       string
	OIDCIssuer, OIDCClientID, OIDCInternalURL string
	AppOrigin, OIDCWebClientID                string
	TasksURL, FilesURL, LLMURL, InternalToken string
	NotificationsURL, AuditURL, AMQPURL       string
	RealtimeURL, EventsURL                    string
	MetricsToken                              string
	MetricsEnabled                            bool
	SessionCookieSecure                       bool
}

func Load() (Config, error) {
	c := Config{
		TasksURL: os.Getenv("TASKS_URL"), FilesURL: os.Getenv("FILES_URL"), LLMURL: os.Getenv("LLM_SERVICE_URL"), NotificationsURL: os.Getenv("NOTIFICATIONS_URL"), AuditURL: os.Getenv("AUDIT_URL"), RealtimeURL: os.Getenv("REALTIME_URL"), EventsURL: os.Getenv("EVENTS_URL"), AMQPURL: os.Getenv("AMQP_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"), MetricsToken: os.Getenv("METRICS_TOKEN"), MetricsEnabled: os.Getenv("METRICS_ENABLED") != "false",
		AppOrigin: Value("APP_ORIGIN", "http://localhost:3000"), OIDCWebClientID: Value("OIDC_WEB_CLIENT_ID", "hackathon-web"), SessionCookieSecure: os.Getenv("SESSION_COOKIE_SECURE") == "true",
		Env: Value("APP_ENV", "development"), Addr: Value("HTTP_ADDR", ":8080"),
		RedisAddr: Value("REDIS_ADDR", "localhost:6379"), RedisPassword: os.Getenv("REDIS_PASSWORD"),
		OIDCIssuer: os.Getenv("OIDC_ISSUER"), OIDCClientID: Value("OIDC_CLIENT_ID", "hackathon-api"), OIDCInternalURL: os.Getenv("OIDC_INTERNAL_URL"),
	}
	required := map[string]string{"REDIS_PASSWORD": c.RedisPassword, "OIDC_ISSUER": c.OIDCIssuer, "TASKS_URL": c.TasksURL, "FILES_URL": c.FilesURL, "INTERNAL_TOKEN": c.InternalToken}
	for name, v := range required {
		if v == "" {
			return c, fmt.Errorf("Не задана переменная %s; выполните make setup", name)
		}
	}
	origin, err := url.Parse(c.AppOrigin)
	if err != nil || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return c, fmt.Errorf("APP_ORIGIN должен содержать только схему и адрес приложения")
	}
	if c.Env == "production" && (origin.Scheme != "https" || !c.SessionCookieSecure) {
		return c, fmt.Errorf("В production нужны HTTPS APP_ORIGIN и SESSION_COOKIE_SECURE=true")
	}
	if c.Env == "production" && c.MetricsEnabled && c.MetricsToken == "" {
		return c, fmt.Errorf("В production для /metrics нужен METRICS_TOKEN или METRICS_ENABLED=false")
	}
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		return c, fmt.Errorf("Недопустимое значение APP_ENV")
	}
	for name, value := range map[string]string{"OIDC_ISSUER": c.OIDCIssuer, "OIDC_INTERNAL_URL": c.OIDCInternalURL, "TASKS_URL": c.TasksURL, "FILES_URL": c.FilesURL, "LLM_SERVICE_URL": c.LLMURL, "NOTIFICATIONS_URL": c.NotificationsURL, "AUDIT_URL": c.AuditURL, "REALTIME_URL": c.RealtimeURL, "EVENTS_URL": c.EventsURL} {
		if value == "" {
			continue
		}
		u, err := url.Parse(value)
		if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
			return c, fmt.Errorf("Некорректное значение %s", name)
		}
		if name == "OIDC_ISSUER" && c.Env == "production" && u.Scheme != "https" {
			return c, fmt.Errorf("В production адрес OIDC_ISSUER должен использовать HTTPS")
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
