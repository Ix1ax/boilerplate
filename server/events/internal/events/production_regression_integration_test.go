//go:build integration

package events

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/server/events/internal/platform"
)

func TestProductionEventRegressions(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	repo := NewRepository(db)
	s := New(repo)
	owner := Actor{ID: "reg-owner", Name: "Олег"}
	participant := Actor{ID: "reg-user", Name: "Анна"}
	for _, a := range []Actor{owner, participant} {
		if err := repo.EnsureProfile(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	create := func(mode string) Event {
		t.Helper()
		e, err := s.CreateEvent(ctx, owner, EventInput{Title: "Встреча", SportName: "Бег", Location: "Первый стадион", StartsAt: time.Now().Add(48 * time.Hour), JoinMode: mode})
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	t.Run("устаревший PATCH не затирает новую правку", func(t *testing.T) {
		e := create("open")
		snapshot := EventInput{Format: e.Format, Title: e.Title, Description: e.Description, SportID: e.Sport.ID, StartsAt: e.StartsAt, EndsAt: e.EndsAt, Location: e.Location, Lat: e.Lat, Lng: e.Lng, Level: e.Level, Requirements: e.Requirements, PriceNote: e.PriceNote, PrivateInfo: e.PrivateInfo, JoinMode: e.JoinMode, Capacity: e.Capacity, MinParticipants: e.MinParticipants, QuorumDeadline: e.QuorumDeadline, Outdoor: e.Outdoor}
		location := "Новый стадион"
		if _, err := s.UpdateEvent(ctx, owner, e.ID, EventPatch{Location: &location}); err != nil {
			t.Fatal(err)
		}
		stale := snapshot
		stale.Title = "Изменённое название"
		if _, err := repo.UpdateEvent(ctx, owner, e.ID, stale, EventChanges{Expected: &snapshot}); !errors.Is(err, platform.ErrTransition) {
			t.Fatalf("ожидали конфликт: %v", err)
		}
		current, err := repo.GetEvent(ctx, owner, e.ID)
		if err != nil || current.Location != location {
			t.Fatalf("правка потеряна: %+v %v", current, err)
		}
	})
	t.Run("напоминание использует актуальное место", func(t *testing.T) {
		e := create("open")
		if _, err := s.Join(ctx, participant, e.ID, "", false); err != nil {
			t.Fatal(err)
		}
		target, err := repo.ReminderTargetFor(ctx, owner, e.ID)
		if err != nil {
			t.Fatal(err)
		}
		location := "Точный новый адрес"
		if _, err = s.UpdateEvent(ctx, owner, e.ID, EventPatch{Location: &location}); err != nil {
			t.Fatal(err)
		}
		n, err := repo.SendReminder(ctx, target, "manual", "")
		if err != nil || n != 1 {
			t.Fatalf("напоминание: %d %v", n, err)
		}
		var body string
		if err = db.QueryRow(ctx, `SELECT payload->>'body' FROM outbox WHERE type='notification.requested' AND payload->>'title'='Напоминание: «Встреча»' AND payload->>'link'=$1`, "/events/"+e.ID).Scan(&body); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(body, location) {
			t.Fatalf("старый адрес в напоминании: %s", body)
		}
	})
	t.Run("отменённое событие не напоминает", func(t *testing.T) {
		e := create("open")
		if _, err := s.Join(ctx, participant, e.ID, "", false); err != nil {
			t.Fatal(err)
		}
		target, err := repo.ReminderTargetFor(ctx, owner, e.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.CancelEvent(ctx, owner, e.ID, "Планы изменились"); err != nil {
			t.Fatal(err)
		}
		if n, err := repo.SendReminder(ctx, target, "manual", ""); err != nil || n != 0 {
			t.Fatalf("отменённая встреча: %d %v", n, err)
		}
	})
	t.Run("нельзя подтвердить заявку после окончания", func(t *testing.T) {
		e := create("request")
		p, err := s.Join(ctx, participant, e.ID, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = db.Exec(ctx, "UPDATE events SET starts_at=now()-interval '2 hours',ends_at=now()-interval '1 hour' WHERE id=$1", e.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = repo.Decide(ctx, owner, e.ID, p.ID, true, ""); !errors.Is(err, ErrEventPast) {
			t.Fatalf("ожидали завершённую встречу: %v", err)
		}
	})
}
