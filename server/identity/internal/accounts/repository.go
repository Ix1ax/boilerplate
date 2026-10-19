package accounts

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Пороги защиты от подбора: после maxFailures неверных паролей подряд вход закрыт на lockFor.
// Счётчик хранится в базе, поэтому общий для всех реплик и переживает перезапуск.
const (
	maxFailures = 10
	lockFor     = 15 * time.Minute
)

// credentials — то, что нужно для проверки входа; наружу не уходит.
type credentials struct {
	Account
	Hash        string
	LockedUntil *time.Time
}

// Store — хранилище учётных записей; фейк в тестах сервиса.
type Store interface {
	Create(ctx context.Context, email, name, hash string) (Account, error)
	Credentials(ctx context.Context, email string) (credentials, error)
	CredentialsByID(ctx context.Context, id string) (credentials, error)
	Get(ctx context.Context, id string) (Account, error)
	LoginSucceeded(ctx context.Context, id string) error
	LoginFailed(ctx context.Context, id string) error
	SetPassword(ctx context.Context, id, hash string) error
	SetBlocked(ctx context.Context, id string, blocked bool, reason, actor string) (Account, error)
	List(ctx context.Context, q, status string, limit, offset int) (Page, error)
	History(ctx context.Context, id string) ([]HistoryEntry, error)
}

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

const accountSelect = `SELECT id::text, email, display_name, status, block_reason, last_login_at, created_at FROM accounts`

func scanAccount(row pgx.Row, extra ...any) (Account, error) {
	var a Account
	err := row.Scan(append([]any{&a.ID, &a.Email, &a.Name, &a.Status, &a.BlockReason, &a.LastLoginAt, &a.CreatedAt}, extra...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	a.Roles = userRoles
	return a, err
}

func (r *Repository) inTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func history(ctx context.Context, tx pgx.Tx, id, action, actor, comment string) error {
	_, err := tx.Exec(ctx, "INSERT INTO account_history(account_id,action,actor,comment) VALUES($1,$2,$3,$4)", id, action, actor, comment)
	return err
}

func (r *Repository) Create(ctx context.Context, email, name, hash string) (Account, error) {
	id := uuid.NewString()
	var a Account
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		a, err = scanAccount(tx.QueryRow(ctx, `INSERT INTO accounts(id,email,display_name,password_hash) VALUES($1,$2,$3,$4)
RETURNING id::text, email, display_name, status, block_reason, last_login_at, created_at`, id, email, name, hash))
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return ErrEmailTaken
		}
		if err != nil {
			return err
		}
		return history(ctx, tx, id, "registered", "self", "")
	})
	return a, err
}

func (r *Repository) Credentials(ctx context.Context, email string) (credentials, error) {
	return r.credentials(ctx, "email=$1", email)
}

func (r *Repository) CredentialsByID(ctx context.Context, id string) (credentials, error) {
	return r.credentials(ctx, "id=$1", id)
}

func (r *Repository) credentials(ctx context.Context, where string, arg string) (credentials, error) {
	var c credentials
	a, err := scanAccount(r.db.QueryRow(ctx, `SELECT id::text, email, display_name, status, block_reason, last_login_at, created_at,
password_hash, locked_until FROM accounts WHERE `+where, arg), &c.Hash, &c.LockedUntil)
	c.Account = a
	return c, err
}

func (r *Repository) Get(ctx context.Context, id string) (Account, error) {
	return scanAccount(r.db.QueryRow(ctx, accountSelect+" WHERE id=$1", id))
}

func (r *Repository) LoginSucceeded(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "UPDATE accounts SET failed_logins=0, locked_until=NULL, last_login_at=now() WHERE id=$1", id)
	return err
}

// LoginFailed считает неудачу атомарно в SQL: параллельные попытки не обходят порог.
func (r *Repository) LoginFailed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE accounts SET
failed_logins = CASE WHEN failed_logins + 1 >= $2 THEN 0 ELSE failed_logins + 1 END,
locked_until  = CASE WHEN failed_logins + 1 >= $2 THEN now() + $3::interval ELSE locked_until END
WHERE id=$1`, id, maxFailures, lockFor.String())
	return err
}

func (r *Repository) SetPassword(ctx context.Context, id, hash string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, "UPDATE accounts SET password_hash=$2, password_changed_at=now(), failed_logins=0, locked_until=NULL, updated_at=now() WHERE id=$1", id, hash)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return history(ctx, tx, id, "password_changed", "self", "")
	})
}

func (r *Repository) SetBlocked(ctx context.Context, id string, blocked bool, reason, actor string) (Account, error) {
	var a Account
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var status string
		err := tx.QueryRow(ctx, "SELECT status FROM accounts WHERE id=$1 FOR UPDATE", id).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		next, action := "active", "unblocked"
		if blocked {
			next, action = "blocked", "blocked"
		}
		if status == next {
			return ErrTransition
		}
		if blocked {
			_, err = tx.Exec(ctx, "UPDATE accounts SET status='blocked', block_reason=$2, blocked_by=$3, blocked_at=now(), updated_at=now() WHERE id=$1", id, reason, actor)
		} else {
			_, err = tx.Exec(ctx, "UPDATE accounts SET status='active', block_reason='', blocked_by=NULL, blocked_at=NULL, updated_at=now() WHERE id=$1", id)
		}
		if err != nil {
			return err
		}
		if err = history(ctx, tx, id, action, actor, reason); err != nil {
			return err
		}
		a, err = scanAccount(tx.QueryRow(ctx, accountSelect+" WHERE id=$1", id))
		return err
	})
	return a, err
}

// List — поиск по почте и имени; значения передаются параметрами, LIKE-символы экранируются.
func (r *Repository) List(ctx context.Context, q, status string, limit, offset int) (Page, error) {
	pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(strings.TrimSpace(q))) + "%"
	where := ` WHERE (email LIKE $1 OR lower(display_name) LIKE $1) AND ($2 = '' OR status = $2)`
	page := Page{Items: []Account{}}
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM accounts"+where, pattern, status).Scan(&page.Total); err != nil {
		return page, err
	}
	rows, err := r.db.Query(ctx, accountSelect+where+" ORDER BY created_at DESC, id LIMIT $3 OFFSET $4", pattern, status, limit, offset)
	if err != nil {
		return page, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Account, error) { return scanAccount(row) })
	if err != nil {
		return page, err
	}
	page.Items = items
	return page, nil
}

func (r *Repository) History(ctx context.Context, id string) ([]HistoryEntry, error) {
	rows, err := r.db.Query(ctx, "SELECT action, actor, comment, created_at FROM account_history WHERE account_id=$1 ORDER BY created_at, id", id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (HistoryEntry, error) {
		var h HistoryEntry
		return h, row.Scan(&h.Action, &h.Actor, &h.Comment, &h.CreatedAt)
	})
}
