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

// Админские операции домена и доменные события для сервиса moderation.
func TestAdminAndDomainEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := isolatedDB(t, ctx)
	s := New(NewRepository(db))
	person := func(name string) Actor {
		a := Actor{ID: "it-" + uuid.NewString(), Name: name}
		if err := s.EnsureProfile(ctx, a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	owner, anna, admin := person("Олег"), person("Анна"), person("Модератор")
	admin.Admin = true

	org, err := s.SubmitOrganization(ctx, owner, OrganizationInput{Name: "Ладья", Description: "Шахматный клуб для всех"})
	must(err)
	_, err = s.ReviewOrganization(ctx, admin, org.ID, "approved", "")
	must(err)
	ev, err := s.CreateEvent(ctx, owner, EventInput{Title: "Блиц", SportName: "Шахматы", StartsAt: time.Now().Add(48 * time.Hour), Location: "Библиотека", JoinMode: "open", AsOrganization: true})
	must(err)
	_, err = s.ModerateEvent(ctx, admin, ev.ID, "unpublished", "Неверный адрес")
	must(err)

	// Карточка организации для админа: история допуска и события во всех статусах.
	detail, err := s.AdminOrganization(ctx, admin, org.ID)
	must(err)
	if len(detail.History) != 2 || detail.History[1].To != "approved" || len(detail.Events) != 1 || detail.Events[0].Status != "unpublished" {
		t.Fatalf("карточка организации: %+v", detail)
	}
	history, err := s.EventHistory(ctx, admin, ev.ID)
	must(err)
	if len(history) != 2 || history[1].To != "unpublished" || history[1].Comment != "Неверный адрес" {
		t.Fatalf("история события: %+v", history)
	}

	// Доменное событие несёт снимок после изменения: подписчику не нужно идти в эту базу.
	var payload []byte
	must(db.QueryRow(ctx, `SELECT payload FROM outbox WHERE type='domain.event.changed' AND payload->>'id'=$1 AND payload->>'to'='unpublished'`, ev.ID).Scan(&payload))
	var env struct{ Entity, Actor, Comment string }
	must(json.Unmarshal(payload, &env))
	var snap map[string]any
	var raw struct {
		Snapshot json.RawMessage `json:"snapshot"`
	}
	must(json.Unmarshal(payload, &raw))
	must(json.Unmarshal(raw.Snapshot, &snap))
	if env.Entity != "event" || env.Comment != "Неверный адрес" || snap["status"] != "unpublished" || snap["organization_name"] != "Ладья" || snap["title"] != "Блиц" {
		t.Fatalf("доменное событие: %s", payload)
	}
	var kinds int
	must(db.QueryRow(ctx, `SELECT count(DISTINCT type) FROM outbox WHERE type LIKE 'domain.%'`).Scan(&kinds))
	if kinds != 2 { // организация и событие
		t.Fatalf("типов доменных событий: %d", kinds)
	}

	// Пользовательское направление: модератор скрывает спам, базовое скрыть нельзя.
	_, err = s.CreateEvent(ctx, anna, EventInput{Title: "Реклама", SportName: "Купи абонемент", StartsAt: time.Now().Add(48 * time.Hour), Location: "Где-то", JoinMode: "open"})
	must(err)
	sports, err := s.AdminSports(ctx)
	must(err)
	var spam, base AdminSport
	for _, sp := range sports {
		switch {
		case sp.Name == "Купи абонемент":
			spam = sp
		case !sp.IsCustom && base.ID == "":
			base = sp
		}
	}
	if spam.ID == "" || !spam.IsCustom || spam.EventsCount != 1 || spam.CreatedBy != "Анна" {
		t.Fatalf("пользовательское направление: %+v", spam)
	}
	_, err = s.SetSportHidden(ctx, admin, spam.ID, true, "")
	if !errors.Is(err, platform.ErrCommentRequired) {
		t.Fatalf("скрытие без причины: %v", err)
	}
	hidden, err := s.SetSportHidden(ctx, admin, spam.ID, true, "Реклама")
	must(err)
	if !hidden.Hidden {
		t.Fatal("направление не скрыто")
	}
	_, err = s.SetSportHidden(ctx, admin, base.ID, true, "Тест")
	if !errors.Is(err, ErrBaseSport) {
		t.Fatalf("скрытие базового направления: %v", err)
	}
	_, err = s.CreateEvent(ctx, anna, EventInput{Title: "Ещё реклама", SportName: "купи абонемент", StartsAt: time.Now().Add(48 * time.Hour), Location: "Где-то", JoinMode: "open"})
	if !errors.Is(err, ErrSportHidden) {
		t.Fatalf("событие со скрытым направлением: %v", err)
	}
	list, err := s.Sports(ctx)
	must(err)
	for _, sp := range list {
		if sp.ID == spam.ID {
			t.Fatal("скрытое направление осталось в справочнике")
		}
	}
	_, err = s.SetSportHidden(ctx, admin, spam.ID, false, "")
	must(err)

	// Переотправка снимков догоняет подписчика: по снимку на каждую сущность.
	n, err := s.RepublishDomain(ctx)
	must(err)
	if n != 3 { // организация и два события; сообщений, записей и запросов нет
		t.Fatalf("переотправлено %d снимков", n)
	}
}
