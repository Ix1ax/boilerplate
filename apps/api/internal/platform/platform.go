package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"example.com/hackathon/apps/api/internal/config"
)

type Dependencies struct {
	DB      *pgxpool.Pool
	Cache   *redis.Client
	Objects *s3.Client
	Bucket  string
}

func Open(ctx context.Context, c config.Config) (*Dependencies, error) {
	cfg, err := pgxpool.ParseConfig(c.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("Некорректные настройки подключения к базе данных")
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("Не удалось создать пул подключений к базе данных: %w", err)
	}
	d := &Dependencies{DB: db, Cache: redis.NewClient(&redis.Options{Addr: c.RedisAddr, Password: c.RedisPassword, DialTimeout: 3 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second}), Bucket: c.S3Bucket}
	d.Objects = s3.New(s3.Options{Region: c.S3Region, BaseEndpoint: aws.String(c.S3Endpoint), UsePathStyle: true, Credentials: credentials.NewStaticCredentialsProvider(c.S3AccessKey, c.S3SecretKey, ""), RetryMaxAttempts: 2})
	if err := d.Check(ctx); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
func (d *Dependencies) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := d.DB.Ping(ctx); err != nil {
		return fmt.Errorf("Postgres недоступен")
	}
	if err := d.Cache.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis недоступен")
	}
	if _, err := d.Objects.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(d.Bucket)}); err != nil {
		return fmt.Errorf("Хранилище файлов недоступно")
	}
	return nil
}
func (d *Dependencies) Close() { d.Cache.Close(); d.DB.Close() }
