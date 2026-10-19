// Package config читает настройки сервиса identity из окружения и сразу проверяет обязательные.
package config

import (
	"fmt"
	"os"
)

// Config — настройки сервиса. Обязательны DATABASE_URL, INTERNAL_TOKEN и учётные данные
// суперадминистратора: без него в админку войти некому.
type Config struct {
	Addr, DatabaseURL, InternalToken string
	// Суперадминистратор создаётся при первом старте; дальше пароль меняют в админке, не здесь.
	SuperadminEmail, SuperadminPassword, SuperadminName string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"),
		SuperadminEmail: os.Getenv("SUPERADMIN_EMAIL"), SuperadminPassword: os.Getenv("SUPERADMIN_PASSWORD"), SuperadminName: Value("SUPERADMIN_NAME", "Суперадминистратор")}
	for name, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "INTERNAL_TOKEN": c.InternalToken, "SUPERADMIN_EMAIL": c.SuperadminEmail, "SUPERADMIN_PASSWORD": c.SuperadminPassword} {
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
