//go:build integration

package events

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
)

// Жалобы на отзывы, истории и личные сообщения: кто может пожаловаться, что видит админ,
// и что его решение закрывает жалобы, пишет историю и уведомляет автора цели.
func TestReportsOnSocialContent(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	r := NewRepository(db)
	s := New(r)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	person := func(name string) Actor {
		a := Actor{ID: uuid.NewString(), Name: name}
		must(s.EnsureProfile(ctx, a))
		return a
	}
	owner, author, friend, outsider := person("Организатор"), person("Автор"), person("Друг"), person("Посторонний")
	admin := Actor{ID: uuid.NewString(), Name: "Модератор", Admin: true}
	_, err := r.Friendship(ctx, author, friend.ID, "request")
	must(err)
	_, err = r.Friendship(ctx, friend, author.ID, "accept")
	must(err)
	notified := func(userID, title string) bool {
		t.Helper()
		var n int
		must(db.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE type='notification.requested' AND user_id=$1 AND payload->>'title'=$2`, userID, title).Scan(&n))
		return n == 1
	}
	history := func(entity, id string) []platform.Transition {
		t.Helper()
		items, err := platform.History(ctx, db, entity, id)
		must(err)
		return items
	}
	report := func(a Actor, targetType, parent, id string) (MyReport, error) {
		return s.Report(ctx, a, targetType, parent, id, ReportInput{Reason: "offensive"})
	}
	queued := func(id string) Report {
		t.Helper()
		x, err := r.AdminReport(ctx, id)
		must(err)
		return x
	}

	t.Run("отзыв", func(t *testing.T) {
		e, err := s.CreateEvent(ctx, owner, EventInput{Title: "Пробежка", SportName: "Бег", StartsAt: time.Now().Add(24 * time.Hour), Location: "Парк у реки", JoinMode: "open"})
		must(err)
		_, err = s.Join(ctx, author, e.ID, "", false)
		must(err)
		_, err = db.Exec(ctx, `UPDATE events SET starts_at=now()-interval '4 hours' WHERE id=$1`, e.ID)
		must(err)
		out, err := r.CreateReview(ctx, author, e.ID, 1, "Организатор — идиот")
		must(err)
		raw, _ := json.Marshal(out)
		var review struct{ ID string }
		must(json.Unmarshal(raw, &review))
		if _, err = report(author, "review", "", review.ID); !errors.Is(err, ErrOwnTarget) {
			t.Fatalf("жалоба на свой отзыв: %v", err)
		}
		mine, err := report(owner, "review", "", review.ID)
		must(err)
		if mine.EventID == nil || *mine.EventID != e.ID || mine.EventTitle != "Пробежка" {
			t.Fatalf("жалоба на отзыв привязана к мероприятию: %+v", mine)
		}
		page, err := s.AdminReports(ctx, "open", "review", 20, 0)
		must(err)
		if page.Total != 1 || page.Items[0].Target.Title != "Отзыв о «Пробежка»" || page.Items[0].Target.Author.ID != author.ID || page.Items[0].Target.Status != "visible" {
			t.Fatalf("очередь жалоб на отзывы: %+v", page)
		}
		must(r.HideReview(ctx, admin, review.ID, "Оскорбление организатора"))
		if x := queued(mine.ID); x.Status != "resolved" || x.Target.Status != "hidden" || x.Target.Text == "" {
			t.Fatalf("после скрытия: %+v", x)
		}
		if !notified(author.ID, "Ваш отзыв скрыт модератором") || !notified(owner.ID, "Жалоба рассмотрена: меры приняты") {
			t.Fatal("нет уведомлений автору отзыва и автору жалобы")
		}
		if h := history("review", review.ID); len(h) != 1 || h[0].To != "hidden" {
			t.Fatalf("история отзыва: %+v", h)
		}
		if _, err = report(outsider, "review", "", review.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("жалоба на скрытый отзыв: %v", err)
		}
	})

	t.Run("история", func(t *testing.T) {
		out, err := r.CreateStory(ctx, author, "Продам абонемент со скидкой")
		must(err)
		raw, _ := json.Marshal(out)
		var story struct{ ID string }
		must(json.Unmarshal(raw, &story))
		if _, err = report(outsider, "story", "", story.ID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("посторонний не видит историю, значит и не жалуется: %v", err)
		}
		mine, err := report(friend, "story", "", story.ID)
		must(err)
		if mine.EventID != nil || mine.EventTitle != "" {
			t.Fatalf("у истории нет мероприятия: %+v", mine)
		}
		// Жалоба переживает истечение истории: админ видит её и может удалить.
		_, err = db.Exec(ctx, `UPDATE stories SET expires_at=now()-interval '1 minute' WHERE id=$1`, story.ID)
		must(err)
		if x := queued(mine.ID); x.Target.Title != "История" || x.Target.Status != "expired" || x.EventID != nil {
			t.Fatalf("история в очереди: %+v", x)
		}
		must(r.DeleteStory(ctx, admin, story.ID, "Реклама"))
		if x := queued(mine.ID); x.Status != "resolved" || x.Target.Status != "deleted" {
			t.Fatalf("после удаления: %+v", x)
		}
		if !notified(author.ID, "Вашу историю удалил модератор") {
			t.Fatal("автор истории не уведомлён")
		}
	})

	t.Run("личное сообщение", func(t *testing.T) {
		chat, err := r.OpenDirectChat(ctx, author, friend.ID)
		must(err)
		chatID := socialObject(t, chat)["id"].(string)
		sent, err := r.SendDirectMessage(ctx, author, chatID, "Ты тупой")
		must(err)
		msgID := socialObject(t, sent)["id"].(string)
		other, err := r.SendDirectMessage(ctx, author, chatID, "Без жалобы")
		must(err)
		if _, err = report(outsider, "direct_message", chatID, msgID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("жалоба не участника переписки: %v", err)
		}
		if _, err = report(author, "direct_message", chatID, msgID); !errors.Is(err, ErrOwnTarget) {
			t.Fatalf("жалоба на своё сообщение: %v", err)
		}
		mine, err := report(friend, "direct_message", chatID, msgID)
		must(err)
		if x := queued(mine.ID); x.Target.Title != "Личное сообщение" || x.Target.Text != "Ты тупой" || x.Target.Author.ID != author.ID {
			t.Fatalf("админ видит только это сообщение: %+v", x)
		}
		if err = r.HideDirectMessage(ctx, admin, socialObject(t, other)["id"].(string), "Проверка"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("сообщение без жалобы админу недоступно: %v", err)
		}
		must(r.HideDirectMessage(ctx, admin, msgID, "Оскорбление"))
		if err = r.HideDirectMessage(ctx, admin, msgID, "Оскорбление"); !errors.Is(err, platform.ErrTransition) {
			t.Fatalf("повторное скрытие: %v", err)
		}
		page, err := r.DirectMessages(ctx, friend, chatID, SocialPage{Before: time.Now().Add(time.Hour), ID: "ffffffff-ffff-ffff-ffff-ffffffffffff", Limit: 10})
		must(err)
		hidden := false
		for _, item := range page.Items {
			m := map[string]any{}
			must(json.Unmarshal(item, &m))
			if m["id"] == msgID {
				hidden = m["body"] == "Сообщение скрыто модератором" && m["hidden"] == true
			}
		}
		if !hidden || queued(mine.ID).Status != "resolved" || !notified(author.ID, "Ваше личное сообщение скрыто модератором") {
			t.Fatal("скрытое сообщение должно стать заглушкой, жалоба — закрыться, автор — получить уведомление")
		}
	})

	t.Run("отказ по жалобе без мероприятия", func(t *testing.T) {
		out, err := r.CreateStory(ctx, author, "Обычная история")
		must(err)
		raw, _ := json.Marshal(out)
		var story struct{ ID string }
		must(json.Unmarshal(raw, &story))
		mine, err := report(friend, "story", "", story.ID)
		must(err)
		x, err := s.DismissReport(ctx, admin, mine.ID, "Нарушений нет")
		must(err)
		if x.Status != "dismissed" || !notified(friend.ID, "Жалоба рассмотрена: нарушений не нашли") {
			t.Fatalf("отказ: %+v", x)
		}
		// У истории нет мероприятия: снимок для moderation без event_id, цель названа словами.
		var eventID *string
		var title string
		must(db.QueryRow(ctx, `SELECT payload->'snapshot'->>'event_id', payload->'snapshot'->>'target_title' FROM outbox
WHERE type='domain.report.changed' AND payload->>'id'=$1 AND payload->>'to'='dismissed'`, mine.ID).Scan(&eventID, &title))
		if eventID != nil || title != "История" {
			t.Fatalf("снимок жалобы на историю: %v %q", eventID, title)
		}
		list, err := s.MyReports(ctx, friend)
		must(err)
		if len(list) != 3 {
			t.Fatalf("мои жалобы: %+v", list)
		}
	})
}
