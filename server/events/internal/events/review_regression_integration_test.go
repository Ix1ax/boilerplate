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
