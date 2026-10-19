package accounts

import (
	"context"
	"errors"
	"sync"
	"time"
)

type Service struct {
	Store  Store
	Hasher *Hasher
	// dummy — хеш для несуществующей почты: проверка занимает то же время,
	// и по задержке ответа нельзя узнать, зарегистрирован ли адрес.
	dummyOnce sync.Once
	dummy     string
}

func New(store Store, hasher *Hasher) *Service { return &Service{Store: store, Hasher: hasher} }

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (Account, error) {
	email, err := NormalizeEmail(in.Email)
	if err != nil {
		return Account{}, err
	}
	name, err := NormalizeName(in.Name)
	if err != nil {
		return Account{}, err
	}
	if err = ValidatePassword(in.Password, email); err != nil {
		return Account{}, err
	}
	hash, err := s.Hasher.Hash(ctx, in.Password)
	if err != nil {
		return Account{}, err
	}
	return s.Store.Create(ctx, email, name, hash)
}

func (s *Service) dummyHash(ctx context.Context) string {
	s.dummyOnce.Do(func() { s.dummy, _ = s.Hasher.Hash(ctx, "timing-equalizer-0") })
	return s.dummy
}

// Login проверяет пароль. Порядок важен: сначала пароль, потом блокировка — иначе статус
// учётной записи (и сам факт её существования) узнал бы любой, кто знает только почту.
func (s *Service) Login(ctx context.Context, rawEmail, password string) (Account, error) {
	email, err := NormalizeEmail(rawEmail)
	if err != nil || password == "" || len(password) > 512 {
		return Account{}, ErrInvalidCredentials
	}
	c, err := s.Store.Credentials(ctx, email)
	if errors.Is(err, ErrNotFound) {
		_, _ = s.Hasher.Verify(ctx, password, s.dummyHash(ctx))
		return Account{}, ErrInvalidCredentials
	}
	if err != nil {
		return Account{}, err
	}
	if c.LockedUntil != nil && c.LockedUntil.After(time.Now()) {
		return Account{}, LockedError{Until: *c.LockedUntil}
	}
	ok, err := s.Hasher.Verify(ctx, password, c.Hash)
	if err != nil {
		return Account{}, err
	}
	if !ok {
		if err = s.Store.LoginFailed(ctx, c.ID); err != nil {
			return Account{}, err
		}
		return Account{}, ErrInvalidCredentials
	}
	if c.Status == "blocked" {
		return Account{}, BlockedError{Reason: c.BlockReason}
	}
	if err = s.Store.LoginSucceeded(ctx, c.ID); err != nil {
		return Account{}, err
	}
	now := time.Now()
	c.Account.LastLoginAt = &now
	return c.Account, nil
}

func (s *Service) Get(ctx context.Context, id string) (Account, error) { return s.Store.Get(ctx, id) }

func (s *Service) ChangePassword(ctx context.Context, id, current, next string) error {
	c, err := s.Store.CredentialsByID(ctx, id)
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
	if err = ValidatePassword(next, c.Email); err != nil {
		return err
	}
	hash, err := s.Hasher.Hash(ctx, next)
	if err != nil {
		return err
	}
	return s.Store.SetPassword(ctx, id, hash)
}

func (s *Service) Block(ctx context.Context, id, reason, actor string) (Account, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Account{}, err
	}
	return s.Store.SetBlocked(ctx, id, true, reason, actor)
}

func (s *Service) Unblock(ctx context.Context, id, actor string) (Account, error) {
	return s.Store.SetBlocked(ctx, id, false, "", actor)
}

func (s *Service) List(ctx context.Context, q, status string, limit, offset int) (Page, error) {
	return s.Store.List(ctx, q, status, limit, offset)
}

func (s *Service) Detail(ctx context.Context, id string) (AccountDetail, error) {
	a, err := s.Store.Get(ctx, id)
	if err != nil {
		return AccountDetail{}, err
	}
	h, err := s.Store.History(ctx, id)
	if err != nil {
		return AccountDetail{}, err
	}
	return AccountDetail{Account: a, History: h}, nil
}
