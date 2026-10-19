//go:build integration

package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
)

func TestReportsCommentsAndRemoval(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	s := New(NewRepository(db))
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
	owner, anna, igor, petr := person("Организатор"), person("Анна"), person("Игорь"), person("Пётр")
	admin := person("Модератор")
	admin.Admin = true
	one := 1
	e, err := s.CreateEvent(ctx, owner, EventInput{Title: "Пробежка у реки", SportName: "Бег", StartsAt: time.Now().Add(48 * time.Hour), Location: "Набережная", JoinMode: "open", Capacity: &one})
	must(err)

	t.Run("жалоба на сообщение и автоматическое закрытие при скрытии", func(t *testing.T) {
		spam, err := s.AddComment(ctx, igor, e.ID, nil, "Купите кроссовки по ссылке")
		must(err)
		if _, err = s.Report(ctx, igor, "comment", e.ID, spam.ID, ReportInput{Reason: "spam"}); !errors.Is(err, ErrOwnTarget) {
			t.Fatalf("жалоба на своё сообщение: %v", err)
		}
		var fe FieldError
		if _, err = s.Report(ctx, anna, "comment", e.ID, spam.ID, ReportInput{Reason: "other"}); !errors.As(err, &fe) {
			t.Fatalf("«другое» без пояснения: %v", err)
		}
		r, err := s.Report(ctx, anna, "comment", e.ID, spam.ID, ReportInput{Reason: "spam"})
		must(err)
		if _, err = s.Report(ctx, anna, "comment", e.ID, spam.ID, ReportInput{Reason: "spam"}); !errors.Is(err, ErrReportExists) {
			t.Fatalf("дубль жалобы: %v", err)
		}
		_, err = s.Report(ctx, petr, "comment", e.ID, spam.ID, ReportInput{Reason: "offensive"})
		must(err)
		page, err := s.AdminReports(ctx, "open", "comment", 20, 0)
		must(err)
		if page.Total != 2 || page.Items[0].OpenOnTarget != 2 || page.Items[0].Target.Text != spam.Body || page.Items[0].Target.Author.ID != igor.ID {
			t.Fatalf("очередь жалоб: %+v", page)
		}
		_, err = s.HideComment(ctx, admin, spam.ID, "Реклама")
		must(err)
		mine, err := s.MyReports(ctx, anna)
		must(err)
		if len(mine) != 1 || mine[0].ID != r.ID || mine[0].Status != "resolved" || mine[0].ResolvedAt == nil {
			t.Fatalf("жалоба не закрыта скрытием: %+v", mine)
		}
		// Скрытое сообщение нельзя обжаловать повторно — оно уже не видно.
		if _, err = s.Report(ctx, petr, "comment", e.ID, spam.ID, ReportInput{Reason: "spam"}); !errors.Is(err, ErrNotFound) {
			t.Fatalf("жалоба на скрытое: %v", err)
		}
		restored, err := s.RestoreComment(ctx, admin, spam.ID)
		must(err)
		if restored.Status != "visible" || restored.HiddenReason != "" {
			t.Fatalf("возврат сообщения: %+v", restored)
		}
		if _, err = s.RestoreComment(ctx, admin, spam.ID); !errors.Is(err, platform.ErrTransition) {
			t.Fatalf("повторный возврат: %v", err)
		}
	})

	t.Run("отклонение жалоб на событие с пояснением", func(t *testing.T) {
		if _, err := s.Report(ctx, owner, "event", e.ID, e.ID, ReportInput{Reason: "spam"}); !errors.Is(err, ErrOwnTarget) {
			t.Fatalf("жалоба организатора на своё событие: %v", err)
		}
		r, err := s.Report(ctx, anna, "event", e.ID, e.ID, ReportInput{Reason: "wrong_info", Details: "Там стройка"})
		must(err)
		if _, err = s.DismissReport(ctx, admin, r.ID, " "); err == nil {
			t.Fatal("отказ без пояснения принят")
		}
		got, err := s.DismissReport(ctx, admin, r.ID, "Проверили — дорожка открыта")
		must(err)
		if got.Status != "dismissed" || got.Resolution == "" {
			t.Fatalf("отказ: %+v", got)
		}
		if _, err = s.DismissReport(ctx, admin, r.ID, "ещё раз"); !errors.Is(err, platform.ErrTransition) {
			t.Fatalf("повторный отказ: %v", err)
		}
		var notified int
		must(db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE type='notification.requested' AND user_id=$1 AND payload->>'title' LIKE 'Жалоба рассмотрена%'", anna.ID).Scan(&notified))
		if notified != 2 {
			t.Fatalf("уведомлений о жалобах: %d", notified)
		}
	})

	t.Run("удаление своего сообщения", func(t *testing.T) {
		q, err := s.AddComment(ctx, anna, e.ID, nil, "Во сколько сбор?")
		must(err)
		answer, err := s.AddComment(ctx, owner, e.ID, &q.ID, "В 7:00")
		must(err)
		if _, err = s.DeleteComment(ctx, igor, e.ID, q.ID); !errors.Is(err, ErrNotAuthor) {
			t.Fatalf("удаление чужого: %v", err)
		}
		_, err = s.DeleteComment(ctx, anna, e.ID, q.ID)
		must(err)
		list, err := s.Comments(ctx, igor, e.ID)
		must(err)
		var deleted, reply bool
		for _, c := range list {
			deleted = deleted || c.ID == q.ID && c.Status == "deleted" && c.Body == ""
			reply = reply || c.ID == answer.ID && c.Body == "В 7:00"
		}
		if !deleted || !reply {
			t.Fatalf("после удаления: %+v", list)
		}
		adminView, err := s.Comments(ctx, admin, e.ID)
		must(err)
		for _, c := range adminView {
			if c.ID == q.ID && c.Body != "Во сколько сбор?" {
				t.Fatal("админ должен видеть оригинал удалённого сообщения")
			}
		}
		if _, err = s.RestoreComment(ctx, admin, q.ID); !errors.Is(err, platform.ErrTransition) {
			t.Fatalf("удалённое автором не возвращается модератором: %v", err)
		}
	})

	t.Run("исключение участника отдаёт место очереди и закрывает повторную запись", func(t *testing.T) {
		first, err := s.Join(ctx, anna, e.ID, "", false)
		must(err)
		waiting, err := s.Join(ctx, igor, e.ID, "", false)
		must(err)
		if first.Status != "confirmed" || waiting.Status != "waitlist" {
			t.Fatalf("запись: %s %s", first.Status, waiting.Status)
		}
		if _, err = s.RemoveParticipant(ctx, anna, e.ID, first.ID, "x"); !errors.Is(err, ErrNotOrganizer) {
			t.Fatalf("исключение не организатором: %v", err)
		}
		if _, err = s.RemoveParticipant(ctx, owner, e.ID, first.ID, ""); err == nil {
			t.Fatal("исключение без причины")
		}
		removed, err := s.RemoveParticipant(ctx, owner, e.ID, first.ID, "Нарушал правила")
		must(err)
		if removed.Status != "rejected" {
			t.Fatalf("статус исключённого: %s", removed.Status)
		}
		ig, err := s.GetEvent(ctx, igor, e.ID)
		must(err)
		if ig.MyParticipation == nil || ig.MyParticipation.Status != "confirmed" {
			t.Fatalf("место не перешло очереди: %+v", ig.MyParticipation)
		}
		if _, err = s.Join(ctx, anna, e.ID, "", false); !errors.Is(err, ErrRejected) {
			t.Fatalf("исключённый записался снова: %v", err)
		}
		if _, err = s.RemoveParticipant(ctx, owner, e.ID, first.ID, "Ещё раз"); !errors.Is(err, platform.ErrTransition) {
			t.Fatalf("повторное исключение: %v", err)
		}
	})

	t.Run("снятие события закрывает жалобы на него", func(t *testing.T) {
		_, err := s.Report(ctx, petr, "event", e.ID, e.ID, ReportInput{Reason: "dangerous", Details: "Гололёд"})
		must(err)
		_, err = s.ModerateEvent(ctx, admin, e.ID, "unpublished", "Небезопасное место")
		must(err)
		page, err := s.AdminReports(ctx, "open", "event", 20, 0)
		must(err)
		if page.Total != 0 {
			t.Fatalf("жалобы на снятое событие остались открытыми: %+v", page.Items)
		}
	})
}
