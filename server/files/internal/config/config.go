package config

import (
	"fmt"
	"os"
)

type Config struct {
	Addr, DatabaseURL, InternalToken string
	S3                               S3Config
}

type S3Config struct{ Endpoint, Region, AccessKey, SecretKey, Bucket string }

func Load() (Config, error) {
	c := Config{Addr: Value("HTTP_ADDR", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"), InternalToken: os.Getenv("INTERNAL_TOKEN"),
		S3: S3Config{Endpoint: os.Getenv("S3_ENDPOINT"), Region: Value("S3_REGION", "us-east-1"), AccessKey: os.Getenv("S3_ACCESS_KEY"), SecretKey: os.Getenv("S3_SECRET_KEY"), Bucket: os.Getenv("S3_BUCKET")},
	}
	for name, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "INTERNAL_TOKEN": c.InternalToken, "S3_ENDPOINT": c.S3.Endpoint, "S3_ACCESS_KEY": c.S3.AccessKey, "S3_SECRET_KEY": c.S3.SecretKey, "S3_BUCKET": c.S3.Bucket} {
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
