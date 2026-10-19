//go:build integration

package events

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestReviewRegressions(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	repo := NewRepository(db)
	s := New(repo)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	person := func() Actor {
		a := Actor{ID: uuid.NewString(), Name: "Участник"}
		must(s.EnsureProfile(ctx, a))
		return a
	}
	owner, admin := person(), person()
	admin.Admin = true
	create := func(capacity, minimum *int) Event {
		e, err := s.CreateEvent(ctx, owner, EventInput{Title: "Встреча", SportName: "Бег", StartsAt: time.Now().Add(48 * time.Hour), Location: "Стадион", JoinMode: "open", Capacity: capacity, MinParticipants: minimum})
		must(err)
		return e
	}
	t.Run("снимок после изменения организации", func(t *testing.T) {
		input := OrganizationInput{Name: "Клуб", Description: "Спортивный клуб для всех"}
		org, err := s.SubmitOrganization(ctx, owner, input)
		must(err)
		_, err = s.ReviewOrganization(ctx, admin, org.ID, "rejected", "Уточните описание")
		must(err)
		input.Name = "Новый клуб"
		_, err = s.UpdateOrganization(ctx, owner, input)
		must(err)
		var status, name string
		must(db.QueryRow(ctx, `SELECT payload->'snapshot'->>'status', payload->'snapshot'->>'name' FROM outbox WHERE type='domain.organization.changed' AND payload->>'id'=$1 AND payload->>'from'='rejected'`, org.ID).Scan(&status, &name))
		if status != "pending" || name != input.Name {
			t.Fatalf("устаревший снимок: %s %s", status, name)
		}
		input.Name = "Исправленный клуб"
		_, err = s.UpdateOrganization(ctx, owner, input)
		must(err)
		var count int
		must(db.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE type='domain.organization.changed' AND payload->>'id'=$1 AND payload->'snapshot'->>'name'=$2`, org.ID, input.Name).Scan(&count))
		if count != 1 {
			t.Fatalf("снимков правки без смены статуса: %d", count)
		}
	})
	t.Run("свободные места получает очередь", func(t *testing.T) {
		one, three, two := 1, 3, 2
		e := create(&one, nil)
		first, second, third, newcomer := person(), person(), person(), person()
		for _, a := range []Actor{first, second, third} {
			_, err := s.Join(ctx, a, e.ID, "", false)
			must(err)
		}
		// Порядок очереди фиксирован независимо от точности часов базы.
		_, err := db.Exec(ctx, "UPDATE participations SET created_at=now()-interval '1 hour' WHERE event_id=$1 AND user_id=$2", e.ID, second.ID)
		must(err)
		_, err = s.UpdateEvent(ctx, owner, e.ID, EventPatch{Capacity: &two})
		must(err)
		secondEvent, err := s.GetEvent(ctx, second, e.ID)
		must(err)
		thirdEvent, err := s.GetEvent(ctx, third, e.ID)
		must(err)
		if secondEvent.MyParticipation.Status != "confirmed" || thirdEvent.MyParticipation.Status != "waitlist" {
			t.Fatal("очередь не соблюдена")
		}
		_, err = s.UpdateEvent(ctx, owner, e.ID, EventPatch{Capacity: &three})
		must(err)
		joined, err := s.Join(ctx, newcomer, e.ID, "", false)
		must(err)
		if joined.Status != "waitlist" {
			t.Fatal("новый участник обошёл очередь")
		}
		var notifications int
		must(db.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE type='notification.requested' AND user_id IN ($1,$2) AND payload->>'title'='Освободилось место — вы участвуете'`, second.ID, third.ID).Scan(&notifications))
		if notifications != 2 {
			t.Fatalf("уведомлений очереди: %d", notifications)
		}
	})
	t.Run("перезапись из очереди не обгоняет стоящих", func(t *testing.T) {
		one := 1
		e := create(&one, nil)
		holder, a, b := person(), person(), person()
		for _, p := range []Actor{holder, a, b} {
			_, err := s.Join(ctx, p, e.ID, "", false)
			must(err)
		}
		// A встал в очередь раньше B; без сброса created_at повторная запись сохранила бы его место.
		_, err := db.Exec(ctx, "UPDATE participations SET created_at=now()-interval '2 hours' WHERE event_id=$1 AND user_id=$2", e.ID, a.ID)
		must(err)
		_, err = db.Exec(ctx, "UPDATE participations SET created_at=now()-interval '1 hour' WHERE event_id=$1 AND user_id=$2", e.ID, b.ID)
		must(err)
		must(s.Leave(ctx, a, e.ID))
		again, err := s.Join(ctx, a, e.ID, "", false)
		must(err)
		if again.Status != "waitlist" {
			t.Fatalf("повторная запись: %s", again.Status)
		}
		must(s.Leave(ctx, holder, e.ID))
		for p, want := range map[Actor]string{b: "confirmed", a: "waitlist"} {
			got, err := s.GetEvent(ctx, p, e.ID)
			must(err)
			if got.MyParticipation.Status != want {
				t.Fatalf("после освобождения места: ждали %s, получили %s", want, got.MyParticipation.Status)
			}
		}
	})
	t.Run("вернувшийся встаёт в конец очереди", func(t *testing.T) {
		one := 1
		e := create(&one, nil)
		holder, returner, waiting := person(), person(), person()
		for _, a := range []Actor{holder, returner} {
			_, err := s.Join(ctx, a, e.ID, "", false)
			must(err)
		}
		// Первая запись вернувшегося — самая давняя: без сброса created_at он обошёл бы очередь.
		_, err := db.Exec(ctx, "UPDATE participations SET created_at=now()-interval '2 hours' WHERE event_id=$1 AND user_id=$2", e.ID, returner.ID)
		must(err)
		must(s.Leave(ctx, returner, e.ID))
		_, err = s.Join(ctx, waiting, e.ID, "", false)
		must(err)
		_, err = db.Exec(ctx, "UPDATE participations SET created_at=now()-interval '1 hour' WHERE event_id=$1 AND user_id=$2", e.ID, waiting.ID)
		must(err)
		again, err := s.Join(ctx, returner, e.ID, "", false)
		must(err)
		if again.Status != "waitlist" {
			t.Fatalf("повторная запись: %s", again.Status)
		}
		must(s.Leave(ctx, holder, e.ID))
		waitingEvent, err := s.GetEvent(ctx, waiting, e.ID)
		must(err)
		returnerEvent, err := s.GetEvent(ctx, returner, e.ID)
		must(err)
		if waitingEvent.MyParticipation.Status != "confirmed" || returnerEvent.MyParticipation.Status != "waitlist" {
			t.Fatalf("место ушло вернувшемуся: ждал %s, вернулся %s", waitingEvent.MyParticipation.Status, returnerEvent.MyParticipation.Status)
		}
	})
	t.Run("правка после сбора состава", func(t *testing.T) {
		two := 2
		e := create(nil, &two)
		for range 2 {
			_, err := s.Join(ctx, person(), e.ID, "", false)
			must(err)
		}
		_, err := db.Exec(ctx, "UPDATE events SET quorum_deadline=now()-interval '1 hour' WHERE id=$1", e.ID)
		must(err)
		place := "Другой стадион"
		updated, err := s.UpdateEvent(ctx, owner, e.ID, EventPatch{Location: &place})
		must(err)
		if updated.Location != place || updated.QuorumStatus != "reached" {
			t.Fatalf("правка потеряна: %+v", updated)
		}
	})
	t.Run("автоотмена перепроверяет перенесённый срок", func(t *testing.T) {
		two := 2
		e := create(nil, &two)
		_, err := db.Exec(ctx, "UPDATE events SET quorum_deadline=now()-interval '1 hour' WHERE id=$1", e.ID)
		must(err)
		var selected string
		must(db.QueryRow(ctx, "SELECT id FROM events WHERE id=$1 AND quorum_deadline<=now()", e.ID).Scan(&selected))
		later := e.StartsAt.Add(24 * time.Hour)
		_, err = s.UpdateEvent(ctx, owner, e.ID, EventPatch{StartsAt: &later})
		must(err)
		must(repo.inTx(ctx, func(tx pgx.Tx) error { return failQuorum(ctx, tx, selected) }))
		current, err := s.GetEvent(ctx, owner, e.ID)
		must(err)
		if current.Status != "published" || current.QuorumStatus != "gathering" {
			t.Fatalf("перенесённое событие отменено: %+v", current)
		}
	})
}
