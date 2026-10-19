package events

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// «Ищу компанию» — главная собственная идея продукта. Человек, не нашедший подходящего события,
// описывает, чем и когда хочет заниматься. Новое подходящее событие приходит ему уведомлением
// «Нашлась компания», а организатор ещё при создании видит, сколько людей ищут такую активность.
// Подбор — в SQL (companyMatch в company_repository.go), в той же транзакции, что и публикация.

// companyRequestTTL — срок жизни запроса: месячной давности интерес уже не должен будить уведомлениями.
const companyRequestTTL = 30 * 24 * time.Hour

// maxRadiusKm — радиус поиска по точке; дальше это уже не «рядом».
const maxRadiusKm = 100

var ErrRequestExists = errors.New("У вас уже есть активный запрос на это направление: измените его или закройте")

var dayParts = map[string]bool{"morning": true, "day": true, "evening": true}

type CompanyRequestInput struct {
	SportID   string   `json:"sport_id"`
	SportName string   `json:"sport_name"`
	Level     string   `json:"level"`
	DayParts  []string `json:"day_parts"`
	Weekdays  []int    `json:"weekdays"`
	Area      string   `json:"area"`
	Lat       *float64 `json:"lat"`
	Lng       *float64 `json:"lng"`
	RadiusKm  *int     `json:"radius_km"`
	Note      string   `json:"note"`
}

type CompanyRequest struct {
	ID        string    `json:"id"`
	Sport     Sport     `json:"sport"`
	Level     string    `json:"level"`
	DayParts  []string  `json:"day_parts"`
	Weekdays  []int     `json:"weekdays"`
	Area      string    `json:"area"`
	Lat       *float64  `json:"lat"`
	Lng       *float64  `json:"lng"`
	RadiusKm  *int      `json:"radius_km"`
	Note      string    `json:"note"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	// MatchesCount — подходящие предстоящие события; OthersCount — сколько ещё людей ищут
	// компанию на то же направление («вы не один»), без имён.
	MatchesCount int `json:"matches_count"`
	OthersCount  int `json:"others_count"`
}

// Demand — обезличенный спрос для организатора: сколько людей ищут компанию и когда им удобно.
type Demand struct {
	SportID   *string        `json:"sport_id"`
	Count     int            `json:"count"`
	ByDayPart map[string]int `json:"by_day_part"`
	// ByWeekday — семь чисел, с понедельника по воскресенье.
	ByWeekday []int `json:"by_weekday"`
}

// ValidateCompanyRequest нормализует форму: убирает дубли во времени и днях, пустой уровень — «не важно».
func ValidateCompanyRequest(in *CompanyRequestInput) error {
	in.SportID, in.SportName = strings.TrimSpace(in.SportID), strings.Join(strings.Fields(in.SportName), " ")
	in.Area, in.Note = strings.TrimSpace(in.Area), strings.TrimSpace(in.Note)
	if in.Level == "" {
		in.Level = "any"
	}
	in.DayParts = uniqueSorted(in.DayParts)
	in.Weekdays = uniqueSorted(in.Weekdays)
	switch {
	case in.SportID == "" && in.SportName == "":
		return FieldError{"sport", "Выберите направление или впишите своё"}
	case in.SportID != "" && uuid.Validate(in.SportID) != nil:
		return FieldError{"sport", "Неизвестное направление"}
	case in.SportID == "" && !between(in.SportName, 2, 60):
		return FieldError{"sport", "Название направления — от 2 до 60 символов"}
	case !levels[in.Level]:
		return FieldError{"level", "Уровень: any, beginner, amateur или pro"}
	case slices.ContainsFunc(in.DayParts, func(p string) bool { return !dayParts[p] }):
		return FieldError{"day_parts", "Время: morning, day или evening"}
	case slices.ContainsFunc(in.Weekdays, func(d int) bool { return d < 1 || d > 7 }):
		return FieldError{"weekdays", "Дни недели — числа от 1 (понедельник) до 7 (воскресенье)"}
	case utf8.RuneCountInString(in.Area) > 200:
		return FieldError{"area", "Район — не длиннее 200 символов"}
	case (in.Lat == nil) != (in.Lng == nil) || in.Lat != nil && (*in.Lat < -90 || *in.Lat > 90 || *in.Lng < -180 || *in.Lng > 180):
		return FieldError{"location", "Некорректные координаты"}
	case in.RadiusKm != nil && in.Lat == nil:
		return FieldError{"radius_km", "Радиус задаётся вместе с точкой на карте"}
	case in.RadiusKm != nil && (*in.RadiusKm < 1 || *in.RadiusKm > maxRadiusKm):
		return FieldError{"radius_km", "Радиус — от 1 до 100 км"}
	case utf8.RuneCountInString(in.Note) > 500:
		return FieldError{"note", "Комментарий — не длиннее 500 символов"}
	}
	return nil
}

// uniqueSorted всегда возвращает непустой срез: nil ушёл бы в Postgres как NULL, а «пусто» значит «когда угодно».
func uniqueSorted[T int | string](in []T) []T {
	out := append([]T{}, in...)
	slices.Sort(out)
	return slices.Compact(out)
}

func (s *Service) CreateCompanyRequest(ctx context.Context, a Actor, in CompanyRequestInput) (CompanyRequest, error) {
	if err := ValidateCompanyRequest(&in); err != nil {
		return CompanyRequest{}, err
	}
	return s.Store.CreateCompanyRequest(ctx, a, in)
}

// Demand отвечает организатору на «есть ли спрос»: по направлению из справочника или вписанному своему.
func (s *Service) Demand(ctx context.Context, a Actor, sportID, sportName, level string) (Demand, error) {
	sportID, sportName = strings.TrimSpace(sportID), strings.Join(strings.Fields(sportName), " ")
	if level == "" {
		level = "any"
	}
	switch {
	case sportID == "" && sportName == "":
		return Demand{}, FieldError{"sport", "Укажите sport_id или sport_name"}
	case sportID != "" && uuid.Validate(sportID) != nil:
		return Demand{}, FieldError{"sport", "Неизвестное направление"}
	case utf8.RuneCountInString(sportName) > 60:
		return Demand{}, FieldError{"sport", "Название направления — не длиннее 60 символов"}
	case !levels[level]:
		return Demand{}, FieldError{"level", "Уровень: any, beginner, amateur или pro"}
	}
	return s.Store.Demand(ctx, a, sportID, sportName, level)
}
