package events

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Поиск по месту, «найти другую встречу» и «добавить в календарь».

// Geo — фильтр выдачи по месту: точка с радиусом («рядом со мной») или видимая область карты.
// События без координат в такую выдачу не попадают — их показывает обычный список.
type Geo struct {
	Lat, Lng, RadiusKm float64
	// BBox — minLng, minLat, maxLng, maxLat (порядок как у Leaflet getBounds().toBBoxString()).
	BBox *[4]float64
}

const (
	defaultRadiusKm = 10
	maxRadiusKmGeo  = 100
	earthRadiusKm   = 6371
)

// distanceSQL — расстояние от события e до точки ($lat, $lng) по гаверсинусу, в километрах.
func distanceSQL(lat, lng string) string {
	return fmt.Sprintf(`(2 * %d * asin(sqrt(power(sin(radians(e.lat - %[2]s) / 2), 2) +
cos(radians(%[2]s)) * cos(radians(e.lat)) * power(sin(radians(e.lng - %[3]s) / 2), 2))))`, earthRadiusKm, lat, lng)
}

// DistanceKm — то же расстояние в Go: для ответа, чтобы не тащить вычисленную колонку через общий SELECT.
func DistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	rad := func(d float64) float64 { return d * math.Pi / 180 }
	h := math.Pow(math.Sin(rad(lat2-lat1)/2), 2) + math.Cos(rad(lat1))*math.Cos(rad(lat2))*math.Pow(math.Sin(rad(lng2-lng1)/2), 2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(h))
}

// parseGeo разбирает near=lat,lng&radius_km= или bbox=minLng,minLat,maxLng,maxLat.
func parseGeo(near, radius, bbox string) (*Geo, error) {
	if near == "" && bbox == "" {
		return nil, nil
	}
	if near != "" && bbox != "" {
		return nil, errors.New("near и bbox вместе не задаются")
	}
	if bbox != "" {
		v, err := floats(bbox, 4)
		if err != nil || v[0] >= v[2] || v[1] >= v[3] || !validLat(v[1]) || !validLat(v[3]) || !validLng(v[0]) || !validLng(v[2]) {
			return nil, errors.New("bbox: minLng,minLat,maxLng,maxLat")
		}
		b := [4]float64{v[0], v[1], v[2], v[3]}
		return &Geo{BBox: &b}, nil
	}
	v, err := floats(near, 2)
	if err != nil || !validLat(v[0]) || !validLng(v[1]) {
		return nil, errors.New("near: широта,долгота, например 56.01,92.87")
	}
	g := &Geo{Lat: v[0], Lng: v[1], RadiusKm: defaultRadiusKm}
	if radius != "" {
		r, err := strconv.ParseFloat(radius, 64)
		if err != nil || math.IsNaN(r) || math.IsInf(r, 0) || r <= 0 || r > maxRadiusKmGeo {
			return nil, errors.New("radius_km: от 0 до 100")
		}
		g.RadiusKm = r
	}
	return g, nil
}

func floats(s string, n int) ([]float64, error) {
	parts := strings.Split(s, ",")
	if len(parts) != n {
		return nil, errors.New("неверное число значений")
	}
	out := make([]float64, n)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, err
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, errors.New("координаты должны быть конечными числами")
		}
		out[i] = v
	}
	return out, nil
}

func validLat(v float64) bool { return v >= -90 && v <= 90 }
func validLng(v float64) bool { return v >= -180 && v <= 180 }

// Alternatives — «найти другую встречу»: если событие отменили, сняли, места кончились или заявку
// отклонили, путь не обрывается. То же направление, опубликованные предстоящие, ближайшие по времени
// к исходному; при координатах — сначала в пределах 15 км.
func (r *Repository) Alternatives(ctx context.Context, a Actor, id string, limit int) ([]Event, error) {
	src, err := r.GetEvent(ctx, a, id)
	if err != nil {
		return nil, err
	}
	near := "0"
	args := []any{a.ID, id, src.Sport.ID, src.StartsAt, limit}
	if src.Lat != nil {
		args = append(args, *src.Lat, *src.Lng)
		near = "CASE WHEN e.lat IS NOT NULL AND " + distanceSQL("$6", "$7") + " <= 15 THEN 0 ELSE 1 END"
	}
	rows, err := r.db.Query(ctx, eventSelect+` WHERE e.id <> $2 AND e.sport_id = $3 AND e.status='published'
AND COALESCE(e.ends_at, e.starts_at) > now() AND e.organizer_id <> $1
AND (e.capacity IS NULL OR (SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed') < e.capacity)
ORDER BY `+near+`, abs(extract(epoch FROM e.starts_at - $4::timestamptz)), e.id LIMIT $5`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		e, err := scanEvent(rows, a)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ICS — событие в формате iCalendar (RFC 5545) для «Добавить в календарь». Без длительности —
// полтора часа: календарю нужен конец, а большинство встреч укладываются в это время.
func ICS(e Event, now time.Time) string {
	end := e.StartsAt.Add(90 * time.Minute)
	if e.EndsAt != nil {
		end = *e.EndsAt
	}
	desc := e.Description
	if e.Requirements != "" {
		desc += "\n\nВозьмите: " + e.Requirements
	}
	if e.PrivateInfo != "" {
		desc += "\n\nТочка сбора: " + e.PrivateInfo
	}
	desc += "\n\nОрганизатор: " + e.Organizer.Name
	lines := []string{
		"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Sport Together//RU", "CALSCALE:GREGORIAN", "METHOD:PUBLISH",
		"BEGIN:VEVENT",
		"UID:" + e.ID + "@sport-together",
		"DTSTAMP:" + icsTime(now),
		"DTSTART:" + icsTime(e.StartsAt),
		"DTEND:" + icsTime(end),
		"SUMMARY:" + icsText(e.Title),
		"LOCATION:" + icsText(e.Location),
		"DESCRIPTION:" + icsText(strings.TrimSpace(desc)),
	}
	if e.Lat != nil {
		lines = append(lines, fmt.Sprintf("GEO:%.6f;%.6f", *e.Lat, *e.Lng))
	}
	lines = append(lines,
		// Напоминание за два часа — то же, что шлёт платформа.
		"BEGIN:VALARM", "ACTION:DISPLAY", "DESCRIPTION:"+icsText(e.Title), "TRIGGER:-PT2H", "END:VALARM",
		"END:VEVENT", "END:VCALENDAR")
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(fold(l))
		b.WriteString("\r\n")
	}
	return b.String()
}

func icsTime(t time.Time) string { return t.UTC().Format("20060102T150405Z") }

// icsText экранирует спецсимволы текста iCalendar.
func icsText(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`).Replace(s)
}

// fold переносит строки длиннее 75 байт (RFC 5545, 3.1), не разрезая символы UTF-8.
func fold(line string) string {
	var b strings.Builder
	n := 0
	for _, r := range line {
		size := len(string(r))
		if n+size > 75 {
			b.WriteString("\r\n ")
			n = 1
		}
		b.WriteRune(r)
		n += size
	}
	return b.String()
}
