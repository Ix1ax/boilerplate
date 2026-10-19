// Package accounts — учётные записи обычной регистрации: почта и пароль, вход, смена пароля,
// блокировка администратором. Сессии и cookie остаются в gateway; сервис только проверяет
// данные и отвечает, кто этот пользователь и можно ли ему входить.
package accounts

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Account — учётная запись без хеша пароля: её можно отдавать gateway и админке.
type Account struct {
	ID                string     `json:"id"`
	Email             string     `json:"email"`
	Name              string     `json:"name"`
	Status            string     `json:"status"`
	BlockReason       string     `json:"block_reason"`
	Roles             []string   `json:"roles"`
	LastLoginAt       *time.Time `json:"last_login_at"`
	CreatedAt         time.Time  `json:"created_at"`
	PasswordChangedAt time.Time  `json:"password_changed_at"`
}

// HistoryEntry — регистрация, блокировки и смены пароля.
type HistoryEntry struct {
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
}

// AccountDetail — карточка для админа.
type AccountDetail struct {
	Account
	History []HistoryEntry `json:"history"`
}

type Page struct {
	Items []Account `json:"items"`
	Total int       `json:"total"`
}

// userRoles — роли обычной регистрации. Администраторы входят отдельно (admins.go, своя сессия
// в gateway): регистрация не должна давать способа получить роль admin.
var userRoles = []string{"user"}

var (
	ErrNotFound           = errors.New("Учётная запись не найдена")
	ErrEmailTaken         = errors.New("Эта почта уже зарегистрирована. Войдите или используйте другую")
	ErrInvalidCredentials = errors.New("Неверная почта или пароль")
	ErrWrongPassword      = errors.New("Текущий пароль указан неверно")
	ErrSamePassword       = errors.New("Новый пароль совпадает с текущим")
	ErrTransition         = errors.New("Учётная запись уже в этом состоянии")
	ErrReasonRequired     = errors.New("Укажите причину блокировки: её увидит пользователь")
)

// BlockedError — вход запрещён администратором; причину показываем пользователю.
type BlockedError struct{ Reason string }

func (e BlockedError) Error() string {
	return "Учётная запись заблокирована: " + e.Reason
}

// LockedError — временная блокировка после серии неверных паролей.
type LockedError struct{ Until time.Time }

func (e LockedError) Error() string {
	minutes := int(time.Until(e.Until).Round(time.Minute).Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf("Слишком много неудачных попыток. Попробуйте через %d мин.", minutes)
}

// FieldError — ошибка конкретного поля формы.
type FieldError struct{ Field, Message string }

func (e FieldError) Error() string { return e.Message }

// NormalizeEmail приводит почту к виду, в котором она хранится, и проверяет формат.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	addr, err := mail.ParseAddress(email)
	if email == "" || len(email) > 254 || err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@")+1:], ".") {
		return "", FieldError{"email", "Укажите почту в формате name@example.ru"}
	}
	return email, nil
}

// NormalizeName — имя видят другие участники в обсуждениях и списках.
func NormalizeName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if n := utf8.RuneCountInString(name); n < 2 || n > 100 {
		return "", FieldError{"name", "Имя — от 2 до 100 символов"}
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", FieldError{"name", "Имя содержит недопустимые символы"}
		}
	}
	return name, nil
}

// common — самые частые пароли из утечек: длина их не спасает.
var common = map[string]bool{
	"12345678": true, "123456789": true, "1234567890": true, "password": true, "password1": true,
	"qwerty123": true, "qwertyuiop": true, "11111111": true, "00000000": true, "iloveyou": true,
	"йцукенгш": true, "пароль123": true, "qwerty12": true, "1q2w3e4r": true, "1q2w3e4r5t": true,
	"zaq12wsx": true, "abc12345": true, "admin123": true, "stoloto1": true, "stoloto123": true,
}

// ValidatePassword: 8–128 символов, буквы и цифры, не из списка частых и не равен почте.
// Обязательных спецсимволов нет: они ухудшают запоминаемость, не добавляя стойкости.
func ValidatePassword(password, email string) error {
	n := utf8.RuneCountInString(password)
	switch {
	case n < 8:
		return FieldError{"password", "Пароль — не короче 8 символов"}
	case n > 128:
		return FieldError{"password", "Пароль — не длиннее 128 символов"}
	case common[strings.ToLower(password)]:
		return FieldError{"password", "Этот пароль слишком распространён — придумайте другой"}
	case email != "" && strings.EqualFold(password, email), email != "" && strings.EqualFold(password, strings.Split(email, "@")[0]):
		return FieldError{"password", "Пароль не должен совпадать с почтой"}
	}
	var letter, digit bool
	for _, r := range password {
		letter = letter || unicode.IsLetter(r)
		digit = digit || unicode.IsDigit(r)
	}
	if !letter || !digit {
		return FieldError{"password", "Пароль должен содержать буквы и цифры"}
	}
	return nil
}

func cleanReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	switch {
	case reason == "":
		return "", ErrReasonRequired
	case utf8.RuneCountInString(reason) > 500:
		return "", FieldError{"reason", "Причина — не длиннее 500 символов"}
	}
	return reason, nil
}
