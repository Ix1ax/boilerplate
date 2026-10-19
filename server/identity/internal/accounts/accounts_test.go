package accounts

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHasherRoundTrip(t *testing.T) {
	h := NewHasher(1)
	ctx := context.Background()
	hash, err := h.Hash(ctx, "Пароль-2026")
	if err != nil || !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("хеш %q, %v", hash, err)
	}
	if ok, err := h.Verify(ctx, "Пароль-2026", hash); !ok || err != nil {
		t.Fatalf("верный пароль не принят: %v", err)
	}
	if ok, _ := h.Verify(ctx, "пароль-2026", hash); ok {
		t.Fatal("пароль другого регистра принят")
	}
	other, _ := h.Hash(ctx, "Пароль-2026")
	if other == hash {
		t.Fatal("соль не случайна: одинаковые хеши")
	}
	for _, bad := range []string{"", "plain", "$argon2i$v=19$m=1,t=1,p=1$a$b", "$argon2id$v=19$m=999999999,t=1,p=1$YQ$YQ"} {
		if _, err := h.Verify(ctx, "x", bad); err == nil {
			t.Fatalf("испорченный хеш %q принят", bad)
		}
	}
}

func TestValidation(t *testing.T) {
	for raw, want := range map[string]string{" Anna@Example.RU ": "anna@example.ru", "a.b+c@mail.co": "a.b+c@mail.co"} {
		if got, err := NormalizeEmail(raw); err != nil || got != want {
			t.Fatalf("%q → %q, %v", raw, got, err)
		}
	}
	for _, bad := range []string{"", "anna", "anna@", "anna@localhost", "Анна <anna@example.ru>", strings.Repeat("a", 250) + "@x.ru"} {
		if _, err := NormalizeEmail(bad); err == nil {
			t.Fatalf("почта %q принята", bad)
		}
	}
	if name, err := NormalizeName("  Анна   Смирнова "); err != nil || name != "Анна Смирнова" {
		t.Fatalf("имя %q, %v", name, err)
	}
	for _, bad := range []string{"", "А", "a\x00b"} {
		if _, err := NormalizeName(bad); err == nil {
			t.Fatalf("имя %q принято", bad)
		}
	}
	if err := ValidatePassword("ЗарядкаВоДворе7", "anna@example.ru"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"short1", "onlyletters", "1234567890", "Password1", "anna@example.ru", strings.Repeat("a1", 70)} {
		if err := ValidatePassword(bad, "anna@example.ru"); err == nil {
			t.Fatalf("пароль %q принят", bad)
		}
	}
	if err := ValidatePassword("anna12345", "anna12345@example.ru"); err == nil {
		t.Fatal("пароль, совпадающий с именем почты, принят")
	}
}

// fakeStore — хранилище в памяти: проверяем порядок решений сервиса без базы.
type fakeStore struct {
	byEmail map[string]*credentials
	failed  map[string]int
	ok      map[string]int
}

func newFake() *fakeStore {
	return &fakeStore{byEmail: map[string]*credentials{}, failed: map[string]int{}, ok: map[string]int{}}
}

func (f *fakeStore) Create(_ context.Context, email, name, hash string) (Account, error) {
	if _, ok := f.byEmail[email]; ok {
		return Account{}, ErrEmailTaken
	}
	c := &credentials{Account: Account{ID: "id-" + email, Email: email, Name: name, Status: "active", Roles: userRoles}, Hash: hash}
	f.byEmail[email] = c
	return c.Account, nil
}
func (f *fakeStore) Credentials(_ context.Context, email string) (credentials, error) {
	if c, ok := f.byEmail[email]; ok {
		return *c, nil
	}
	return credentials{}, ErrNotFound
}
func (f *fakeStore) find(id string) *credentials {
	for _, c := range f.byEmail {
		if c.ID == id {
			return c
		}
	}
	return nil
}
func (f *fakeStore) CredentialsByID(_ context.Context, id string) (credentials, error) {
	if c := f.find(id); c != nil {
		return *c, nil
	}
	return credentials{}, ErrNotFound
}
func (f *fakeStore) Get(ctx context.Context, id string) (Account, error) {
	c, err := f.CredentialsByID(ctx, id)
	return c.Account, err
}
func (f *fakeStore) LoginSucceeded(_ context.Context, id string) error { f.ok[id]++; return nil }
func (f *fakeStore) LoginFailed(_ context.Context, id string) error    { f.failed[id]++; return nil }
func (f *fakeStore) SetPassword(_ context.Context, id, hash string) error {
	f.find(id).Hash = hash
	return nil
}
func (f *fakeStore) SetBlocked(_ context.Context, id string, blocked bool, reason, _ string) (Account, error) {
	c := f.find(id)
	if (c.Status == "blocked") == blocked {
		return Account{}, ErrTransition
	}
	c.Status, c.BlockReason = map[bool]string{true: "blocked", false: "active"}[blocked], reason
	return c.Account, nil
}
func (f *fakeStore) List(context.Context, string, string, int, int) (Page, error) { return Page{}, nil }
func (f *fakeStore) History(context.Context, string) ([]HistoryEntry, error)      { return nil, nil }

func TestLoginFlow(t *testing.T) {
	ctx := context.Background()
	store := newFake()
	s := New(store, NewHasher(2))
	a, err := s.Register(ctx, RegisterInput{Email: "Anna@Example.ru", Password: "ЗарядкаВоДворе7", Name: "Анна"})
	if err != nil || a.Email != "anna@example.ru" || a.Roles[0] != "user" || len(a.Roles) != 1 {
		t.Fatalf("регистрация: %+v, %v", a, err)
	}
	if _, err = s.Register(ctx, RegisterInput{Email: "anna@example.ru", Password: "ДругойПароль8", Name: "Анна"}); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("повторная регистрация: %v", err)
	}
	if _, err = s.Login(ctx, "ANNA@example.ru", "ЗарядкаВоДворе7"); err != nil {
		t.Fatalf("вход: %v", err)
	}
	if _, err = s.Login(ctx, "anna@example.ru", "неверный1"); !errors.Is(err, ErrInvalidCredentials) || store.failed[a.ID] != 1 {
		t.Fatalf("неверный пароль: %v, неудач %d", err, store.failed[a.ID])
	}
	// Неизвестная почта и неверный пароль неотличимы для клиента.
	if _, err = s.Login(ctx, "nobody@example.ru", "ЗарядкаВоДворе7"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("неизвестная почта: %v", err)
	}
	if _, err = s.Block(ctx, a.ID, "  ", "admin"); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("блокировка без причины: %v", err)
	}
	if _, err = s.Block(ctx, a.ID, "Спам", "admin"); err != nil {
		t.Fatal(err)
	}
	// Причину блокировки видит только знающий пароль.
	if _, err = s.Login(ctx, "anna@example.ru", "неверный1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("заблокированная + неверный пароль: %v", err)
	}
	var blocked BlockedError
	if _, err = s.Login(ctx, "anna@example.ru", "ЗарядкаВоДворе7"); !errors.As(err, &blocked) || blocked.Reason != "Спам" {
		t.Fatalf("заблокированная: %v", err)
	}
	if _, err = s.Unblock(ctx, a.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	until := time.Now().Add(time.Minute)
	store.byEmail["anna@example.ru"].LockedUntil = &until
	var locked LockedError
	if _, err = s.Login(ctx, "anna@example.ru", "ЗарядкаВоДворе7"); !errors.As(err, &locked) {
		t.Fatalf("временная блокировка: %v", err)
	}
}

func TestChangePassword(t *testing.T) {
	ctx := context.Background()
	s := New(newFake(), NewHasher(1))
	a, _ := s.Register(ctx, RegisterInput{Email: "igor@example.ru", Password: "Пробежка2026", Name: "Игорь"})
	if err := s.ChangePassword(ctx, a.ID, "не тот", "Новый2027пароль"); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("неверный текущий: %v", err)
	}
	if err := s.ChangePassword(ctx, a.ID, "Пробежка2026", "Пробежка2026"); !errors.Is(err, ErrSamePassword) {
		t.Fatalf("тот же пароль: %v", err)
	}
	var fe FieldError
	if err := s.ChangePassword(ctx, a.ID, "Пробежка2026", "short"); !errors.As(err, &fe) {
		t.Fatalf("слабый пароль: %v", err)
	}
	if err := s.ChangePassword(ctx, a.ID, "Пробежка2026", "Новый2027пароль"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Login(ctx, "igor@example.ru", "Новый2027пароль"); err != nil {
		t.Fatalf("вход с новым паролем: %v", err)
	}
}
