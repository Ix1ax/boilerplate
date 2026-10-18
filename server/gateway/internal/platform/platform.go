// Package platform — подключение gateway к Redis (сессии и счётчики лимитов).
package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"example.com/hackathon/server/gateway/internal/config"
)

// Gateway хранит в Redis только сессии и счётчики; данных предметной области у него нет.
type Dependencies struct{ Cache *redis.Client }

// Open подключается к Redis и сразу проверяет связь: без Redis gateway не может хранить сессии.
func Open(ctx context.Context, c config.Config) (*Dependencies, error) {
	d := &Dependencies{Cache: redis.NewClient(&redis.Options{Addr: c.RedisAddr, Password: c.RedisPassword, DialTimeout: 3 * time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 2 * time.Second})}
	if err := d.Check(ctx); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}
func (d *Dependencies) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := d.Cache.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("Redis недоступен")
	}
	return nil
}
func (d *Dependencies) Close() { d.Cache.Close() }
