package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	Env, Addr, RedisAddr, RedisPassword       string
	OIDCIssuer, OIDCClientID, OIDCInternalURL string
	AppOrigin, OIDCWebClientID                string
	TasksURL, FilesURL, InternalToken         string
	SessionCookieSecure                       bool
}

func Load() (Config, error) {
	c := Config{
		TasksURL: os.Getenv("TASKS_URL"), FilesURL: os.Getenv("FILES_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"),
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
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		return c, fmt.Errorf("Недопустимое значение APP_ENV")
	}
	for name, value := range map[string]string{"OIDC_ISSUER": c.OIDCIssuer, "OIDC_INTERNAL_URL": c.OIDCInternalURL, "TASKS_URL": c.TasksURL, "FILES_URL": c.FilesURL} {
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
