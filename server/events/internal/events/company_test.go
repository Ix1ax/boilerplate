package events

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestValidateCompanyRequest(t *testing.T) {
	lat, lng := 56.01, 92.89
	zero, ok, far := 0, 5, 101
	cases := []struct {
		name  string
		in    CompanyRequestInput
		field string
	}{
		{"только направление", CompanyRequestInput{SportName: "Бег"}, ""},
		{"полный", CompanyRequestInput{SportName: "Бег", Level: "beginner", DayParts: []string{"morning"}, Weekdays: []int{6, 7}, Lat: &lat, Lng: &lng, RadiusKm: &ok}, ""},
		{"нет направления", CompanyRequestInput{SportName: "  "}, "sport"},
		{"чужой sport_id", CompanyRequestInput{SportID: "x"}, "sport"},
		{"неизвестный уровень", CompanyRequestInput{SportName: "Бег", Level: "master"}, "level"},
		{"неизвестное время", CompanyRequestInput{SportName: "Бег", DayParts: []string{"night"}}, "day_parts"},
		{"восьмой день", CompanyRequestInput{SportName: "Бег", Weekdays: []int{8}}, "weekdays"},
		{"нулевой день", CompanyRequestInput{SportName: "Бег", Weekdays: []int{0}}, "weekdays"},
		{"одна координата", CompanyRequestInput{SportName: "Бег", Lat: &lat}, "location"},
		{"радиус без точки", CompanyRequestInput{SportName: "Бег", RadiusKm: &ok}, "radius_km"},
		{"нулевой радиус", CompanyRequestInput{SportName: "Бег", Lat: &lat, Lng: &lng, RadiusKm: &zero}, "radius_km"},
		{"слишком далеко", CompanyRequestInput{SportName: "Бег", Lat: &lat, Lng: &lng, RadiusKm: &far}, "radius_km"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCompanyRequest(&tc.in)
			var fe FieldError
			switch {
			case tc.field == "" && err != nil:
				t.Fatalf("неожиданная ошибка: %v", err)
			case tc.field != "" && (!errors.As(err, &fe) || fe.Field != tc.field):
				t.Fatalf("ожидалась ошибка поля %s, получено %v", tc.field, err)
			}
		})
	}
}

func TestValidateCompanyRequestNormalizes(t *testing.T) {
	in := CompanyRequestInput{SportName: "  Скандинавская   ходьба ", DayParts: []string{"evening", "morning", "evening"}, Weekdays: []int{7, 6, 7}}
	if err := ValidateCompanyRequest(&in); err != nil {
		t.Fatal(err)
	}
	if in.SportName != "Скандинавская ходьба" || in.Level != "any" {
		t.Fatalf("%+v", in)
	}
	if !slices.Equal(in.DayParts, []string{"evening", "morning"}) || !slices.Equal(in.Weekdays, []int{6, 7}) {
		t.Fatalf("дубли не убраны: %v %v", in.DayParts, in.Weekdays)
	}
}

func TestHumanTimeUsesKrasnoyarsk(t *testing.T) {
	// 03:00 UTC в субботу — 10:00 по Красноярску.
	got := humanTime(time.Date(2026, 10, 24, 3, 0, 0, 0, time.UTC))
	if got != "сб, 24 октября в 10:00" {
		t.Fatal(got)
	}
}
