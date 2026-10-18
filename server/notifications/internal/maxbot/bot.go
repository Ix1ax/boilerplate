// Package maxbot — канал уведомлений в мессенджер MAX через официальный клиент
// github.com/max-messenger/max-bot-api-client-go (Apache 2.0).
//
// Привязка: приложение выдаёт ссылку https://max.ru/<бот>/start/<код>; MAX открывает диалог
// с ботом и присылает код в событии bot_started. Команда /stop отвязывает аккаунт.
// Обновления читаются long polling'ом, поэтому публичный адрес сервису не нужен.
package maxbot

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"example.com/hackathon/apps/notifications/internal/notifications"
	maxbot "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

// Links — хранилище привязок (notifications.Repository).
type Links interface {
	ConsumeLinkCode(ctx context.Context, code string) (string, error)
	Link(ctx context.Context, user string, maxUserID int64) error
	UnlinkMax(ctx context.Context, maxUserID int64) (bool, error)
}

type Bot struct {
	api      *maxbot.Api
	links    Links
	log      *slog.Logger
	username string
}

// New: baseURL пустой — официальный сервер MAX; в тестах — адрес фейкового сервера.
func New(token, username, baseURL string, links Links, log *slog.Logger) (*Bot, error) {
	var opts []maxbot.Opt
	if baseURL != "" {
		opts = append(opts, maxbot.WithBaseURL(baseURL))
	}
	api, err := maxbot.NewApi(token, opts...)
	if err != nil {
		return nil, err
	}
	return &Bot{api: api, links: links, log: log, username: username}, nil
}

// LinkURL — deep link для привязки (код до 128 символов, ограничение MAX).
func (b *Bot) LinkURL(code string) string {
	return "https://max.ru/" + b.username + "/start/" + code
}

// Send отправляет текст пользователю MAX.
func (b *Bot) Send(ctx context.Context, maxUserID int64, text string) error {
	_, err := b.api.Messages.Send(ctx, maxbot.NewMessage().SetUser(maxUserID).SetText(text))
	return err
}

// Run читает обновления, пока ctx не отменён; сетевые сбои переживает с паузой.
func (b *Bot) Run(ctx context.Context) {
	var marker int64
	for ctx.Err() == nil {
		updates, next, err := b.api.Subscriptions.GetUpdates(ctx, marker)
		if err != nil {
			if ctx.Err() == nil {
				b.log.Warn("MAX: не удалось получить обновления", "error", err)
				select {
				case <-ctx.Done():
				case <-time.After(5 * time.Second):
				}
			}
			continue
		}
		marker = next
		for _, u := range updates {
			b.Handle(ctx, u)
		}
	}
}

// Handle обрабатывает одно обновление (вынесено для тестов).
func (b *Bot) Handle(ctx context.Context, u model.Update) {
	userID := senderID(u)
	if userID == 0 {
		return
	}
	switch u.UpdateType {
	case model.UpdateBotStarted:
		b.reply(ctx, userID, b.link(ctx, userID, u.Payload))
	case model.UpdateMessageCreated:
		cmd := u.GetCommand()
		switch strings.ToLower(cmd.Command) {
		case "/start":
			b.reply(ctx, userID, b.link(ctx, userID, strings.TrimSpace(strings.Join(cmd.Params, " "))))
		case "/stop":
			if ok, err := b.links.UnlinkMax(ctx, userID); err != nil {
				b.log.Warn("MAX: отвязка не удалась", "error", err)
				b.reply(ctx, userID, "Не получилось отключить уведомления, попробуйте позже.")
			} else if ok {
				b.reply(ctx, userID, "Уведомления отключены. Подключить снова можно в профиле приложения.")
			} else {
				b.reply(ctx, userID, "Аккаунт не был подключён.")
			}
		default:
			b.reply(ctx, userID, "Я присылаю уведомления из приложения. Подключение — кнопкой «Подключить MAX» в приложении, отключение — /stop.")
		}
	}
}

func (b *Bot) link(ctx context.Context, maxUserID int64, code string) string {
	if code == "" {
		return "Чтобы получать уведомления, нажмите «Подключить MAX» в приложении."
	}
	user, err := b.links.ConsumeLinkCode(ctx, code)
	if err == nil {
		err = b.links.Link(ctx, user, maxUserID)
	}
	switch {
	case err == nil:
		return "Готово! Уведомления из приложения будут приходить сюда. Отключить — /stop."
	case errors.Is(err, notifications.ErrLinkCode):
		return "Ссылка недействительна или устарела. Получите новую в приложении."
	default:
		b.log.Warn("MAX: привязка не удалась", "error", err)
		return "Не получилось подключить уведомления, попробуйте позже."
	}
}

func (b *Bot) reply(ctx context.Context, maxUserID int64, text string) {
	if err := b.Send(ctx, maxUserID, text); err != nil {
		b.log.Warn("MAX: ответ не отправлен", "error", err)
	}
}

func senderID(u model.Update) int64 {
	if u.UserID != 0 {
		return u.UserID
	}
	if id := u.GetUser().UserID; id != 0 {
		return id
	}
	return u.GetMessage().Sender.UserID
}
