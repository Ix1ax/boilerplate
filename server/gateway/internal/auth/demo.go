package auth

import (
	"context"
	"errors"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// DemoAccount — демо-аккаунт для быстрого входа экспертов. Постановка разрешает имитировать вход,
// а роли при проверке переключают десятки раз, поэтому форма Keycloak здесь лишний шаг.
// Пароль остаётся на сервере: клиент получает только ключ, подпись и подсказку.
type DemoAccount struct {
	Key   string `json:"key"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	Hint  string `json:"hint"`
	email string
	pass  string
}

// demoAccounts — порядок карточек на экране входа. Почты и пароли создаёт make setup.
var demoAccounts = []struct {
	key, emailEnv, passEnv, name, role, hint string
}{
	{"anna", "DEMO_EMAIL", "DEMO_PASSWORD", "Анна Смирнова", "Пользователь", "Найти событие, записаться, создать своё, задать вопрос"},
	{"igor", "SECOND_DEMO_EMAIL", "SECOND_DEMO_PASSWORD", "Игорь Волков", "Пользователь", "Второй участник: заявка, ответ в обсуждении"},
	{"oleg", "ORG_DEMO_EMAIL", "ORG_DEMO_PASSWORD", "Олег Ладьин", "Организация «Ладья», допущена", "Публикация от имени клуба, разбор заявок"},
	{"maria", "ORG_PENDING_DEMO_EMAIL", "ORG_PENDING_DEMO_PASSWORD", "Мария Соснова", "Организация «Сосны», на проверке", "Заявка ждёт решения, публикация недоступна"},
	{"petr", "ORG_REJECTED_DEMO_EMAIL", "ORG_REJECTED_DEMO_PASSWORD", "Пётр Пульсов", "Организация «Пульс», отклонена", "Причина отказа и повторная подача"},
	{"admin", "ADMIN_DEMO_EMAIL", "ADMIN_DEMO_PASSWORD", "Модератор Платформы", "Администратор", "Допуск организаций, снятие событий, скрытие сообщений"},
}

// DemoAccountsFromEnv собирает аккаунты, для которых в окружении есть почта и пароль.
func DemoAccountsFromEnv() []DemoAccount {
	var out []DemoAccount
	for _, a := range demoAccounts {
		email, pass := os.Getenv(a.emailEnv), os.Getenv(a.passEnv)
		if email == "" || pass == "" {
			continue
		}
		out = append(out, DemoAccount{Key: a.key, Name: a.name, Role: a.role, Hint: a.hint, email: email, pass: pass})
	}
	return out
}

// RegisterDemo добавляет GET /auth/demo (список аккаунтов) и POST /auth/demo/{key} (вход).
// Вход идёт через Keycloak по паролю (password grant клиента приложения), поэтому дальше
// это обычная сессия: те же токены, роли, обновление и выход, что при входе через форму.
func (s *Sessions) RegisterDemo(r *gin.Engine, accounts []DemoAccount) {
	r.GET("/auth/demo", func(c *gin.Context) {
		c.JSON(200, gin.H{"items": accounts})
	})
	r.POST("/auth/demo/:key", func(c *gin.Context) {
		// Маршрут вне /api, поэтому Origin проверяем сами: иначе чужой сайт мог бы подменить сессию.
		if c.GetHeader("Origin") != s.origin {
			sessionError(c, 403, "invalid_origin", "Запрос с этого адреса запрещён")
			return
		}
		account, found := findDemo(accounts, c.Param("key"))
		if !found {
			sessionError(c, 404, "not_found", "Такого демо-аккаунта нет")
			return
		}
		ctx := context.WithValue(c.Request.Context(), oauth2.HTTPClient, s.client)
		token, err := s.oauth.PasswordCredentialsToken(ctx, account.email, account.pass)
		if err != nil {
			var response *oauth2.RetrieveError
			if errors.As(err, &response) && response.Response.StatusCode < 500 {
				sessionError(c, 409, "demo_unavailable", "Демо-вход недоступен: проверьте, что make setup применён к Keycloak")
				return
			}
			sessionError(c, 502, "login_failed", "Не удалось завершить вход")
			return
		}
		if !s.start(c, ctx, token) {
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": account.Key, "name": account.Name})
	})
}

func findDemo(accounts []DemoAccount, key string) (DemoAccount, bool) {
	for _, a := range accounts {
		if a.Key == key {
			return a, true
		}
	}
	return DemoAccount{}, false
}
