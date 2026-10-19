package events

import (
	"math"
	"testing"
	"time"
)

func TestRecommendationRanking(t *testing.T) {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	start := now.Add(24 * time.Hour)
	p := RecommendationPreferences{SportIDs: []string{"running"}, Level: "beginner", RadiusKm: 20, Weekdays: []int{2}, DayParts: []string{"morning"}}
	e := Event{ID: "run", Format: "offline", Sport: Sport{ID: "running"}, Level: "beginner", StartsAt: start, Organizer: Organizer{ID: "o"}}
	good, ok := rankRecommendation(e, p, nil, nil, now)
	if !ok {
		t.Fatal("подходящая встреча исключена")
	}
	other := e
	other.Sport.ID = "chess"
	neutral, _ := rankRecommendation(other, p, nil, nil, now)
	if good.score <= neutral.score {
		t.Fatal("явный интерес не влияет на порядок")
	}
	e.Level = "pro"
	if _, ok = rankRecommendation(e, p, nil, nil, now); ok {
		t.Fatal("новичку предложен несовместимый уровень")
	}
	e.Level = "any"
	if _, ok = rankRecommendation(e, p, nil, nil, now); !ok {
		t.Fatal("универсальный уровень исключён")
	}
	lat, lng, far := 56.0, 92.0, 10.0
	p.Lat = &lat
	p.Lng = &lng
	e.Lat = &far
	e.Lng = &lng
	if _, ok = rankRecommendation(e, p, nil, nil, now); ok {
		t.Fatal("далёкая офлайн-встреча прошла радиус")
	}
	e.Format = "online"
	if _, ok = rankRecommendation(e, p, nil, nil, now); !ok {
		t.Fatal("онлайн-встреча ошибочно ограничена радиусом")
	}
}

func TestRecommendationPreferenceValidation(t *testing.T) {
	for _, p := range []RecommendationPreferences{{Lat: floatPtr(math.NaN()), Lng: floatPtr(0)}, {Lat: floatPtr(56)}, {Weekdays: []int{0}}, {DayParts: []string{"night"}}, {RadiusKm: 101}, {SportIDs: []string{"bad"}}} {
		if validatePreferences(&p) == nil {
			t.Fatalf("некорректная настройка принята: %+v", p)
		}
	}
}
func floatPtr(v float64) *float64 { return &v }
