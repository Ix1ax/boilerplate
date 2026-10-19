package events

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"
)

func TestParseGeo(t *testing.T) {
	if g, err := parseGeo("", "", ""); g != nil || err != nil {
		t.Fatal("без параметров фильтра нет")
	}
	g, err := parseGeo("56.01,92.87", "", "")
	if err != nil || g.Lat != 56.01 || g.RadiusKm != defaultRadiusKm {
		t.Fatalf("near: %+v %v", g, err)
	}
	g, err = parseGeo("", "", "92.7,55.9,93.1,56.1")
	if err != nil || g.BBox == nil || g.BBox[2] != 93.1 {
		t.Fatalf("bbox: %+v %v", g, err)
	}
	for _, bad := range [][3]string{{"91,0", "", ""}, {"56,92", "0", ""}, {"56,92", "101", ""}, {"", "", "93,56,92,55"}, {"56,92", "", "1,2,3,4"}, {"56", "", ""}, {"a,b", "", ""}} {
		if _, err := parseGeo(bad[0], bad[1], bad[2]); err == nil {
			t.Errorf("принято %v", bad)
		}
	}
}

func TestDistanceKm(t *testing.T) {
	// Центр Красноярска — Академгородок: около 7 км.
	if d := DistanceKm(56.0106, 92.8710, 56.0021, 92.7653); math.Abs(d-6.6) > 0.5 {
		t.Fatal(d)
	}
}

func TestICS(t *testing.T) {
	lat, lng := 56.01, 92.87
	e := Event{ID: "id-1", Title: "Пробежка; вдоль, Енисея", Location: "Набережная", Requirements: "Вода", PrivateInfo: "У моста",
		StartsAt: time.Date(2026, 10, 24, 3, 0, 0, 0, time.UTC), Lat: &lat, Lng: &lng, Organizer: Organizer{Name: "Игорь"},
		Description: strings.Repeat("длинное описание ", 10)}
	got := ICS(e, time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC))
	for _, want := range []string{"BEGIN:VCALENDAR\r\n", "DTSTART:20261024T030000Z\r\n", "DTEND:20261024T043000Z\r\n",
		`SUMMARY:Пробежка\; вдоль\, Енисея`, "GEO:56.010000;92.870000", "TRIGGER:-PT2H", "END:VCALENDAR\r\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("нет %q", want)
		}
	}
	for _, line := range strings.Split(got, "\r\n") {
		if len(line) > 75 {
			t.Fatalf("строка длиннее 75 байт: %q", line)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	long, bad, ok := strings.Repeat("я", 1001), "master", " beginner "
	for _, tc := range []struct {
		p     ProfilePatch
		field string
	}{
		{ProfilePatch{About: &long}, "about"},
		{ProfilePatch{Level: &bad}, "level"},
		{ProfilePatch{Level: &ok}, ""},
	} {
		err := ValidateProfile(&tc.p)
		var fe FieldError
		if tc.field == "" && err != nil || tc.field != "" && (!errors.As(err, &fe) || fe.Field != tc.field) {
			t.Errorf("%+v: %v", tc.p, err)
		}
	}
}
