package notifications

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// ErrLinkCode — код привязки не найден, уже использован или просрочен.
var ErrLinkCode = errors.New("Ссылка для подключения недействительна или устарела")

// CreateLinkCode сохраняет одноразовый код привязки MAX для пользователя.
func (r *Repository) CreateLinkCode(ctx context.Context, user, code string, ttl time.Duration) error {
	_, err := r.db.Exec(ctx, "INSERT INTO max_link_codes(code,user_id,expires_at) VALUES($1,$2,now()+$3::interval)", code, user, ttl.String())
	return err
}

// ConsumeLinkCode погашает код (повторно им воспользоваться нельзя) и возвращает пользователя.
func (r *Repository) ConsumeLinkCode(ctx context.Context, code string) (string, error) {
	var user string
	err := r.db.QueryRow(ctx, "DELETE FROM max_link_codes WHERE code=$1 AND expires_at > now() RETURNING user_id", code).Scan(&user)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrLinkCode
	}
	return user, err
}

// Link привязывает MAX-аккаунт. Если он был привязан к другому пользователю, привязка переносится.
func (r *Repository) Link(ctx context.Context, user string, maxUserID int64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "DELETE FROM max_links WHERE max_user_id=$1 AND user_id<>$2", maxUserID, user); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO max_links(user_id,max_user_id) VALUES($1,$2)
ON CONFLICT (user_id) DO UPDATE SET max_user_id=EXCLUDED.max_user_id, linked_at=now()`, user, maxUserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UnlinkMax — отвязка по команде /stop из мессенджера.
func (r *Repository) UnlinkMax(ctx context.Context, maxUserID int64) (bool, error) {
	tag, err := r.db.Exec(ctx, "DELETE FROM max_links WHERE max_user_id=$1", maxUserID)
	return err == nil && tag.RowsAffected() > 0, err
}

// UnlinkUser — отвязка из интерфейса приложения.
func (r *Repository) UnlinkUser(ctx context.Context, user string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM max_links WHERE user_id=$1", user)
	return err
}

// MaxUserID возвращает привязанный MAX-аккаунт пользователя (ok=false — не привязан).
func (r *Repository) MaxUserID(ctx context.Context, user string) (int64, bool, error) {
	var id int64
	err := r.db.QueryRow(ctx, "SELECT max_user_id FROM max_links WHERE user_id=$1", user).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	return id, err == nil, err
}
