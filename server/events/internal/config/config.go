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
	// AMQPURL необязателен: без брокера уведомления копятся в outbox и уйдут позже.
	AMQPURL string
	// WeatherURL — прогноз Open-Meteo для уличных встреч; пусто — публичный API, off — выключить.
	WeatherURL                                                                        string
	StoriesEndpoint, StoriesRegion, StoriesAccessKey, StoriesSecretKey, StoriesBucket string
}

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"), AMQPURL: os.Getenv("AMQP_URL"), WeatherURL: os.Getenv("WEATHER_URL")}
	c.StoriesEndpoint, c.StoriesRegion = os.Getenv("S3_ENDPOINT"), Value("S3_REGION", "us-east-1")
	c.StoriesAccessKey, c.StoriesSecretKey, c.StoriesBucket = os.Getenv("S3_ACCESS_KEY"), os.Getenv("S3_SECRET_KEY"), os.Getenv("STORIES_S3_BUCKET")
	if c.StoriesEndpoint != "" && (c.StoriesAccessKey == "" || c.StoriesSecretKey == "" || c.StoriesBucket == "") {
		return c, fmt.Errorf("Для историй задайте S3_ACCESS_KEY, S3_SECRET_KEY и STORIES_S3_BUCKET")
	}
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
