package accounts

import (
	"context"
	"errors"
	"sync"
	"time"
	"unicode/utf8"
)

// Администраторы панели — отдельный контур входа. Учётные записи пользователей и «Столото ID»
// к админке доступа не дают: первого суперадминистратора создаёт сервис при старте, остальных
// админов — суперадминистратор. Сессию выдаёт gateway (своя cookie, не пользовательская).

// Admin — учётная запись администратора без хеша пароля.
type Admin struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	BlockReason string `json:"block_reason"`
	// Roles — роли для gateway: admin есть у всех, superadmin — только у суперадминистратора.
	Roles             []string   `json:"roles"`
	CreatedBy         *string    `json:"created_by"`
	LastLoginAt       *time.Time `json:"last_login_at"`
	CreatedAt         time.Time  `json:"created_at"`
	PasswordChangedAt time.Time  `json:"password_changed_at"`
}

// AdminDetail — карточка администратора с историей.
type AdminDetail struct {
	Admin
	History []HistoryEntry `json:"history"`
}

type AdminPage struct {
	Items []Admin `json:"items"`
	Total int     `json:"total"`
}

const (
	RoleSuperadmin = "superadmin"
	RoleAdmin      = "admin"
)

func adminRoles(role string) []string {
	if role == RoleSuperadmin {
		return []string{RoleAdmin, RoleSuperadmin}
	}
	return []string{RoleAdmin}
}

var (
	ErrAdminEmailTaken = errors.New("Администратор с такой почтой уже есть")
	// ErrSuperadminProtected — суперадминистратора нельзя заблокировать, удалить или сбросить ему пароль
	// из списка админов: свой пароль он меняет сам.
	ErrSuperadminProtected = errors.New("Суперадминистратора нельзя заблокировать, удалить или сменить ему пароль из списка")
	ErrAdminDeleted        = errors.New("Администратор удалён")
)

// ValidateAdminPassword — у админа доступ к данным всех пользователей, поэтому пароль длиннее:
// от 12 символов. Остальные правила — как у пользователей.
func ValidateAdminPassword(password, email string) error {
	if utf8.RuneCountInString(password) < 12 {
		return FieldError{"password", "Пароль администратора — не короче 12 символов"}
	}
	return ValidatePassword(password, email)
}

// adminCredentials — то, что нужно для проверки входа администратора.
type adminCredentials struct {
	Admin
	Hash        string
	LockedUntil *time.Time
}

// AdminStore — хранилище администраторов; фейк в тестах сервиса.
type AdminStore interface {
	BootstrapSuperadmin(ctx context.Context, email, name, hash string) (bool, error)
	CreateAdmin(ctx context.Context, email, name, hash, createdBy, actor string) (Admin, error)
	AdminCredentials(ctx context.Context, email string) (adminCredentials, error)
	AdminCredentialsByID(ctx context.Context, id string) (adminCredentials, error)
	GetAdmin(ctx context.Context, id string) (Admin, error)
	AdminLoginSucceeded(ctx context.Context, id string) error
	AdminLoginFailed(ctx context.Context, id string) error
	SetAdminPassword(ctx context.Context, id, hash, action, actor string) error
	SetAdminStatus(ctx context.Context, id, to, reason, actor string) (Admin, error)
	ListAdmins(ctx context.Context, q, status string, limit, offset int) (AdminPage, error)
	AdminHistory(ctx context.Context, id string) ([]HistoryEntry, error)
}

type Admins struct {
	Store     AdminStore
	Hasher    *Hasher
	dummyOnce sync.Once
	dummy     string
}

func NewAdmins(store AdminStore, hasher *Hasher) *Admins {
	return &Admins{Store: store, Hasher: hasher}
}

// Bootstrap создаёт суперадминистратора из окружения, если его ещё нет. Уже существующего не трогает:
// дальше пароль меняют в админке, а не переменной окружения. true — суперадминистратор создан сейчас.
func (s *Admins) Bootstrap(ctx context.Context, rawEmail, password, rawName string) (bool, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return false, errors.New("SUPERADMIN_EMAIL: " + err.Error())
	}
	name, err := NormalizeName(rawName)
	if err != nil {
		return false, errors.New("SUPERADMIN_NAME: " + err.Error())
	}
	if err = ValidateAdminPassword(password, email); err != nil {
		return false, errors.New("SUPERADMIN_PASSWORD: " + err.Error())
	}
	hash, err := s.Hasher.Hash(ctx, password)
	if err != nil {
		return false, err
	}
	return s.Store.BootstrapSuperadmin(ctx, email, name, hash)
}

type AdminInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Create — новый администратор; создаёт только суперадминистратор (проверяет маршрут).
func (s *Admins) Create(ctx context.Context, in AdminInput, createdBy, actor string) (Admin, error) {
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		return Admin{}, err
	}
	name, err := NormalizeName(in.Name)
	if err != nil {
		return Admin{}, err
	}
	if err = ValidateAdminPassword(in.Password, email); err != nil {
		return Admin{}, err
	}
	hash, err := s.Hasher.Hash(ctx, in.Password)
	if err != nil {
		return Admin{}, err
	}
	return s.Store.CreateAdmin(ctx, email, name, hash, createdBy, actor)
}

func (s *Admins) dummyHash(ctx context.Context) string {
	s.dummyOnce.Do(func() { s.dummy, _ = s.Hasher.Hash(ctx, "timing-equalizer-admin-0") })
	return s.dummy
}

// Login — тот же порядок, что у пользователей: сначала пароль, потом блокировка, чтобы по ответу
// нельзя было узнать статус чужой учётной записи.
func (s *Admins) Login(ctx context.Context, rawEmail, password string) (Admin, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil || password == "" || len(password) > 512 {
		return Admin{}, ErrInvalidCredentials
	}
	c, err := s.Store.AdminCredentials(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_, _ = s.Hasher.Verify(ctx, password, s.dummyHash(ctx))
		return Admin{}, ErrInvalidCredentials
	}
	if err != nil {
		return Admin{}, err
	}
	if c.LockedUntil != nil && c.LockedUntil.After(time.Now()) {
		return Admin{}, LockedError{Until: *c.LockedUntil}
	}
	ok, err := s.Hasher.Verify(ctx, password, c.Hash)
	if err != nil {
		return Admin{}, err
	}
	if !ok {
		if err = s.Store.AdminLoginFailed(ctx, c.ID); err != nil {
			return Admin{}, err
		}
		return Admin{}, ErrInvalidCredentials
	}
	if c.Status == "blocked" {
		return Admin{}, BlockedError{Reason: c.BlockReason}
	}
	if err = s.Store.AdminLoginSucceeded(ctx, c.ID); err != nil {
		return Admin{}, err
	}
	now := time.Now()
	c.Admin.LastLoginAt = &now
	return c.Admin, nil
}

// Get — актуальное состояние для проверки сессии в gateway (удалённый тоже возвращается, со статусом).
func (s *Admins) Get(ctx context.Context, id string) (Admin, error) { return s.Store.GetAdmin(ctx, id) }

// ChangePassword — админ меняет свой пароль, зная текущий.
func (s *Admins) ChangePassword(ctx context.Context, id, current, next string) error {
	c, err := s.Store.AdminCredentialsByID(ctx, id)
	if err != nil {
		return err
	}
	if c.Status == "blocked" {
		return BlockedError{Reason: c.BlockReason}
	}
	ok, err := s.Hasher.Verify(ctx, current, c.Hash)
	if err != nil {
		return err
	}
	if !ok {
		return ErrWrongPassword
	}
	if current == next {
		return ErrSamePassword
	}
	if err = ValidateAdminPassword(next, c.Email); err != nil {
		var fe FieldError
		if errors.As(err, &fe) {
			return FieldError{"new_password", fe.Message}
		}
		return err
	}
	hash, err := s.Hasher.Hash(ctx, next)
	if err != nil {
		return err
	}
	return s.Store.SetAdminPassword(ctx, id, hash, "password_changed", "self")
}

// ResetPassword — суперадминистратор задаёт админу новый пароль (забыл, подозрение на утечку).
// Открытые сессии админа закрываются: gateway сверяет password_changed_at.
func (s *Admins) ResetPassword(ctx context.Context, id, password, actor string) (Admin, error) {
	a, err := s.Store.GetAdmin(ctx, id)
	if err != nil {
		return Admin{}, err
	}
	switch {
	case a.Role == RoleSuperadmin:
		return Admin{}, ErrSuperadminProtected
	case a.Status == "deleted":
		return Admin{}, ErrAdminDeleted
	}
	if err = ValidateAdminPassword(password, a.Email); err != nil {
		return Admin{}, err
	}
	hash, err := s.Hasher.Hash(ctx, password)
	if err != nil {
		return Admin{}, err
	}
	if err = s.Store.SetAdminPassword(ctx, id, hash, "password_reset", actor); err != nil {
		return Admin{}, err
	}
	return s.Store.GetAdmin(ctx, id)
}

func (s *Admins) Block(ctx context.Context, id, reason, actor string) (Admin, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Admin{}, err
	}
	return s.Store.SetAdminStatus(ctx, id, "blocked", reason, actor)
}

func (s *Admins) Unblock(ctx context.Context, id, actor string) (Admin, error) {
	return s.Store.SetAdminStatus(ctx, id, "active", "", actor)
}

func (s *Admins) Delete(ctx context.Context, id, actor string) (Admin, error) {
	return s.Store.SetAdminStatus(ctx, id, "deleted", "", actor)
}

func (s *Admins) List(ctx context.Context, q, status string, limit, offset int) (AdminPage, error) {
	return s.Store.ListAdmins(ctx, q, status, limit, offset)
}

func (s *Admins) Detail(ctx context.Context, id string) (AdminDetail, error) {
	a, err := s.Store.GetAdmin(ctx, id)
	if err != nil {
		return AdminDetail{}, err
	}
	h, err := s.Store.AdminHistory(ctx, id)
	if err != nil {
		return AdminDetail{}, err
	}
	return AdminDetail{Admin: a, History: h}, nil
}
