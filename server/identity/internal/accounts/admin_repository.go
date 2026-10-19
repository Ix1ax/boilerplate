package accounts

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const adminColumns = `id::text, email, display_name, role, status, block_reason, created_by::text, last_login_at, created_at, password_changed_at`
const adminSelect = `SELECT ` + adminColumns + ` FROM admins`

func scanAdmin(row pgx.Row, extra ...any) (Admin, error) {
	var a Admin
	err := row.Scan(append([]any{&a.ID, &a.Email, &a.Name, &a.Role, &a.Status, &a.BlockReason, &a.CreatedBy, &a.LastLoginAt, &a.CreatedAt, &a.PasswordChangedAt}, extra...)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	a.Roles = adminRoles(a.Role)
	return a, err
}

func adminHistory(ctx context.Context, tx pgx.Tx, id, action, actor, comment string) error {
	_, err := tx.Exec(ctx, "INSERT INTO admin_history(admin_id,action,actor,comment) VALUES($1,$2,$3,$4)", id, action, actor, comment)
	return err
}

// BootstrapSuperadmin вставляет суперадминистратора, только если его ещё нет. ON CONFLICT без цели
// покрывает оба частичных индекса: «один суперадмин» и «почта занята» — повторный старт ничего не меняет.
func (r *Repository) BootstrapSuperadmin(ctx context.Context, email, name, hash string) (bool, error) {
	created := false
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		id := uuid.NewString()
		tag, err := tx.Exec(ctx, `INSERT INTO admins(id,email,display_name,password_hash,role) VALUES($1,$2,$3,$4,'superadmin') ON CONFLICT DO NOTHING`, id, email, name, hash)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		created = true
		return adminHistory(ctx, tx, id, "created", "system", "Создан при первом запуске из SUPERADMIN_EMAIL")
	})
	return created, err
}

func (r *Repository) CreateAdmin(ctx context.Context, email, name, hash, createdBy, actor string) (Admin, error) {
	id := uuid.NewString()
	var a Admin
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		a, err = scanAdmin(tx.QueryRow(ctx, `INSERT INTO admins(id,email,display_name,password_hash,role,created_by)
VALUES($1,$2,$3,$4,'admin',$5) RETURNING `+adminColumns, id, email, name, hash, createdBy))
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return ErrAdminEmailTaken
		}
		if err != nil {
			return err
		}
		return adminHistory(ctx, tx, id, "created", actor, "")
	})
	return a, err
}

func (r *Repository) AdminCredentials(ctx context.Context, email string) (adminCredentials, error) {
	return r.adminCredentials(ctx, "email=$1 AND status <> 'deleted'", email)
}

func (r *Repository) AdminCredentialsByID(ctx context.Context, id string) (adminCredentials, error) {
	return r.adminCredentials(ctx, "id=$1 AND status <> 'deleted'", id)
}

func (r *Repository) adminCredentials(ctx context.Context, where, arg string) (adminCredentials, error) {
	var c adminCredentials
	a, err := scanAdmin(r.db.QueryRow(ctx, `SELECT `+adminColumns+`, password_hash, locked_until FROM admins WHERE `+where, arg), &c.Hash, &c.LockedUntil)
	c.Admin = a
	return c, err
}

func (r *Repository) GetAdmin(ctx context.Context, id string) (Admin, error) {
	return scanAdmin(r.db.QueryRow(ctx, adminSelect+" WHERE id=$1", id))
}

func (r *Repository) AdminLoginSucceeded(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "UPDATE admins SET failed_logins=0, locked_until=NULL, last_login_at=now() WHERE id=$1", id)
	return err
}

// AdminLoginFailed — тот же атомарный счётчик неудач, что у пользователей.
func (r *Repository) AdminLoginFailed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE admins SET
failed_logins = CASE WHEN failed_logins + 1 >= $2 THEN 0 ELSE failed_logins + 1 END,
locked_until  = CASE WHEN failed_logins + 1 >= $2 THEN now() + $3::interval ELSE locked_until END
WHERE id=$1`, id, maxFailures, lockFor.String())
	return err
}

func (r *Repository) SetAdminPassword(ctx context.Context, id, hash, action, actor string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		// Новая версия пароля закрывает открытые сессии админа в gateway.
		tag, err := tx.Exec(ctx, `UPDATE admins SET password_hash=$2,
password_changed_at=GREATEST(clock_timestamp(),password_changed_at+interval '1 microsecond'),
failed_logins=0, locked_until=NULL, updated_at=now() WHERE id=$1 AND status <> 'deleted'`, id, hash)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return adminHistory(ctx, tx, id, action, actor, "")
	})
}

// SetAdminStatus — блокировка, разблокировка и удаление. Переходы проверяются под блокировкой строки;
// суперадминистратора не трогаем никогда (это же гарантирует CHECK в базе).
func (r *Repository) SetAdminStatus(ctx context.Context, id, to, reason, actor string) (Admin, error) {
	var a Admin
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var role, status string
		err := tx.QueryRow(ctx, "SELECT role, status FROM admins WHERE id=$1 FOR UPDATE", id).Scan(&role, &status)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		switch {
		case role == RoleSuperadmin:
			return ErrSuperadminProtected
		case status == "deleted":
			return ErrAdminDeleted
		case status == to:
			return ErrTransition
		}
		action := map[string]string{"blocked": "blocked", "active": "unblocked", "deleted": "deleted"}[to]
		switch to {
		case "blocked":
			_, err = tx.Exec(ctx, "UPDATE admins SET status='blocked', block_reason=$2, updated_at=now() WHERE id=$1", id, reason)
		case "active":
			_, err = tx.Exec(ctx, "UPDATE admins SET status='active', block_reason='', updated_at=now() WHERE id=$1", id)
		case "deleted":
			_, err = tx.Exec(ctx, "UPDATE admins SET status='deleted', block_reason='', deleted_at=now(), updated_at=now() WHERE id=$1", id)
		default:
			return ErrTransition
		}
		if err != nil {
			return err
		}
		if err = adminHistory(ctx, tx, id, action, actor, reason); err != nil {
			return err
		}
		a, err = scanAdmin(tx.QueryRow(ctx, adminSelect+" WHERE id=$1", id))
		return err
	})
	return a, err
}

// ListAdmins — поиск по почте и имени. Без фильтра удалённые не показываются.
func (r *Repository) ListAdmins(ctx context.Context, q, status string, limit, offset int) (AdminPage, error) {
	pattern := "%" + strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(strings.TrimSpace(q))) + "%"
	where := ` WHERE (email LIKE $1 OR lower(display_name) LIKE $1) AND (($2 = '' AND status <> 'deleted') OR status = $2)`
	page := AdminPage{Items: []Admin{}}
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM admins"+where, pattern, status).Scan(&page.Total); err != nil {
		return page, err
	}
	rows, err := r.db.Query(ctx, adminSelect+where+" ORDER BY (role='superadmin') DESC, created_at DESC, id LIMIT $3 OFFSET $4", pattern, status, limit, offset)
	if err != nil {
		return page, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Admin, error) { return scanAdmin(row) })
	if err != nil {
		return page, err
	}
	page.Items = items
	return page, nil
}

func (r *Repository) AdminHistory(ctx context.Context, id string) ([]HistoryEntry, error) {
	rows, err := r.db.Query(ctx, "SELECT action, actor, comment, created_at FROM admin_history WHERE admin_id=$1 ORDER BY created_at, id", id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (HistoryEntry, error) {
		var h HistoryEntry
		return h, row.Scan(&h.Action, &h.Actor, &h.Comment, &h.CreatedAt)
	})
}
