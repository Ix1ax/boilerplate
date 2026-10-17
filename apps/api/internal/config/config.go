package config

import (
	"fmt"
	"net/url"
	"os"
)

type Config struct {
	Env, Addr, DatabaseURL, RedisAddr, RedisPassword         string
	S3Endpoint, S3Region, S3AccessKey, S3SecretKey, S3Bucket string
	OIDCIssuer, OIDCClientID, OIDCInternalURL                string
}

func Load() (Config, error) {
	c := Config{
		Env: Value("APP_ENV", "development"), Addr: Value("HTTP_ADDR", ":8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"), RedisAddr: Value("REDIS_ADDR", "localhost:16379"), RedisPassword: os.Getenv("REDIS_PASSWORD"),
		S3Endpoint: os.Getenv("S3_ENDPOINT"), S3Region: Value("S3_REGION", "us-east-1"), S3AccessKey: os.Getenv("S3_ACCESS_KEY"), S3SecretKey: os.Getenv("S3_SECRET_KEY"), S3Bucket: os.Getenv("S3_BUCKET"),
		OIDCIssuer: os.Getenv("OIDC_ISSUER"), OIDCClientID: Value("OIDC_CLIENT_ID", "hackathon-api"), OIDCInternalURL: os.Getenv("OIDC_INTERNAL_URL"),
	}
	for name, v := range map[string]string{"DATABASE_URL": c.DatabaseURL, "REDIS_PASSWORD": c.RedisPassword, "S3_ENDPOINT": c.S3Endpoint, "S3_ACCESS_KEY": c.S3AccessKey, "S3_SECRET_KEY": c.S3SecretKey, "S3_BUCKET": c.S3Bucket, "OIDC_ISSUER": c.OIDCIssuer} {
		if v == "" {
			return c, fmt.Errorf("Не задана переменная %s; выполните make setup", name)
		}
	}
	if c.Env != "development" && c.Env != "production" && c.Env != "test" {
		return c, fmt.Errorf("Недопустимое значение APP_ENV")
	}
	for name, value := range map[string]string{"S3_ENDPOINT": c.S3Endpoint, "OIDC_ISSUER": c.OIDCIssuer, "OIDC_INTERNAL_URL": c.OIDCInternalURL} {
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
