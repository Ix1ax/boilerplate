package events

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RecommendationPreferences struct {
	SportIDs []string `json:"sport_ids"`
	Level    string   `json:"level"`
	City     string   `json:"city"`
	Weekdays []int    `json:"weekdays"`
	DayParts []string `json:"day_parts"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
	RadiusKm int      `json:"radius_km"`
}

func validatePreferences(p *RecommendationPreferences) error {
	p.SportIDs = uniqueSorted(p.SportIDs)
	p.Weekdays = uniqueSorted(p.Weekdays)
	p.DayParts = uniqueSorted(p.DayParts)
	if p.Level == "" {
		p.Level = "any"
	}
	if p.RadiusKm == 0 {
		p.RadiusKm = 20
	}
	p.City = strings.TrimSpace(p.City)
	if len([]rune(p.City)) > 100 {
		return FieldError{"city", "Город — не длиннее 100 символов"}
	}
	if len(p.SportIDs) > 30 || !levels[p.Level] || p.RadiusKm < 1 || p.RadiusKm > 100 {
		return FieldError{"preferences", "Проверьте интересы, уровень и радиус от 1 до 100 км"}
	}
	for _, id := range p.SportIDs {
		if uuid.Validate(id) != nil {
			return FieldError{"sport_ids", "Неизвестное направление"}
		}
	}
	for _, d := range p.Weekdays {
		if d < 1 || d > 7 {
			return FieldError{"weekdays", "Дни недели — от 1 до 7"}
		}
	}
	for _, d := range p.DayParts {
		if !dayParts[d] {
			return FieldError{"day_parts", "Время — morning, day или evening"}
		}
	}
	if (p.Lat == nil) != (p.Lng == nil) || p.Lat != nil && (!validLat(*p.Lat) || !validLng(*p.Lng) || math.IsNaN(*p.Lat) || math.IsNaN(*p.Lng)) {
		return FieldError{"location", "Некорректные координаты"}
	}
	return nil
}

type RecommendedEvent struct {
	Event   Event    `json:"event"`
	Reasons []string `json:"reasons"`
}
type RecommendationPage struct {
	Items        []RecommendedEvent `json:"items"`
	Personalized bool               `json:"personalized"`
	Algorithm    string             `json:"algorithm"`
	GeneratedAt  time.Time          `json:"generated_at"`
	HasMore      bool               `json:"has_more"`
	NextOffset   int                `json:"next_offset"`
}

type recommendationStore interface {
	RecommendationPreferences(context.Context, Actor) (RecommendationPreferences, error)
	SaveRecommendationPreferences(context.Context, Actor, RecommendationPreferences) (RecommendationPreferences, error)
	RecommendationFeedback(context.Context, Actor, string, string) error
	RecommendEvents(context.Context, Actor, int, int) (RecommendationPage, error)
}

// RegisterRecommendations использует ту же личность и синхронизацию профиля, что остальные ручки events.
func RegisterRecommendations(g *gin.RouterGroup, s *Service) {
	store := func() recommendationStore { return s.Store.(recommendationStore) }
	g.GET("/my/recommendation-preferences", func(c *gin.Context) {
		p, err := store().RecommendationPreferences(c.Request.Context(), actor(c))
		if !handle(c, err) {
			c.JSON(200, p)
		}
	})
	g.PUT("/my/recommendation-preferences", func(c *gin.Context) {
		var p RecommendationPreferences
		if c.ShouldBindJSON(&p) != nil {
			fail(c, 400, "invalid_body", "Проверьте формат предпочтений")
			return
		}
		if handle(c, validatePreferences(&p)) {
			return
		}
		out, err := store().SaveRecommendationPreferences(c.Request.Context(), actor(c), p)
		if !handle(c, err) {
			c.JSON(200, out)
		}
	})
	g.GET("/recommendations/events", func(c *gin.Context) {
		limit, offset, ok := recommendationPagination(c)
		if !ok {
			fail(c, 400, "invalid_query", "limit — от 1 до 50, offset — от 0 до 1000")
			return
		}
		out, err := store().RecommendEvents(c.Request.Context(), actor(c), limit, offset)
		if !handle(c, err) {
			c.JSON(200, out)
		}
	})
	g.POST("/events/:id/recommendation-feedback", withID(func(c *gin.Context, id string) {
		var in struct {
			Action string `json:"action"`
		}
		if c.ShouldBindJSON(&in) != nil {
			fail(c, 400, "invalid_body", "Укажите действие")
			return
		}
		if in.Action != "view" && in.Action != "interested" && in.Action != "dismiss" && in.Action != "reset" {
			handle(c, FieldError{"action", "Действие: view, interested, dismiss или reset"})
			return
		}
		if !handle(c, store().RecommendationFeedback(c.Request.Context(), actor(c), id, in.Action)) {
			c.Status(204)
		}
	}))
}

func (r *Repository) RecommendationPreferences(ctx context.Context, a Actor) (RecommendationPreferences, error) {
	p := RecommendationPreferences{SportIDs: []string{}, Weekdays: []int{}, DayParts: []string{}, Level: "any", RadiusKm: 20}
	err := r.db.QueryRow(ctx, `SELECT sport_ids::text[],level,city,weekdays,day_parts,lat,lng,radius_km FROM recommendation_preferences WHERE user_id=$1`, a.ID).Scan(&p.SportIDs, &p.Level, &p.City, &p.Weekdays, &p.DayParts, &p.Lat, &p.Lng, &p.RadiusKm)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.db.QueryRow(ctx, `SELECT COALESCE(level,'any'),city FROM profiles WHERE user_id=$1`, a.ID).Scan(&p.Level, &p.City)
		if errors.Is(err, pgx.ErrNoRows) {
			err = nil
		}
	}
	return p, err
}

func (r *Repository) SaveRecommendationPreferences(ctx context.Context, a Actor, p RecommendationPreferences) (RecommendationPreferences, error) {
	if err := validatePreferences(&p); err != nil {
		return p, err
	}
	var n int
	if err := r.db.QueryRow(ctx, `SELECT count(*) FROM sports WHERE id=ANY($1::uuid[]) AND NOT hidden`, p.SportIDs).Scan(&n); err != nil {
		return p, err
	}
	if n != len(p.SportIDs) {
		return p, FieldError{"sport_ids", "Направление скрыто или не найдено"}
	}
	_, err := r.db.Exec(ctx, `INSERT INTO recommendation_preferences(user_id,sport_ids,level,city,weekdays,day_parts,lat,lng,radius_km) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)
 ON CONFLICT(user_id) DO UPDATE SET sport_ids=EXCLUDED.sport_ids,level=EXCLUDED.level,city=EXCLUDED.city,weekdays=EXCLUDED.weekdays,day_parts=EXCLUDED.day_parts,lat=EXCLUDED.lat,lng=EXCLUDED.lng,radius_km=EXCLUDED.radius_km,updated_at=now()`, a.ID, p.SportIDs, p.Level, p.City, p.Weekdays, p.DayParts, p.Lat, p.Lng, p.RadiusKm)
	return p, err
}

func (r *Repository) RecommendationFeedback(ctx context.Context, a Actor, id, action string) error {
	if _, err := r.GetEvent(ctx, a, id); err != nil {
		return err
	}
	if action == "reset" {
		_, err := r.db.Exec(ctx, `DELETE FROM recommendation_feedback WHERE user_id=$1 AND event_id=$2`, a.ID, id)
		return err
	}
	// Просмотр не отменяет явный отказ и не снимает сохранённый интерес.
	_, err := r.db.Exec(ctx, `INSERT INTO recommendation_feedback(user_id,event_id,action) VALUES($1,$2,$3) ON CONFLICT(user_id,event_id) DO UPDATE SET action=EXCLUDED.action,updated_at=now() WHERE EXCLUDED.action<>'view' OR recommendation_feedback.action='view'`, a.ID, id, action)
	return err
}

type scoredRecommendation struct {
	item  RecommendedEvent
	score float64
}

func dayPartAt(t time.Time) string {
	h := t.In(localZone).Hour()
	if h < 12 {
		return "morning"
	}
	if h < 17 {
		return "day"
	}
	return "evening"
}
func containsValue[T comparable](values []T, v T) bool {
	for _, x := range values {
		if x == v {
			return true
		}
	}
	return false
}

func rankRecommendation(e Event, p RecommendationPreferences, affinity map[string]float64, interested map[string]bool, now time.Time) (scoredRecommendation, bool) {
	out := scoredRecommendation{item: RecommendedEvent{Event: e, Reasons: []string{}}, score: 5}
	if p.Level != "any" && e.Level != "any" && p.Level != e.Level {
		return out, false
	}
	if p.City != "" && strings.Contains(strings.ToLower(e.Location), strings.ToLower(p.City)) {
		out.score += 15
		out.item.Reasons = append(out.item.Reasons, "Встреча в указанном вами городе")
	}
	if e.Format == "online" {
		out.item.Reasons = append(out.item.Reasons, "Онлайн — можно присоединиться из дома")
	} else if p.Lat != nil && e.Lat != nil {
		d := DistanceKm(*p.Lat, *p.Lng, *e.Lat, *e.Lng)
		if d > float64(p.RadiusKm) {
			return out, false
		}
		out.item.Event.DistanceKm = &d
		out.score += 25 * (1 - d/float64(p.RadiusKm))
		out.item.Reasons = append(out.item.Reasons, "В пределах выбранного радиуса")
	}
	if containsValue(p.SportIDs, e.Sport.ID) {
		out.score += 65
		out.item.Reasons = append(out.item.Reasons, "Вы выбрали это направление")
	} else if w := affinity[e.Sport.ID]; w > 0 {
		out.score += math.Min(w, 35)
		out.item.Reasons = append(out.item.Reasons, "По вашим записям и запросам компании")
	}
	local := e.StartsAt.In(localZone)
	day := int(local.Weekday())
	if day == 0 {
		day = 7
	}
	if len(p.Weekdays) > 0 {
		if containsValue(p.Weekdays, day) {
			out.score += 14
			out.item.Reasons = append(out.item.Reasons, "Подходит выбранный день недели")
		} else {
			out.score -= 15
		}
	}
	if len(p.DayParts) > 0 {
		if containsValue(p.DayParts, dayPartAt(local)) {
			out.score += 14
			out.item.Reasons = append(out.item.Reasons, "Подходит выбранное время суток")
		} else {
			out.score -= 15
		}
	}
	if p.Level != "any" {
		out.score += 8
		out.item.Reasons = append(out.item.Reasons, "Подходит вашему уровню")
	}
	if e.SpotsLeft == nil || *e.SpotsLeft > 0 {
		out.score += 8
		out.item.Reasons = append(out.item.Reasons, "Есть места для записи")
	} else {
		out.score -= 10
		if e.JoinMode == "open" {
			out.item.Reasons = append(out.item.Reasons, "Можно встать в лист ожидания")
		} else {
			out.item.Reasons = append(out.item.Reasons, "Мест пока нет, решение принимает организатор")
		}
	}
	if interested[e.ID] {
		out.score += 20
		out.item.Reasons = append(out.item.Reasons, "Вы отметили встречу как интересную")
	}
	out.score += 10 / (1 + e.StartsAt.Sub(now).Hours()/24)
	if len(out.item.Reasons) == 0 {
		out.item.Reasons = append(out.item.Reasons, "Предстоящая встреча для знакомства с платформой")
	}
	return out, true
}

func (r *Repository) RecommendEvents(ctx context.Context, a Actor, limit, offset int) (RecommendationPage, error) {
	now := time.Now()
	out := RecommendationPage{Items: []RecommendedEvent{}, Algorithm: "explainable-v1", GeneratedAt: now}
	if limit < 1 || limit > 50 || offset < 0 || offset > 1000 {
		return out, FieldError{"pagination", "Некорректная страница"}
	}
	p, err := r.RecommendationPreferences(ctx, a)
	if err != nil {
		return out, err
	}
	affinity := map[string]float64{}
	rows, err := r.db.Query(ctx, `SELECT sport_id::text,sum(weight) FROM (
 SELECT e.sport_id,LEAST(count(*)*8,24)::float8 AS weight FROM participations p JOIN events e ON e.id=p.event_id WHERE p.user_id=$1 AND p.status IN ('confirmed','pending') AND e.starts_at>now()-interval '180 days' GROUP BY e.sport_id
 UNION ALL SELECT sport_id,30::float8 FROM company_requests WHERE user_id=$1 AND status='active' AND expires_at>now()
 UNION ALL SELECT e.sport_id,LEAST(count(*)*3,9)::float8 FROM recommendation_feedback f JOIN events e ON e.id=f.event_id WHERE f.user_id=$1 AND f.action='interested' AND f.updated_at>now()-interval '90 days' GROUP BY e.sport_id) x GROUP BY sport_id`, a.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id string
		var w float64
		if err = rows.Scan(&id, &w); err != nil {
			rows.Close()
			return out, err
		}
		affinity[id] = w
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	interested := map[string]bool{}
	rows, err = r.db.Query(ctx, `SELECT event_id::text FROM recommendation_feedback WHERE user_id=$1 AND action='interested'`, a.ID)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return out, err
		}
		interested[id] = true
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	out.Personalized = len(p.SportIDs) > 0 || len(affinity) > 0 || p.Lat != nil || p.City != "" || p.Level != "any" || len(p.Weekdays) > 0 || len(p.DayParts) > 0
	// Ограничиваем горизонт и размер выборки: подбор не превращается в полный просмотр базы.
	rows, err = r.db.Query(ctx, eventSelect+` WHERE e.status='published' AND NOT s.hidden AND e.starts_at>now() AND e.starts_at<=now()+interval '60 days' AND e.organizer_id<>$1
 AND (mp.id IS NULL OR mp.status='cancelled') AND NOT EXISTS(SELECT 1 FROM recommendation_feedback f WHERE f.user_id=$1 AND f.event_id=e.id AND f.action='dismiss')
 AND ($2::float8 IS NULL OR e.format='online' OR (e.lat IS NOT NULL AND `+distanceSQL("$2", "$3")+` <= $4))
 ORDER BY e.starts_at,e.id LIMIT 1000`, a.ID, p.Lat, p.Lng, p.RadiusKm)
	if err != nil {
		return out, err
	}
	ranked := []scoredRecommendation{}
	for rows.Next() {
		e, err := scanEvent(rows, a)
		if err != nil {
			rows.Close()
			return out, err
		}
		if candidate, ok := rankRecommendation(e, p, affinity, interested, now); ok {
			ranked = append(ranked, candidate)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		if !ranked[i].item.Event.StartsAt.Equal(ranked[j].item.Event.StartsAt) {
			return ranked[i].item.Event.StartsAt.Before(ranked[j].item.Event.StartsAt)
		}
		return ranked[i].item.Event.ID < ranked[j].item.Event.ID
	})
	// В первых карточках не даём одному организатору занять всю выдачу; остальные встречи не пропадают.
	ordered := make([]scoredRecommendation, 0, len(ranked))
	deferred := []scoredRecommendation{}
	organizers := map[string]int{}
	for _, v := range ranked {
		id := v.item.Event.Organizer.Type + ":" + v.item.Event.Organizer.ID
		if len(ordered) < 10 && organizers[id] >= 2 {
			deferred = append(deferred, v)
			continue
		}
		organizers[id]++
		ordered = append(ordered, v)
	}
	ordered = append(ordered, deferred...)
	start := min(offset, len(ordered))
	end := min(start+limit, len(ordered))
	for _, v := range ordered[start:end] {
		out.Items = append(out.Items, v.item)
	}
	out.HasMore = end < len(ordered)
	out.NextOffset = end
	return out, nil
}
