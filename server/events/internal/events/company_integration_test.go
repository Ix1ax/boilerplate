//go:build integration

package events

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// nextLocal — ближайший такой день недели (ISO, 1 — понедельник) не раньше чем через двое суток,
// в заданный час по Красноярску. Подбор считает время суток и день по местному времени.
func nextLocal(isoWeekday, hour int) time.Time {
	t := time.Now().In(localZone).Add(48 * time.Hour)
	for int((t.Weekday()+6)%7)+1 != isoWeekday {
		t = t.Add(24 * time.Hour)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), hour, 0, 0, 0, localZone)
}

// «Ищу компанию» на настоящем Postgres: правила подбора живут в SQL.
func TestCompanyMatching(t *testing.T) {
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
	invites := func(a Actor) []string {
		t.Helper()
		rows, err := db.Query(ctx, "SELECT payload->>'title' || ' | ' || (payload->>'body') FROM outbox WHERE user_id=$1 AND type='notification.requested' ORDER BY created_at", a.ID)
		must(err)
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			must(rows.Scan(&s))
			out = append(out, s)
		}
		return out
	}
	sport := "Тест-бег " + uuid.NewString()[:8]
	saturdayMorning, saturdayEvening, mondayMorning := nextLocal(6, 10), nextLocal(6, 19), nextLocal(1, 9)
	lat, lng, km := 56.0153, 92.8932, 5
	farLat, farLng := 56.10, 93.10 // ~16 км от точки
	anna, igor, petr, maria, organizer := person("Анна"), person("Игорь"), person("Пётр"), person("Мария"), person("Организатор")

	// Событие, которое уже есть на платформе, попадает в подборку сразу при создании запроса — без уведомления.
	early, err := s.CreateEvent(ctx, organizer, EventInput{Title: "Пробежка заранее", SportName: sport, StartsAt: saturdayMorning, Location: "Набережная", JoinMode: "open"})
	must(err)
	req, err := s.CreateCompanyRequest(ctx, anna, CompanyRequestInput{SportName: sport, Level: "beginner", DayParts: []string{"morning"}, Weekdays: []int{6, 7}, Lat: &lat, Lng: &lng, RadiusKm: &km})
	must(err)
	if req.Sport.ID != early.Sport.ID || req.MatchesCount != 1 || req.Status != "active" || req.OthersCount != 0 {
		t.Fatalf("запрос: %+v", req)
	}
	if len(invites(anna)) != 0 {
		t.Fatal("за уже существующее событие пришло уведомление")
	}
	_, err = s.CreateCompanyRequest(ctx, anna, CompanyRequestInput{SportName: sport})
	if !errors.Is(err, ErrRequestExists) {
		t.Fatalf("второй активный запрос на то же направление: %v", err)
	}
	_, err = s.CreateCompanyRequest(ctx, igor, CompanyRequestInput{SportName: sport, DayParts: []string{"evening"}})
	must(err)
	_, err = s.CreateCompanyRequest(ctx, petr, CompanyRequestInput{SportName: sport, Level: "pro"})
	must(err)
	_, err = s.CreateCompanyRequest(ctx, maria, CompanyRequestInput{SportName: sport})
	must(err)

	// Спрос для организатора: чужие запросы с совместимым уровнем, без самого спрашивающего.
	d, err := s.Demand(ctx, organizer, "", sport, "beginner")
	must(err)
	if d.Count != 3 || d.ByDayPart["morning"] != 2 || d.ByDayPart["evening"] != 2 || d.ByWeekday[5] != 3 || d.ByWeekday[0] != 2 {
		t.Fatalf("спрос новичков (Анна, Игорь, Мария; Пётр — pro): %+v", d)
	}
	none, err := s.Demand(ctx, organizer, "", "Нет такого "+uuid.NewString()[:6], "any")
	must(err)
	if none.Count != 0 || none.SportID != nil {
		t.Fatalf("спрос на незаведённое направление: %+v", none)
	}

	// Утро субботы для новичков рядом: Анна (утро, рядом) и Мария (когда угодно) получают уведомление;
	// Игорь ищет вечер, Пётр — уровень pro.
	near, err := s.CreateEvent(ctx, organizer, EventInput{Title: "Пробежка для новичков", SportName: sport, StartsAt: saturdayMorning, Location: "Остров Татышев", Lat: &lat, Lng: &lng, Level: "beginner", JoinMode: "open"})
	must(err)
	if near.CompanyMatches != 2 {
		t.Fatalf("событие получили %d человек, ожидалось 2", near.CompanyMatches)
	}
	if got := invites(anna); len(got) != 1 || !strings.Contains(got[0], "Нашлась компания") || !strings.Contains(got[0], "в 10:00") {
		t.Fatalf("уведомление Анне: %v", got)
	}
	if len(invites(maria)) != 1 || len(invites(igor)) != 0 || len(invites(petr)) != 0 {
		t.Fatal("уведомления ушли не тем людям")
	}

	// Далеко от точки Анны и в понедельник: Анне не подходит (радиус, день), Марии — подходит.
	_, err = s.CreateEvent(ctx, organizer, EventInput{Title: "Пробежка далеко", SportName: sport, StartsAt: saturdayMorning, Location: "Академгородок", Lat: &farLat, Lng: &farLng, JoinMode: "open"})
	must(err)
	_, err = s.CreateEvent(ctx, organizer, EventInput{Title: "Пробежка в понедельник", SportName: sport, StartsAt: mondayMorning, Location: "Набережная", JoinMode: "open"})
	must(err)
	if len(invites(anna)) != 1 || len(invites(maria)) != 3 {
		t.Fatalf("радиус или день недели не учтены: Анна %d, Мария %d", len(invites(anna)), len(invites(maria)))
	}
	// Вечер — Игорю; своё событие самому себе не подбирается.
	_, err = s.CreateEvent(ctx, igor, EventInput{Title: "Вечерняя пробежка", SportName: sport, StartsAt: saturdayEvening, Location: "Набережная", JoinMode: "open"})
	must(err)
	if len(invites(igor)) != 0 {
		t.Fatal("автору подобралось его собственное событие")
	}

	// Подборка Анны: событие «заранее» и «для новичков»; чужая подборка недоступна.
	page, err := s.ListEvents(ctx, anna, EventFilter{MatchedFor: req.ID, Limit: 10})
	must(err)
	if page.Total != 2 {
		t.Fatalf("в подборке Анны %d событий", page.Total)
	}
	foreign, err := s.ListEvents(ctx, igor, EventFilter{MatchedFor: req.ID, Limit: 10})
	must(err)
	if foreign.Total != 0 {
		t.Fatal("чужая подборка видна")
	}
	if _, err = s.MyCompanyRequest(ctx, igor, req.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("чужой запрос открылся")
	}

	// Снятое событие уходит из подборки; закрытый запрос больше не получает уведомлений.
	admin := person("Модератор")
	admin.Admin = true
	_, err = s.ModerateEvent(ctx, admin, early.ID, "unpublished", "Проверка")
	must(err)
	mine, err := s.MyCompanyRequest(ctx, anna, req.ID)
	must(err)
	if mine.MatchesCount != 1 {
		t.Fatalf("снятое событие осталось в подборке: %d", mine.MatchesCount)
	}
	must(s.CloseCompanyRequest(ctx, anna, req.ID))
	if err = s.CloseCompanyRequest(ctx, anna, req.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("повторное закрытие")
	}
	_, err = s.CreateEvent(ctx, organizer, EventInput{Title: "Ещё пробежка", SportName: sport, StartsAt: saturdayMorning, Location: "Набережная", JoinMode: "open"})
	must(err)
	if len(invites(anna)) != 1 {
		t.Fatal("закрытый запрос получил уведомление")
	}
	// После закрытия можно снова искать компанию на то же направление.
	_, err = s.CreateCompanyRequest(ctx, anna, CompanyRequestInput{SportName: sport})
	must(err)
}
