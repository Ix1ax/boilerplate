package accounts

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/hackathon/server/identity/internal/platform"
	"github.com/gin-gonic/gin"
)

// fakeAdmins — хранилище администраторов в памяти: проверяем решения сервиса без базы.
type fakeAdmins struct {
	byID   map[string]*adminCredentials
	failed map[string]int
	log    []string
}

func newFakeAdmins() *fakeAdmins {
	return &fakeAdmins{byID: map[string]*adminCredentials{}, failed: map[string]int{}}
}

func (f *fakeAdmins) byEmail(email string) *adminCredentials {
	for _, c := range f.byID {
		if c.Email == email && c.Status != "deleted" {
			return c
		}
	}
	return nil
}
func (f *fakeAdmins) add(email, name, hash, role string) *adminCredentials {
	c := &adminCredentials{Admin: Admin{ID: "id-" + email, Email: email, Name: name, Role: role, Status: "active", Roles: adminRoles(role)}, Hash: hash}
	f.byID[c.ID] = c
	return c
}
func (f *fakeAdmins) BootstrapSuperadmin(_ context.Context, email, name, hash string) (bool, error) {
	for _, c := range f.byID {
		if c.Role == RoleSuperadmin {
			return false, nil
		}
	}
	f.add(email, name, hash, RoleSuperadmin)
	return true, nil
}
func (f *fakeAdmins) CreateAdmin(_ context.Context, email, name, hash, _, _ string) (Admin, error) {
	if f.byEmail(email) != nil {
		return Admin{}, ErrAdminEmailTaken
	}
	return f.add(email, name, hash, RoleAdmin).Admin, nil
}
func (f *fakeAdmins) AdminCredentials(_ context.Context, email string) (adminCredentials, error) {
	if c := f.byEmail(email); c != nil {
		return *c, nil
	}
	return adminCredentials{}, ErrNotFound
}
func (f *fakeAdmins) AdminCredentialsByID(_ context.Context, id string) (adminCredentials, error) {
	if c, ok := f.byID[id]; ok && c.Status != "deleted" {
		return *c, nil
	}
	return adminCredentials{}, ErrNotFound
}
func (f *fakeAdmins) GetAdmin(_ context.Context, id string) (Admin, error) {
	if c, ok := f.byID[id]; ok {
		return c.Admin, nil
	}
	return Admin{}, ErrNotFound
}
func (f *fakeAdmins) AdminLoginSucceeded(context.Context, string) error { return nil }
func (f *fakeAdmins) AdminLoginFailed(_ context.Context, id string) error {
	f.failed[id]++
	return nil
}
func (f *fakeAdmins) SetAdminPassword(_ context.Context, id, hash, action, _ string) error {
	f.byID[id].Hash = hash
	f.log = append(f.log, action)
	return nil
}
func (f *fakeAdmins) SetAdminStatus(_ context.Context, id, to, reason, _ string) (Admin, error) {
	c, ok := f.byID[id]
	switch {
	case !ok:
		return Admin{}, ErrNotFound
	case c.Role == RoleSuperadmin:
		return Admin{}, ErrSuperadminProtected
	case c.Status == "deleted":
		return Admin{}, ErrAdminDeleted
	case c.Status == to:
		return Admin{}, ErrTransition
	}
	c.Status, c.BlockReason = to, reason
	return c.Admin, nil
}
func (f *fakeAdmins) ListAdmins(context.Context, string, string, int, int) (AdminPage, error) {
	return AdminPage{}, nil
}
func (f *fakeAdmins) AdminHistory(context.Context, string) ([]HistoryEntry, error) { return nil, nil }

func TestAdminPasswordRules(t *testing.T) {
	if err := ValidateAdminPassword("Модерация2026", "boss@example.ru"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"Короткий1", "длинныйбезцифр", "123456789012"} {
		if err := ValidateAdminPassword(bad, "boss@example.ru"); err == nil {
			t.Fatalf("пароль администратора %q принят", bad)
		}
	}
}

func TestAdminsFlow(t *testing.T) {
	ctx := context.Background()
	store := newFakeAdmins()
	s := NewAdmins(store, NewHasher(2))
	if _, err := s.Bootstrap(ctx, "boss@example.ru", "short", "Босс"); err == nil {
		t.Fatal("слабый пароль суперадминистратора принят")
	}
	created, err := s.Bootstrap(ctx, "Boss@Example.ru", "ГлавныйПароль2026", "Главный Админ")
	if err != nil || !created {
		t.Fatalf("бутстрап: %v, %v", created, err)
	}
	// Повторный старт с другим паролем не меняет существующего суперадминистратора.
	if created, err = s.Bootstrap(ctx, "boss@example.ru", "ДругойПароль2027", "Главный Админ"); err != nil || created {
		t.Fatalf("повторный бутстрап: %v, %v", created, err)
	}
	boss, err := s.Login(ctx, "boss@example.ru", "ГлавныйПароль2026")
	if err != nil || boss.Role != RoleSuperadmin || len(boss.Roles) != 2 {
		t.Fatalf("вход суперадминистратора: %+v, %v", boss, err)
	}
	if _, err = s.Create(ctx, AdminInput{Email: "mod@example.ru", Name: "Модератор", Password: "короткий1"}, boss.ID, "boss"); err == nil {
		t.Fatal("админ с коротким паролем создан")
	}
	mod, err := s.Create(ctx, AdminInput{Email: "mod@example.ru", Name: "Модератор", Password: "Модерация2026"}, boss.ID, "boss")
	if err != nil || mod.Role != RoleAdmin || len(mod.Roles) != 1 {
		t.Fatalf("создание админа: %+v, %v", mod, err)
	}
	if _, err = s.Create(ctx, AdminInput{Email: "MOD@example.ru", Name: "Ещё", Password: "Модерация2026"}, boss.ID, "boss"); !errors.Is(err, ErrAdminEmailTaken) {
		t.Fatalf("повтор почты: %v", err)
	}
	if _, err = s.Login(ctx, "mod@example.ru", "неверный12345"); !errors.Is(err, ErrInvalidCredentials) || store.failed[mod.ID] != 1 {
		t.Fatalf("неверный пароль: %v", err)
	}
	for _, act := range []func() (Admin, error){
		func() (Admin, error) { return s.Block(ctx, boss.ID, "Нельзя", "boss") },
		func() (Admin, error) { return s.Delete(ctx, boss.ID, "boss") },
		func() (Admin, error) { return s.ResetPassword(ctx, boss.ID, "НовыйПароль2026", "boss") },
	} {
		if _, err = act(); !errors.Is(err, ErrSuperadminProtected) {
			t.Fatalf("действие над суперадминистратором: %v", err)
		}
	}
	if _, err = s.Block(ctx, mod.ID, " ", "boss"); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("блокировка без причины: %v", err)
	}
	if _, err = s.Block(ctx, mod.ID, "Отпуск", "boss"); err != nil {
		t.Fatal(err)
	}
	var blocked BlockedError
	if _, err = s.Login(ctx, "mod@example.ru", "Модерация2026"); !errors.As(err, &blocked) || blocked.Reason != "Отпуск" {
		t.Fatalf("вход заблокированного: %v", err)
	}
	if _, err = s.Unblock(ctx, mod.ID, "boss"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ResetPassword(ctx, mod.ID, "СброшенныйПароль1", "boss"); err != nil || store.log[len(store.log)-1] != "password_reset" {
		t.Fatalf("сброс пароля: %v", err)
	}
	if err = s.ChangePassword(ctx, mod.ID, "СброшенныйПароль1", "short1"); err == nil || !strings.Contains(err.Error(), "12") {
		t.Fatalf("слабый новый пароль: %v", err)
	}
	if err = s.ChangePassword(ctx, mod.ID, "СброшенныйПароль1", "СвойПароль2026"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Delete(ctx, mod.ID, "boss"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Login(ctx, "mod@example.ru", "СвойПароль2026"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("вход удалённого: %v", err)
	}
	if _, err = s.Unblock(ctx, mod.ID, "boss"); !errors.Is(err, ErrAdminDeleted) {
		t.Fatalf("действие над удалённым: %v", err)
	}
}

// Управление админами доступно только суперадминистратору: обычный админ получает 403.
func TestAdminsRequireSuperadmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterAdmins(r.Group("/api/v1", platform.TrustGateway("secret")), NewAdmins(newFakeAdmins(), NewHasher(1)))
	for roles, want := range map[string]int{"admin": 403, "user,admin": 403, "admin,superadmin": 200} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/admins", nil)
		req.Header.Set("X-Internal-Token", "secret")
		req.Header.Set("X-User-ID", "id-boss")
		req.Header.Set("X-User-Roles", roles)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("роли %q: %d, ждали %d", roles, w.Code, want)
		}
	}
}
