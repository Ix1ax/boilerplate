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

// Профиль, правка и перенос события, «найти другую встречу», поиск рядом, доверие к организатору.
func TestExtras(t *testing.T) {
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
	expect := func(err, want error) {
		t.Helper()
		if !errors.Is(err, want) {
			t.Fatalf("ожидалась ошибка %v, получено %v", want, err)
		}
	}
	org, anna, igor := person("Организатор"), person("Анна"), person("Игорь")

	// Профиль: правятся только переданные поля, пустой уровень — «не указан».
	about, city, level := "Бегаю по выходным", "Красноярск", "beginner"
	p, err := s.UpdateMyProfile(ctx, anna, ProfilePatch{About: &about, City: &city, Level: &level})
	must(err)
	empty := ""
	p, err = s.UpdateMyProfile(ctx, anna, ProfilePatch{Level: &empty})
	must(err)
	if p.About != about || p.City != city || p.Level != "" || p.Stats == nil {
		t.Fatalf("профиль: %+v", p)
	}

	// Перенос времени и места: записавшимся «было → стало», мест не меньше записанных.
	center, lng := 56.0106, 92.8710
	two := 2
	run, err := s.CreateEvent(ctx, org, EventInput{Title: "Пробежка", SportName: "Бег", StartsAt: time.Now().Add(48 * time.Hour),
		Location: "Набережная", Lat: &center, Lng: &lng, JoinMode: "open", Capacity: &two, Requirements: "Вода"})
	must(err)
	_, err = s.Join(ctx, anna, run.ID, "", false)
	must(err)
	_, err = s.Join(ctx, igor, run.ID, "", false)
	must(err)
	one := 1
	_, err = s.UpdateEvent(ctx, org, run.ID, EventPatch{Capacity: &one})
	var fe FieldError
	if !errors.As(err, &fe) || fe.Field != "capacity" {
		t.Fatalf("мест меньше, чем записано: %v", err)
	}
	_, err = s.UpdateEvent(ctx, anna, run.ID, EventPatch{Location: &city})
	expect(err, ErrNotEventOwner)
	later, place := run.StartsAt.Add(24*time.Hour), "Остров Татышев"
	moved, err := s.UpdateEvent(ctx, org, run.ID, EventPatch{StartsAt: &later, Location: &place})
	must(err)
	if !moved.StartsAt.Equal(later) || moved.Location != place || moved.Requirements != "Вода" {
		t.Fatalf("перенос: %+v", moved)
	}
	var body string
	must(db.QueryRow(ctx, "SELECT payload->>'body' FROM outbox WHERE user_id=$1 AND payload->>'title' LIKE 'Перенос:%'", anna.ID).Scan(&body))
	if !strings.Contains(body, "Было:") || !strings.Contains(body, "Набережная") || !strings.Contains(body, place) {
		t.Fatalf("уведомление о переносе: %q", body)
	}
	history, err := s.EventHistory(ctx, org, run.ID)
	must(err)
	if last := history[len(history)-1]; last.Comment != "Изменено: время, место" {
		t.Fatalf("история правки: %+v", last)
	}
	past := time.Now().Add(-time.Hour)
	_, err = s.UpdateEvent(ctx, org, run.ID, EventPatch{StartsAt: &past})
	if !errors.As(err, &fe) || fe.Field != "starts_at" {
		t.Fatalf("перенос в прошлое: %v", err)
	}

	// Поиск рядом: ближайшие первыми, расстояние в ответе; далёкие и без координат не попадают.
	academ, academLng := 56.0021, 92.7653
	far, err := s.CreateEvent(ctx, org, EventInput{Title: "Пробежка в Академгородке", SportName: "Бег", StartsAt: time.Now().Add(72 * time.Hour),
		Location: "Академгородок", Lat: &academ, Lng: &academLng, JoinMode: "open"})
	must(err)
	_, err = s.CreateEvent(ctx, org, EventInput{Title: "Пробежка без точки", SportName: "Бег", StartsAt: time.Now().Add(72 * time.Hour), Location: "Где-то", JoinMode: "open"})
	must(err)
	near, err := s.ListEvents(ctx, igor, EventFilter{Geo: &Geo{Lat: center, Lng: lng, RadiusKm: 3}, Limit: 10})
	must(err)
	if near.Total != 1 || near.Items[0].ID != run.ID || near.Items[0].DistanceKm == nil {
		t.Fatalf("рядом в 3 км: %+v", near.Items)
	}
	wide, err := s.ListEvents(ctx, igor, EventFilter{Geo: &Geo{Lat: center, Lng: lng, RadiusKm: 20}, Limit: 10})
	must(err)
	if wide.Total != 2 || wide.Items[0].ID != run.ID || *wide.Items[1].DistanceKm < 5 {
		t.Fatalf("рядом в 20 км: %+v", wide.Items)
	}
	box, err := s.ListEvents(ctx, igor, EventFilter{Geo: &Geo{BBox: &[4]float64{92.70, 55.99, 92.80, 56.01}}, Limit: 10})
	must(err)
	if box.Total != 1 || box.Items[0].ID != far.ID {
		t.Fatalf("область карты: %+v", box.Items)
	}

	// Найти другую встречу: то же направление, есть места, сначала рядом; заполненное — нет.
	alt, err := s.Alternatives(ctx, igor, far.ID, 5)
	must(err)
	for _, e := range alt {
		if e.ID == run.ID {
			t.Fatal("заполненное событие предложено как альтернатива")
		}
		if e.Sport.Name != "Бег" || e.ID == far.ID {
			t.Fatalf("альтернатива: %+v", e)
		}
	}
	if len(alt) != 1 {
		t.Fatalf("альтернатив %d, ожидалась одна (без точки)", len(alt))
	}

	// Доверие: прошедшая встреча с записями попадает в статистику организатора.
	_, err = db.Exec(ctx, "UPDATE events SET starts_at=now() - interval '2 days', ends_at=NULL, quorum_deadline=NULL WHERE id=$1", run.ID)
	must(err)
	prof, err := s.GetProfile(ctx, anna, org.ID)
	must(err)
	if prof.Stats.EventsHeld != 1 || prof.Stats.PeopleCame != 2 || prof.Stats.UpcomingEvents != 2 {
		t.Fatalf("статистика организатора: %+v", prof.Stats)
	}
}
