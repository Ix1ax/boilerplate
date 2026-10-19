//go:build integration

package events

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/server/events/internal/weather"
	"github.com/google/uuid"
)

type fakeForecast struct{}

func (fakeForecast) At(context.Context, float64, float64, time.Time) (weather.Forecast, error) {
	return weather.Forecast{TemperatureC: 5, Description: "пасмурно", Advice: "прохладно: возьмите кофту"}, nil
}

// «Соберём состав», «Встречу новичка» и напоминания на настоящем Postgres.
func TestGatherings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := isolatedDB(t, ctx)
	s := New(NewRepository(db))
	s.Weather = fakeForecast{}
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
	expect := func(err, want error) {
		t.Helper()
		if !errors.Is(err, want) {
			t.Fatalf("ожидалась ошибка %v, получено %v", want, err)
		}
	}
	// inbox — заголовки и тексты уведомлений человека из outbox, по порядку.
	inbox := func(a Actor) []string {
		t.Helper()
		rows, err := db.Query(ctx, "SELECT (payload->>'title') || ' | ' || (payload->>'body') FROM outbox WHERE user_id=$1 AND type='notification.requested' ORDER BY created_at, id", a.ID)
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
	has := func(a Actor, prefix string) int {
		n := 0
		for _, m := range inbox(a) {
			if strings.HasPrefix(m, prefix) {
				n++
			}
		}
		return n
	}
	org, anna, igor, petr, maria, stranger := person("Организатор"), person("Анна"), person("Игорь"), person("Пётр"), person("Мария"), person("Гость")
	lat, lng := 56.0153, 92.8932
	three, ten := 3, 10
	game, err := s.CreateEvent(ctx, org, EventInput{Title: "Футбол 5×5", SportName: "Футбол", StartsAt: time.Now().Add(48 * time.Hour),
		Location: "Стадион «Локомотив»", Lat: &lat, Lng: &lng, Outdoor: true, JoinMode: "open", Capacity: &ten, MinParticipants: &three,
		PrivateInfo: "Северная трибуна", Requirements: "Бутсы или кеды"})
	must(err)
	if game.QuorumStatus != "gathering" || game.QuorumDeadline == nil || *game.SpotsLeft != 10 {
		t.Fatalf("новое событие с минимумом: %+v", game)
	}

	// Новичок без проводников: организатора просят встретить самому.
	_, err = s.Join(ctx, anna, game.ID, "", true)
	must(err)
	if has(org, "Новичка некому встретить") != 1 {
		t.Fatal("организатор не узнал, что новичка некому встретить")
	}
	_, err = s.Join(ctx, igor, game.ID, "", false)
	must(err)
	_, err = s.SetBuddy(ctx, anna, game.ID, true)
	expect(err, ErrNewcomerBuddy)
	_, err = s.SetBuddy(ctx, stranger, game.ID, true)
	expect(err, ErrNotConfirmed)
	me, err := s.SetBuddy(ctx, igor, game.ID, true)
	must(err)
	if !me.IsBuddy || len(me.Newcomers) != 1 || me.Newcomers[0].Name != "Анна" {
		t.Fatalf("проводник не получил новичка: %+v", me)
	}
	card, err := s.GetEvent(ctx, anna, game.ID)
	must(err)
	if card.MyParticipation.Buddy == nil || card.MyParticipation.Buddy.Name != "Игорь" || card.FirstTimersCount != 1 || card.BuddiesCount != 1 {
		t.Fatalf("карточка новичка: %+v / %+v", card.MyParticipation, card)
	}
	if has(anna, "Вас встретит Игорь") != 1 || has(igor, "Анна придёт впервые") != 1 {
		t.Fatal("пара не получила уведомления")
	}
	roster, err := s.Participants(ctx, org, game.ID)
	must(err)
	for _, p := range roster {
		if p.User.Name == "Анна" && (p.Buddy == nil || !p.FirstTime) {
			t.Fatalf("организатор не видит пару: %+v", p)
		}
	}
	public, err := s.Participants(ctx, petr, game.ID)
	must(err)
	for _, p := range public {
		if p.Buddy != nil {
			t.Fatal("пары видны постороннему")
		}
	}

	// Третий участник — состав собран, всем объявляем.
	_, err = s.Join(ctx, petr, game.ID, "", false)
	must(err)
	card, err = s.GetEvent(ctx, petr, game.ID)
	must(err)
	if card.QuorumStatus != "reached" || *card.SpotsLeft != 7 {
		t.Fatalf("после третьего: %s, мест %d", card.QuorumStatus, *card.SpotsLeft)
	}
	for _, a := range []Actor{anna, igor, petr, org} {
		if has(a, "Состав собран") != 1 {
			t.Fatalf("%s не получил «Состав собран»", a.Name)
		}
	}

	// Проводник уходит — новичка забирает другой; передумал и он — новичок и организатор предупреждены.
	_, err = s.Join(ctx, maria, game.ID, "", false)
	must(err)
	_, err = s.SetBuddy(ctx, maria, game.ID, true)
	must(err)
	must(s.Leave(ctx, igor, game.ID))
	if has(anna, "Вас встретит Мария") != 1 {
		t.Fatalf("новичка не передали другому проводнику: %v", inbox(anna))
	}
	card, err = s.GetEvent(ctx, petr, game.ID)
	must(err)
	if card.QuorumStatus != "reached" {
		t.Fatal("объявленный состав откатился после отмены записи")
	}
	_, err = s.SetBuddy(ctx, maria, game.ID, false)
	must(err)
	if has(anna, "Проводник не сможет прийти") != 1 || has(org, "Новичка некому встретить") != 2 {
		t.Fatal("о новичке без проводника не предупредили")
	}

	// Не набрали к сроку — автоотмена с причиной, записавшиеся предупреждены.
	five := 5
	small, err := s.CreateEvent(ctx, org, EventInput{Title: "Баскетбол 3×3", SportName: "Баскетбол", StartsAt: time.Now().Add(48 * time.Hour),
		Location: "Школа №10", JoinMode: "open", MinParticipants: &five})
	must(err)
	_, err = s.Join(ctx, petr, small.ID, "", false)
	must(err)
	_, err = db.Exec(ctx, "UPDATE events SET quorum_deadline=now() - interval '1 minute' WHERE id=$1", small.ID)
	must(err)
	must(s.FailQuorums(ctx))
	failed, err := s.GetEvent(ctx, petr, small.ID)
	must(err)
	if failed.Status != "cancelled" || failed.QuorumStatus != "failed" || !strings.Contains(failed.ModerationReason, "1 из 5") {
		t.Fatalf("автоотмена: %s %s %q", failed.Status, failed.QuorumStatus, failed.ModerationReason)
	}
	if has(petr, "Встреча не состоится") != 1 || has(org, "Встреча не состоится") != 1 {
		t.Fatal("об автоотмене не предупредили")
	}
	must(s.FailQuorums(ctx))
	if has(petr, "Встреча не состоится") != 1 {
		t.Fatal("повторный запуск задания продублировал отмену")
	}
	_, err = s.Join(ctx, maria, small.ID, "", false)
	expect(err, ErrEventClosed)

	// Напоминание за сутки: один раз, с точкой сбора, погодой и проводником.
	walk, err := s.CreateEvent(ctx, org, EventInput{Title: "Скандинавская ходьба", SportName: "Скандинавская ходьба", StartsAt: time.Now().Add(20 * time.Hour),
		Location: "Центральный парк", Lat: &lat, Lng: &lng, Outdoor: true, JoinMode: "open", PrivateInfo: "У колеса обозрения"})
	must(err)
	_, err = db.Exec(ctx, "UPDATE events SET created_at=now() - interval '3 days' WHERE id=$1", walk.ID)
	must(err)
	_, err = s.Join(ctx, igor, walk.ID, "", false)
	must(err)
	_, err = s.SetBuddy(ctx, igor, walk.ID, true)
	must(err)
	_, err = s.Join(ctx, maria, walk.ID, "", true)
	must(err)
	must(s.SendReminders(ctx))
	must(s.SendReminders(ctx))
	got := inbox(maria)
	reminder := ""
	for _, m := range got {
		if strings.HasPrefix(m, "Напоминание: «Скандинавская ходьба»") {
			if reminder != "" {
				t.Fatal("напоминание пришло дважды")
			}
			reminder = m
		}
	}
	for _, want := range []string{"У колеса обозрения", "Погода: +5°, пасмурно", "Вас встретит Игорь"} {
		if !strings.Contains(reminder, want) {
			t.Fatalf("в напоминании нет %q: %q", want, reminder)
		}
	}
	if has(anna, "Напоминание: «Футбол") != 0 {
		t.Fatal("напоминание о встрече через двое суток ушло раньше времени")
	}

	// Кнопка «Напомнить»: только организатор и не чаще раза в час.
	_, err = s.Remind(ctx, igor, walk.ID)
	expect(err, ErrNotOrganizer)
	n, err := s.Remind(ctx, org, walk.ID)
	must(err)
	if n != 2 {
		t.Fatalf("напоминание получили %d", n)
	}
	_, err = s.Remind(ctx, org, walk.ID)
	expect(err, ErrRemindedRecently)

	// Погода: для уличной встречи — прогноз, для зала — понятная причина.
	w, err := s.EventWeather(ctx, maria, walk.ID)
	must(err)
	if !w.Available || w.Forecast.Advice == "" {
		t.Fatalf("погода: %+v", w)
	}
	w, err = s.EventWeather(ctx, petr, small.ID)
	must(err)
	if w.Available || w.Reason == "" {
		t.Fatalf("погода для зала: %+v", w)
	}
}
