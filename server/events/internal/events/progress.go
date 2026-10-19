package events

import (
	"context"
	"github.com/gin-gonic/gin"
)

type Achievement struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Current     int    `json:"current"`
	Target      int    `json:"target"`
	Points      int    `json:"points"`
	Earned      bool   `json:"earned"`
}
type PersonalProgress struct {
	Points       int           `json:"points"`
	Title        string        `json:"title"`
	NextTitleAt  int           `json:"next_title_at"`
	Achievements []Achievement `json:"achievements"`
	Note         string        `json:"note"`
}
type progressStore interface {
	PersonalProgress(context.Context, Actor) (PersonalProgress, error)
}

func RegisterProgress(g *gin.RouterGroup, s *Service) {
	g.GET("/my/progress", func(c *gin.Context) {
		out, err := s.Store.(progressStore).PersonalProgress(c.Request.Context(), actor(c))
		if !handle(c, err) {
			c.JSON(200, out)
		}
	})
}

// Прогресс считается из действующих данных. Повторный HTTP-запрос не начисляет очки,
// а запись на встречу не выдаётся за доказанное посещение.
func (r *Repository) PersonalProgress(ctx context.Context, a Actor) (PersonalProgress, error) {
	out := PersonalProgress{Title: "Первый шаг", NextTitleAt: 100, Achievements: []Achievement{}, Note: "Очки отражают действия на платформе. Посещение встреч не проверяется; очки нельзя купить, передать или обменять на деньги."}
	var interests, company, question, hosted, joined, buddy int
	err := r.db.QueryRow(ctx, `SELECT
 (SELECT count(*) FROM recommendation_preferences WHERE user_id=$1 AND cardinality(sport_ids)>0),
 LEAST((SELECT count(*) FROM company_requests WHERE user_id=$1),1),
 LEAST((SELECT count(*) FROM comments WHERE author_id=$1 AND status='visible'),1),
 LEAST((SELECT count(*) FROM events e WHERE e.organizer_id=$1 AND e.status='published' AND COALESCE(e.ends_at,e.starts_at+interval '90 minutes')<now() AND (SELECT count(*) FROM participations p WHERE p.event_id=e.id AND p.status='confirmed')>=2),5),
 LEAST((SELECT count(DISTINCT (e.starts_at AT TIME ZONE 'Asia/Krasnoyarsk')::date) FROM participations p JOIN events e ON e.id=p.event_id WHERE p.user_id=$1 AND p.status='confirmed' AND e.status='published' AND COALESCE(e.ends_at,e.starts_at+interval '90 minutes')<now()),5),
 LEAST((SELECT count(DISTINCT p.event_id) FROM participations p JOIN events e ON e.id=p.event_id WHERE p.user_id=$1 AND p.status='confirmed' AND e.status='published' AND COALESCE(e.ends_at,e.starts_at+interval '90 minutes')<now() AND EXISTS(SELECT 1 FROM participations n WHERE n.buddy_id=p.id AND n.status='confirmed')),5)`, a.ID).Scan(&interests, &company, &question, &hosted, &joined, &buddy)
	if err != nil {
		return out, err
	}
	out.Achievements = []Achievement{
		{Code: "interests", Title: "Свой маршрут", Description: "Выберите интересующие направления", Current: interests, Target: 1, Points: 10, Earned: interests > 0},
		{Code: "company", Title: "Первый шаг навстречу", Description: "Оставьте запрос компании", Current: company, Target: 1, Points: 10, Earned: company > 0},
		{Code: "question", Title: "Начали разговор", Description: "Задайте вопрос или ответьте в обсуждении", Current: question, Target: 1, Points: 10, Earned: question > 0},
		{Code: "organizer", Title: "Собираете людей", Description: "Проведите пять завершившихся встреч с минимум двумя подтверждёнными участниками", Current: hosted, Target: 5, Points: 125, Earned: hosted >= 5},
		{Code: "community", Title: "В ритме компании", Description: "Будьте записаны на завершившиеся встречи в пять разных дней", Current: joined, Target: 5, Points: 75, Earned: joined >= 5},
		{Code: "buddy", Title: "Помогаете начать", Description: "Станьте проводником новичков на пяти завершившихся встречах", Current: buddy, Target: 5, Points: 100, Earned: buddy >= 5},
	}
	out.Points = interests*10 + company*10 + question*10 + hosted*25 + joined*15 + buddy*20
	if out.Points >= 100 {
		out.Title = "Собираете компанию"
		out.NextTitleAt = 200
	}
	if out.Points >= 200 {
		out.Title = "Опора сообщества"
		out.NextTitleAt = 330
	}
	if out.Points >= 330 {
		out.Title = "Вместе получается"
		out.NextTitleAt = 0
	}
	return out, nil
}
