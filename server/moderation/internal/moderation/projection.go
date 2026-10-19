// Package moderation — админка-аналитика платформы: дашборд, журнал модерации, лента сообщений
// и отчёты. Сервис ничего не решает в домене: решения принимает events, а сюда приходят его
// доменные события domain.*.changed со снимком сущности, из которых строятся таблицы для чтения.
// Так тяжёлые выборки и выгрузки админа не нагружают базу, где идёт запись на события.
package moderation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Envelope — data доменного события (см. events/internal/events/domain.go).
type Envelope struct {
	Entity   string          `json:"entity"`
	ID       string          `json:"id"`
	Kind     string          `json:"kind"`
	From     string          `json:"from"`
	To       string          `json:"to"`
	Actor    string          `json:"actor"`
	Comment  string          `json:"comment"`
	At       time.Time       `json:"at"`
	Snapshot json.RawMessage `json:"snapshot"`
}

// projections — таблица проекции для каждой сущности и колонки, которые обновляет новый снимок.
// Имена колонок совпадают с ключами снимка: jsonb_populate_record раскладывает его сам.
var projections = map[string]struct {
	table   string
	columns []string
}{
	"organization":    {"organizations", []string{"name", "city", "owner_id", "owner_name", "status", "review_reason", "created_at"}},
	"event":           {"events", []string{"title", "sport", "organizer_id", "organizer_name", "organization_id", "organization_name", "starts_at", "location", "join_mode", "capacity", "min_participants", "quorum_status", "outdoor", "status", "moderation_reason", "created_at"}},
	"participation":   {"participations", []string{"event_id", "event_title", "user_id", "user_name", "status", "first_time", "created_at"}},
	"comment":         {"comments", []string{"event_id", "event_title", "author_id", "author_name", "body", "status", "hidden_reason", "created_at"}},
	"company_request": {"company_requests", []string{"user_id", "sport", "level", "status", "created_at"}},
}

// upsertSQL — вставка снимка или обновление, только если он не старше сохранённого.
func upsertSQL(table string, columns []string) string {
	excluded := make([]string, len(columns))
	for i, c := range columns {
		excluded[i] = "EXCLUDED." + c
	}
	return fmt.Sprintf(`INSERT INTO %[1]s SELECT (jsonb_populate_record(NULL::%[1]s, $1::jsonb || jsonb_build_object('changed_at', $2::timestamptz))).*
ON CONFLICT (id) DO UPDATE SET (%[2]s, changed_at) = ROW(%[3]s, EXCLUDED.changed_at)
WHERE %[1]s.changed_at <= EXCLUDED.changed_at`, table, strings.Join(columns, ", "), strings.Join(excluded, ", "))
}

// Logged — попадает ли переход в журнал модерации: решения по организациям, смена статуса событий
// (снятие, удаление, возврат, отмена, в том числе автоматическая) и скрытие сообщений.
// Создание сущностей и переотправка снимков — не решения.
func Logged(e Envelope) bool {
	if e.Actor == "snapshot" || e.From == "none" || e.From == "" || e.From == e.To {
		return false
	}
	switch e.Entity {
	case "organization":
		return true
	case "event":
		return e.Kind == "event"
	case "comment":
		return e.To == "hidden"
	}
	return false
}

type Repository struct{ db *pgxpool.Pool }

func NewRepository(db *pgxpool.Pool) *Repository { return &Repository{db: db} }

// Apply сохраняет снимок и запись журнала в одной транзакции. Повтор того же события безопасен:
// проекция не откатится (changed_at), журнал не задвоится (message_id).
func (r *Repository) Apply(ctx context.Context, messageID string, e Envelope) error {
	p, ok := projections[e.Entity]
	if !ok {
		return nil // новая сущность в events, о которой moderation ещё не знает
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, upsertSQL(p.table, p.columns), string(e.Snapshot), e.At); err != nil {
		return fmt.Errorf("проекция %s: %w", p.table, err)
	}
	if Logged(e) {
		if err = insertLog(ctx, tx, messageID, e); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func insertLog(ctx context.Context, tx pgx.Tx, messageID string, e Envelope) error {
	var snap struct {
		Title      string `json:"title"`
		Name       string `json:"name"`
		EventTitle string `json:"event_title"`
		Body       string `json:"body"`
	}
	_ = json.Unmarshal(e.Snapshot, &snap)
	title := snap.Title
	switch e.Entity {
	case "organization":
		title = snap.Name
	case "comment":
		title = snap.EventTitle + ": «" + truncate(snap.Body, 80) + "»"
	}
	_, err := tx.Exec(ctx, `INSERT INTO moderation_log(message_id,entity,entity_id,kind,title,from_status,to_status,actor,comment,at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (message_id) DO NOTHING`,
		messageID, e.Entity, e.ID, e.Kind, title, e.From, e.To, e.Actor, e.Comment, e.At)
	return err
}

func truncate(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}

// Store — то, что нужно обработчику событий (Repository).
type Store interface {
	Apply(ctx context.Context, messageID string, e Envelope) error
}

type Service struct{ store Store }

func New(store Store) *Service { return &Service{store: store} }

// HandleEvent принимает domain.*.changed. Нечитаемое событие отбрасываем, чтобы не зацикливать
// очередь; ошибку базы возвращаем — сообщение вернётся и будет доставлено повторно.
func (s *Service) HandleEvent(ctx context.Context, messageID, eventType string, data []byte) error {
	if !strings.HasPrefix(eventType, "domain.") {
		return nil
	}
	var e Envelope
	if json.Unmarshal(data, &e) != nil || uuid.Validate(e.ID) != nil || uuid.Validate(messageID) != nil || e.At.IsZero() {
		return nil
	}
	if len(e.Snapshot) == 0 || string(e.Snapshot) == "null" {
		return nil // сущность удалена раньше, чем событие дошло
	}
	return s.store.Apply(ctx, messageID, e)
}
