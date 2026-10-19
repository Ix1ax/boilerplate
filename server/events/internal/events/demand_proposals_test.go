package events

import (
	"testing"
	"time"
)

func TestDemandSlotIntersection(t *testing.T) {
	now := time.Date(2026, 10, 19, 8, 0, 0, 0, localZone)
	group := []demandCandidate{{Days: []int{6, 7}, Parts: []string{"morning"}, ExpiresAt: now.Add(14 * 24 * time.Hour)}, {Days: []int{6}, ExpiresAt: now.Add(14 * 24 * time.Hour)}, {Parts: []string{"morning", "day"}, ExpiresAt: now.Add(14 * 24 * time.Hour)}}
	slot, ok := demandSlot(now, group)
	if !ok || slot.Weekday() != time.Saturday || slot.Hour() != 10 {
		t.Fatalf("ожидали субботу 10:00: %v %v", slot, ok)
	}
	group[2].Parts = []string{"evening"}
	if _, ok = demandSlot(now, group); ok {
		t.Fatal("несовместимое время не должно давать предложение")
	}
}

func TestDemandLocationCompatibility(t *testing.T) {
	a := demandCandidate{Area: " Октябрьский район "}
	b := demandCandidate{Area: "октябрьский  район"}
	if !demandPlaceFits(a, b) {
		t.Fatal("одинаковые районы совместимы")
	}
	b.Area = "Центр"
	if demandPlaceFits(a, b) {
		t.Fatal("нельзя угадывать близость разных районов")
	}
	a.Lat, a.Lng = demandPtr(56.0), demandPtr(92.0)
	b.Lat, b.Lng, b.Radius = demandPtr(56.001), demandPtr(92.001), demandPtr(1)
	if !demandPlaceFits(a, b) {
		t.Fatal("точка рядом входит в радиус")
	}
	b.Lat = demandPtr(57.0)
	if demandPlaceFits(a, b) {
		t.Fatal("далёкая точка не подходит")
	}
}

func TestArrivalBoundariesAndInput(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Hour)
	for _, now := range []time.Time{start.Add(-30 * time.Minute), start, end.Add(30 * time.Minute)} {
		if !arrivalWindow(now, start, end) {
			t.Fatal("граничное время входит в окно")
		}
	}
	if arrivalWindow(start.Add(-31*time.Minute), start, end) || arrivalWindow(end.Add(31*time.Minute), start, end) {
		t.Fatal("вне окна отметки недоступны")
	}
	for _, in := range []ArrivalInput{{Status: "late", ETAMinutes: 0}, {Status: "late", ETAMinutes: 121}, {Status: "here", ETAMinutes: 10}, {Status: "unknown"}} {
		if validateArrival(&in) == nil {
			t.Fatalf("должна быть ошибка: %+v", in)
		}
	}
	in := ArrivalInput{Status: "late", ETAMinutes: 10, Description: " Оранжевая куртка "}
	if err := validateArrival(&in); err != nil || in.Description != "Оранжевая куртка" {
		t.Fatal(err)
	}
}

func demandPtr[T any](v T) *T { return &v }
