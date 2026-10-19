package events

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Подбор запускается при открытии списка. Advisory lock исключает разные предложения
// на одни запросы при одновременном обращении нескольких реплик.
func (r *Repository) DemandProposals(ctx context.Context, a Actor) ([]DemandProposal, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(733182945)"); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT r.id,r.user_id,r.sport_id,r.level,r.area,r.lat,r.lng,r.radius_km,r.day_parts,r.weekdays::int[],r.expires_at
FROM company_requests r JOIN sports s ON s.id=r.sport_id
WHERE r.status='active' AND r.expires_at>now()+interval '2 hours' AND NOT s.hidden
AND NOT EXISTS(SELECT 1 FROM demand_proposal_members m JOIN demand_proposals p ON p.id=m.proposal_id
 WHERE m.request_id=r.id AND (p.expires_at>now() OR p.status='claimed'))
ORDER BY r.created_at,r.id LIMIT 1000`)
		if err != nil {
			return err
		}
		candidates := []demandCandidate{}
		for rows.Next() {
			var c demandCandidate
			if err = rows.Scan(&c.ID, &c.UserID, &c.SportID, &c.Level, &c.Area, &c.Lat, &c.Lng, &c.Radius, &c.Parts, &c.Days, &c.ExpiresAt); err != nil {
				rows.Close()
				return err
			}
			candidates = append(candidates, c)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		used := map[string]bool{}
		now := time.Now()
		for _, anchor := range candidates {
			if used[anchor.ID] || (len(strings.TrimSpace(anchor.Area)) < 3 && anchor.Lat == nil) {
				continue
			}
			group := []demandCandidate{anchor}
			level := anchor.Level
			for _, c := range candidates {
				if c.ID == anchor.ID || used[c.ID] || c.SportID != anchor.SportID || c.UserID == anchor.UserID || !demandPlaceFits(anchor, c) {
					continue
				}
				if level != "any" && c.Level != "any" && level != c.Level {
					continue
				}
				trial := append(append([]demandCandidate{}, group...), c)
				if _, ok := demandSlot(now, trial); !ok {
					continue
				}
				group = trial
				if level == "any" {
					level = c.Level
				}
				if len(group) == 20 {
					break
				}
			}
			if len(group) < 3 {
				continue
			}
			slot, ok := demandSlot(now, group)
			if !ok {
				continue
			}
			expiry := now.Add(48 * time.Hour)
			if slot.Add(-time.Hour).Before(expiry) {
				expiry = slot.Add(-time.Hour)
			}
			for _, c := range group {
				if c.ExpiresAt.Before(expiry) {
					expiry = c.ExpiresAt
				}
			}
			id := uuid.NewString()
			area := anchor.Area
			if strings.TrimSpace(area) == "" {
				area = "Район выбранной точки"
			}
			_, err = tx.Exec(ctx, `INSERT INTO demand_proposals(id,sport_id,level,starts_at,area,lat,lng,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, id, anchor.SportID, level, slot, area, anchor.Lat, anchor.Lng, expiry)
			if err != nil {
				return err
			}
			for _, c := range group {
				if _, err = tx.Exec(ctx, `INSERT INTO demand_proposal_members(proposal_id,request_id,user_id)VALUES($1,$2,$3)`, id, c.ID, c.UserID); err != nil {
					return err
				}
				if err = notify(ctx, tx, c.UserID, "Компания уже собралась", fmt.Sprintf("Вас %d. %s, район: %s. Кто готов провести встречу? Точную точку сбора подтвердит организатор.", len(group), humanTime(slot), area), "/company-proposals/"+id); err != nil {
					return err
				}
				used[c.ID] = true
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `SELECT p.id,p.sport_id,p.level,p.starts_at,p.area,(p.lat IS NOT NULL),
CASE WHEN p.status='pending' AND p.expires_at<=now() THEN 'expired' ELSE p.status END,p.event_id,p.expires_at,
(SELECT count(*) FROM demand_proposal_members n JOIN company_requests r ON r.id=n.request_id WHERE n.proposal_id=p.id AND NOT n.declined AND r.status='active' AND r.expires_at>now()),m.declined
FROM demand_proposals p JOIN demand_proposal_members m ON m.proposal_id=p.id WHERE m.user_id=$1 ORDER BY p.created_at DESC LIMIT 50`, a.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DemandProposal{}
	for rows.Next() {
		var p DemandProposal
		if err = rows.Scan(&p.ID, &p.SportID, &p.Level, &p.StartsAt, &p.Area, &p.RequiresCoordinates, &p.Status, &p.EventID, &p.ExpiresAt, &p.MembersCount, &p.Declined); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) ClaimDemandProposal(ctx context.Context, a Actor, id string, in ProposalClaim) (Event, error) {
	in.Location = strings.TrimSpace(in.Location)
	if !between(in.Location, 3, 300) {
		return Event{}, FieldError{"location", "Укажите точную точку сбора: от 3 до 300 символов"}
	}
	if (in.Lat == nil) != (in.Lng == nil) || (in.Lat != nil && (math.IsNaN(*in.Lat) || math.IsInf(*in.Lat, 0) || math.IsNaN(*in.Lng) || math.IsInf(*in.Lng, 0) || *in.Lat < -90 || *in.Lat > 90 || *in.Lng < -180 || *in.Lng > 180)) {
		return Event{}, FieldError{"location", "Некорректные координаты точки сбора"}
	}
	eventID := ""
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var sport, level, area, status string
		var starts, expires time.Time
		var lat, lng *float64
		var existing *string
		err := tx.QueryRow(ctx, `SELECT p.sport_id,p.level,p.area,p.status,p.starts_at,p.expires_at,p.lat,p.lng,p.event_id FROM demand_proposals p
WHERE p.id=$1 AND EXISTS(SELECT 1 FROM demand_proposal_members m WHERE m.proposal_id=p.id AND m.user_id=$2 AND NOT m.declined) FOR UPDATE`, id, a.ID).Scan(&sport, &level, &area, &status, &starts, &expires, &lat, &lng, &existing)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "claimed" {
			return fmt.Errorf("%w: организатор уже найден", platform.ErrTransition)
		}
		if !expires.After(time.Now()) {
			return fmt.Errorf("%w: предложение истекло", platform.ErrTransition)
		}
		// Закрытие запроса не должно пересечься с публикацией встречи.
		locked, err := tx.Query(ctx, `SELECT r.id,r.lat,r.lng,r.radius_km FROM company_requests r JOIN demand_proposal_members m ON m.request_id=r.id WHERE m.proposal_id=$1 AND NOT m.declined AND r.status='active' AND r.expires_at>now() ORDER BY r.id FOR UPDATE OF r`, id)
		if err != nil {
			return err
		}
		fits := true
		for locked.Next() {
			var rid string
			var rlat, rlng *float64
			var radius *int
			if err = locked.Scan(&rid, &rlat, &rlng, &radius); err != nil {
				locked.Close()
				return err
			}
			if rlat != nil && radius != nil && in.Lat != nil && demandDistance(demandCandidate{Lat: in.Lat, Lng: in.Lng}, demandCandidate{Lat: rlat, Lng: rlng}) > float64(*radius) {
				fits = false
			}
		}
		locked.Close()
		if err = locked.Err(); err != nil {
			return err
		}
		if lat != nil && in.Lat == nil {
			return FieldError{"location", "Подтвердите координаты точной точки сбора"}
		}
		if !fits {
			return FieldError{"location", "Точка сбора выходит за выбранный участниками радиус"}
		}
		var n int
		var self bool
		err = tx.QueryRow(ctx, `SELECT count(*),COALESCE(bool_or(m.user_id=$2),false) FROM demand_proposal_members m JOIN company_requests r ON r.id=m.request_id
WHERE m.proposal_id=$1 AND NOT m.declined AND r.status='active' AND r.expires_at>now()`, id, a.ID).Scan(&n, &self)
		if err != nil {
			return err
		}
		if n < 3 || !self {
			return fmt.Errorf("%w: для встречи нужны три активных запроса", platform.ErrTransition)
		}
		var hidden bool
		if err = tx.QueryRow(ctx, "SELECT hidden FROM sports WHERE id=$1 FOR SHARE", sport).Scan(&hidden); err != nil {
			return err
		}
		if hidden {
			return ErrSportHidden
		}
		eventID = uuid.NewString()
		_, err = tx.Exec(ctx, `INSERT INTO events(id,organizer_id,sport_id,title,description,starts_at,ends_at,location,lat,lng,level,join_mode)
VALUES($1,$2,$3,'Встреча по запросам участников',$4,$5,$5::timestamptz+interval '1 hour',$6,$7,$8,$9,'open')`, eventID, a.ID, sport, "Компания найдена платформой. Район: "+area+". Участники подтверждают запись самостоятельно.", starts, in.Location, in.Lat, in.Lng, level)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `UPDATE demand_proposals SET status='claimed',event_id=$2 WHERE id=$1`, id, eventID); err != nil {
			return err
		}
		// Участники предложения уже получают отдельное приглашение; обычный подбор не дублирует его.
		if _, err = tx.Exec(ctx, `INSERT INTO company_matches(request_id,event_id,notified)
SELECT m.request_id,$2,true FROM demand_proposal_members m JOIN company_requests r ON r.id=m.request_id
WHERE m.proposal_id=$1 AND NOT m.declined AND m.user_id<>$3 AND r.status='active' AND r.expires_at>now()
ON CONFLICT DO NOTHING`, id, eventID, a.ID); err != nil {
			return err
		}
		if err = record(ctx, tx, "event", eventID, platform.Transition{From: "none", To: "published", Actor: actorLabel(a)}); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT m.user_id FROM demand_proposal_members m JOIN company_requests r ON r.id=m.request_id WHERE m.proposal_id=$1 AND NOT m.declined AND m.user_id<>$2 AND r.status='active' AND r.expires_at>now()`, id, a.ID)
		if err != nil {
			return err
		}
		users := []string{}
		for rows.Next() {
			var u string
			if err = rows.Scan(&u); err != nil {
				rows.Close()
				return err
			}
			users = append(users, u)
		}
		rows.Close()
		if err = rows.Err(); err != nil {
			return err
		}
		for _, u := range users {
			if err = notify(ctx, tx, u, "Организатор найден", "Встреча опубликована. Откройте условия и запишитесь, если готовы прийти.", "/events/"+eventID); err != nil {
				return err
			}
		}
		return matchNewEvent(ctx, tx, eventID)
	})
	if err != nil {
		return Event{}, err
	}
	return r.GetEvent(ctx, a, eventID)
}

func (r *Repository) DeclineDemandProposal(ctx context.Context, a Actor, id string) error {
	return r.inTx(ctx, func(tx pgx.Tx) error {
		var p string
		err := tx.QueryRow(ctx, `SELECT id FROM demand_proposals WHERE id=$1 AND status='pending' AND expires_at>now() FOR UPDATE`, id).Scan(&p)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE demand_proposal_members SET declined=true WHERE proposal_id=$1 AND user_id=$2`, id, a.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		return nil
	})
}
