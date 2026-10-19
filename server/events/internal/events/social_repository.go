package events

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func socialJSON(row pgx.Row) (any, error) {
	var b json.RawMessage
	err := row.Scan(&b)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return b, err
}
func socialRows(rows pgx.Rows, p SocialPage) (SocialResult, error) {
	defer rows.Close()
	out := SocialResult{Items: []json.RawMessage{}}
	for rows.Next() {
		var b json.RawMessage
		if err := rows.Scan(&b); err != nil {
			return out, err
		}
		out.Items = append(out.Items, b)
	}
	if err := rows.Err(); err != nil {
		return out, err
	}
	if len(out.Items) > p.Limit {
		out.Items = out.Items[:p.Limit]
		var last struct {
			CreatedAt time.Time `json:"created_at"`
			ID        string    `json:"id"`
		}
		if err := json.Unmarshal(out.Items[len(out.Items)-1], &last); err != nil {
			return out, err
		}
		b, _ := json.Marshal(socialCursor{At: last.CreatedAt, ID: last.ID})
		out.NextCursor = base64.RawURLEncoding.EncodeToString(b)
	}
	return out, nil
}

func socialSignal(ctx context.Context, tx pgx.Tx, userID, kind, id string) error {
	// В личный канал отправляется только сигнал. Историю всегда выдаёт API с проверкой участника.
	b, _ := json.Marshal(map[string]any{"channel": "user:" + userID, "from": "events", "data": map[string]string{"type": kind, "id": id}, "at": time.Now().UTC()})
	_, err := tx.Exec(ctx, `INSERT INTO outbox(id,type,user_id,request_id,payload) VALUES($1,'realtime.publish',$2,$3,$4)`, uuid.NewString(), userID, platform.RequestID(ctx), b)
	return err
}
func orderedPeers(a, b string) (string, string) {
	if a > b {
		return b, a
	}
	return a, b
}

func (r *Repository) Friendship(ctx context.Context, a Actor, peer, action string) (any, error) {
	if err := socialPeer(a, peer); err != nil {
		return nil, err
	}
	lo, hi := orderedPeers(a.ID, peer)
	var out any
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		// Одна строка на пару и одна блокировка для всех переходов: встречные заявки не принимают друг друга молча.
		if action == "request" {
			tag, err := tx.Exec(ctx, `INSERT INTO friendships(user_a,user_b,requested_by,status)
SELECT $1,$2,$3,'pending' WHERE EXISTS(SELECT 1 FROM profiles WHERE user_id=$4)
ON CONFLICT(user_a,user_b) DO UPDATE SET requested_by=EXCLUDED.requested_by,status='pending',updated_at=now()
WHERE friendships.status='declined'`, lo, hi, a.ID, peer)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return ErrSocialConflict
			}
		} else {
			query := `UPDATE friendships SET status=$4,updated_at=now() WHERE user_a=$1 AND user_b=$2 AND requested_by<>$3 AND status='pending'`
			status := "accepted"
			if action == "decline" {
				status = "declined"
			}
			if action == "remove" {
				query = `UPDATE friendships SET status=$4,updated_at=now() WHERE user_a=$1 AND user_b=$2 AND (status='accepted' OR (status='pending' AND requested_by=$3))`
				status = "declined"
			}
			if action != "accept" && action != "decline" && action != "remove" {
				return ErrNotFound
			}
			tag, err := tx.Exec(ctx, query, lo, hi, a.ID, status)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				return ErrSocialConflict
			}
		}
		var err error
		out, err = socialJSON(tx.QueryRow(ctx, `SELECT jsonb_build_object('user_id',$3::text,'requested_by',requested_by,'status',status,'created_at',created_at) FROM friendships WHERE user_a=$1 AND user_b=$2`, lo, hi, peer))
		if err != nil {
			return err
		}
		if err = socialSignal(ctx, tx, peer, "friendship.changed", a.ID); err != nil {
			return err
		}
		return socialSignal(ctx, tx, a.ID, "friendship.changed", peer)
	})
	return out, err
}
func (r *Repository) Friends(ctx context.Context, a Actor, p SocialPage) (SocialResult, error) {
	rows, err := r.db.Query(ctx, `SELECT to_jsonb(x) FROM (SELECT pr.user_id AS id,pr.display_name,f.status,f.requested_by,f.created_at
FROM friendships f JOIN profiles pr ON pr.user_id=CASE WHEN f.user_a=$1 THEN f.user_b ELSE f.user_a END
WHERE $1 IN(f.user_a,f.user_b) AND f.status IN('pending','accepted') AND (f.created_at,pr.user_id)<($2,$3)
ORDER BY f.created_at DESC,pr.user_id DESC LIMIT $4) x`, a.ID, p.Before, p.ID, p.Limit+1)
	if err != nil {
		return SocialResult{}, err
	}
	return socialRows(rows, p)
}
func (r *Repository) OpenDirectChat(ctx context.Context, a Actor, peer string) (any, error) {
	if err := socialPeer(a, peer); err != nil {
		return nil, err
	}
	lo, hi := orderedPeers(a.ID, peer)
	var out any
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		var yes bool
		// Дружбу держим до конца записи: удаление друга не может обогнать создание чата.
		if err := tx.QueryRow(ctx, `SELECT true FROM friendships WHERE user_a=$1 AND user_b=$2 AND status='accepted' FOR SHARE`, lo, hi).Scan(&yes); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSocialConflict
			}
			return err
		}
		var err error
		out, err = socialJSON(tx.QueryRow(ctx, `WITH c AS (INSERT INTO direct_chats(id,user_a,user_b) VALUES($1,$2,$3)
ON CONFLICT(user_a,user_b) DO UPDATE SET user_a=EXCLUDED.user_a RETURNING *) SELECT jsonb_build_object('id',id,'kind','direct','peer_id',$4::text,'created_at',created_at) FROM c`, uuid.NewString(), lo, hi, peer))
		return err
	})
	return out, err
}
func (r *Repository) Chats(ctx context.Context, a Actor, p SocialPage) (SocialResult, error) {
	// У мероприятия нет второй истории: чат — это его comments, доступный сразу после создания события.
	rows, err := r.db.Query(ctx, `SELECT to_jsonb(x) FROM (
SELECT c.id,'direct'::text AS kind,pr.display_name AS title,pr.user_id AS peer_id,NULL::text AS event_id,
'/api/v1/chats/'||c.id::text||'/messages' AS messages_url,c.created_at FROM direct_chats c
JOIN profiles pr ON pr.user_id=CASE WHEN c.user_a=$1 THEN c.user_b ELSE c.user_a END WHERE $1 IN(c.user_a,c.user_b)
UNION ALL SELECT e.id,'event',e.title,NULL,e.id::text,'/api/v1/events/'||e.id::text||'/comments',e.created_at FROM events e
WHERE (e.organizer_id=$1 OR EXISTS(SELECT 1 FROM participations p WHERE p.event_id=e.id AND p.user_id=$1 AND p.status='confirmed'))
AND e.status<>'deleted') x WHERE (x.created_at,x.id)<($2,$3::uuid) ORDER BY x.created_at DESC,x.id DESC LIMIT $4`, a.ID, p.Before, p.ID, p.Limit+1)
	if err != nil {
		return SocialResult{}, err
	}
	return socialRows(rows, p)
}
func (r *Repository) DirectMessages(ctx context.Context, a Actor, id string, p SocialPage) (SocialResult, error) {
	if err := socialUUID(id); err != nil {
		return SocialResult{}, err
	}
	var access bool
	if err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM direct_chats WHERE id=$1 AND $2 IN(user_a,user_b))`, id, a.ID).Scan(&access); err != nil {
		return SocialResult{}, err
	}
	if !access {
		return SocialResult{}, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `SELECT to_jsonb(x) FROM (SELECT m.id,m.chat_id,m.author_id,m.body,m.created_at FROM direct_messages m
JOIN direct_chats c ON c.id=m.chat_id WHERE c.id=$1 AND $2 IN(c.user_a,c.user_b) AND (m.created_at,m.id)<($3,$4::uuid)
ORDER BY m.created_at DESC,m.id DESC LIMIT $5) x`, id, a.ID, p.Before, p.ID, p.Limit+1)
	if err != nil {
		return SocialResult{}, err
	}
	return socialRows(rows, p)
}
func (r *Repository) SendDirectMessage(ctx context.Context, a Actor, id, body string) (any, error) {
	if err := socialUUID(id); err != nil {
		return nil, err
	}
	body, err := socialBody(body, 2000)
	if err != nil {
		return nil, err
	}
	var out any
	err = r.inTx(ctx, func(tx pgx.Tx) error {
		var peer, lo, hi string
		if err := tx.QueryRow(ctx, `SELECT user_a,user_b,CASE WHEN user_a=$2 THEN user_b ELSE user_a END FROM direct_chats WHERE id=$1 AND $2 IN(user_a,user_b)`, id, a.ID).Scan(&lo, &hi, &peer); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
		var yes bool
		if err := tx.QueryRow(ctx, `SELECT true FROM friendships WHERE user_a=$1 AND user_b=$2 AND status='accepted' FOR SHARE`, lo, hi).Scan(&yes); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrSocialConflict
			}
			return err
		}
		var err error
		out, err = socialJSON(tx.QueryRow(ctx, `WITH m AS (INSERT INTO direct_messages(id,chat_id,author_id,body) VALUES($1,$2,$3,$4) RETURNING *) SELECT to_jsonb(m) FROM m`, uuid.NewString(), id, a.ID, body))
		if err != nil {
			return err
		}
		if err = socialSignal(ctx, tx, peer, "chat.changed", id); err != nil {
			return err
		}
		return socialSignal(ctx, tx, a.ID, "chat.changed", id)
	})
	return out, err
}

func (r *Repository) CreateReview(ctx context.Context, a Actor, eventID string, score int, body string) (any, error) {
	if err := socialUUID(eventID); err != nil {
		return nil, err
	}
	if score < 1 || score > 5 {
		return nil, FieldError{"score", "Оценка — от 1 до 5"}
	}
	body, err := socialBody(body, 2000)
	if err != nil {
		return nil, err
	}
	var out any
	err = r.inTx(ctx, func(tx pgx.Tx) error {
		var organizer string
		// Подтверждение записи не доказывает посещение. Поэтому отзыв относится к мероприятию,
		// а не превращается в рейтинг надёжности участника.
		err := tx.QueryRow(ctx, `SELECT e.organizer_id FROM events e JOIN participations p ON p.event_id=e.id
WHERE e.id=$1 AND p.user_id=$2 AND p.status='confirmed' AND e.organizer_id<>$2 AND e.status='published'
AND COALESCE(e.ends_at,e.starts_at+interval '2 hours')<=now() FOR SHARE OF e,p`, eventID, a.ID).Scan(&organizer)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSocialConflict
		}
		if err != nil {
			return err
		}
		out, err = socialJSON(tx.QueryRow(ctx, `WITH x AS (INSERT INTO event_reviews(id,event_id,author_id,organizer_id,score,body) VALUES($1,$2,$3,$4,$5,$6)
ON CONFLICT(event_id,author_id) DO NOTHING RETURNING id,event_id,author_id,organizer_id,score,body,created_at) SELECT to_jsonb(x) FROM x`, uuid.NewString(), eventID, a.ID, organizer, score, body))
		if errors.Is(err, ErrNotFound) {
			return ErrSocialConflict
		}
		if err != nil {
			return err
		}
		return socialSignal(ctx, tx, organizer, "reviews.changed", eventID)
	})
	return out, err
}
func (r *Repository) ProfileReviews(ctx context.Context, a Actor, id string, p SocialPage) (SocialResult, error) {
	rows, err := r.db.Query(ctx, `SELECT to_jsonb(x) FROM (SELECT r.id,r.event_id,r.author_id,pr.display_name AS author_name,r.organizer_id,r.score,r.body,r.created_at
FROM event_reviews r JOIN profiles pr ON pr.user_id=r.author_id JOIN events e ON e.id=r.event_id
WHERE r.organizer_id=$1 AND r.hidden_at IS NULL AND e.status='published' AND (r.created_at,r.id)<($2,$3::uuid)
ORDER BY r.created_at DESC,r.id DESC LIMIT $4) x`, id, p.Before, p.ID, p.Limit+1)
	if err != nil {
		return SocialResult{}, err
	}
	return socialRows(rows, p)
}
func (r *Repository) HideReview(ctx context.Context, a Actor, id, reason string) error {
	if err := socialUUID(id); err != nil {
		return err
	}
	reason, err := socialBody(reason, 500)
	if err != nil {
		return err
	}
	return r.inTx(ctx, func(tx pgx.Tx) error {
		var organizer string
		err := tx.QueryRow(ctx, `UPDATE event_reviews SET hidden_at=COALESCE(hidden_at,now()),hidden_by=$2,hidden_reason=$3 WHERE id=$1 AND (author_id=$2 OR $4) RETURNING organizer_id`, id, a.ID, reason, a.Admin).Scan(&organizer)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		return socialSignal(ctx, tx, organizer, "reviews.changed", id)
	})
}
func (r *Repository) CreateStory(ctx context.Context, a Actor, body string) (any, error) {
	body, err := socialBody(body, 1000)
	if err != nil {
		return nil, err
	}
	// Медиа добавляется отдельной загрузкой после проверки владельца истории.
	return socialJSON(r.db.QueryRow(ctx, `WITH x AS (INSERT INTO stories(id,author_id,body) VALUES($1,$2,$3) RETURNING id,author_id,body,created_at,expires_at) SELECT to_jsonb(x) FROM x`, uuid.NewString(), a.ID, body))
}
func (r *Repository) Stories(ctx context.Context, a Actor, p SocialPage) (SocialResult, error) {
	rows, err := r.db.Query(ctx, `SELECT to_jsonb(x) FROM (SELECT s.id,s.author_id,pr.display_name AS author_name,s.body,s.created_at,s.expires_at,
CASE WHEN s.media_key IS NOT NULL THEN '/api/v1/stories/'||s.id::text||'/media' ELSE NULL END AS media_url,s.media_content_type,s.media_size
FROM stories s JOIN profiles pr ON pr.user_id=s.author_id WHERE s.deleted_at IS NULL AND s.expires_at>now()
AND (s.author_id=$1 OR EXISTS(SELECT 1 FROM friendships f WHERE f.user_a=LEAST($1,s.author_id) AND f.user_b=GREATEST($1,s.author_id) AND f.status='accepted'))
AND (s.created_at,s.id)<($2,$3::uuid) ORDER BY s.created_at DESC,s.id DESC LIMIT $4) x`, a.ID, p.Before, p.ID, p.Limit+1)
	if err != nil {
		return SocialResult{}, err
	}
	return socialRows(rows, p)
}
func (r *Repository) DeleteStory(ctx context.Context, a Actor, id, reason string) error {
	if err := socialUUID(id); err != nil {
		return err
	}
	reason, err := socialBody(reason, 500)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `UPDATE stories SET deleted_at=COALESCE(deleted_at,now()),deleted_by=$2,delete_reason=$3 WHERE id=$1 AND (author_id=$2 OR $4)`, id, a.ID, reason, a.Admin)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (r *Repository) ClubMembership(ctx context.Context, a Actor, id string, join bool) error {
	if err := socialUUID(id); err != nil {
		return err
	}
	return r.inTx(ctx, func(tx pgx.Tx) error {
		var owner string
		err := tx.QueryRow(ctx, `SELECT owner_id FROM organizations WHERE id=$1 AND status='approved' FOR SHARE`, id).Scan(&owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if owner == a.ID {
			return ErrSocialConflict
		}
		if join {
			_, err = tx.Exec(ctx, `INSERT INTO organization_members(organization_id,user_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, id, a.ID)
		} else {
			_, err = tx.Exec(ctx, `DELETE FROM organization_members WHERE organization_id=$1 AND user_id=$2`, id, a.ID)
		}
		return err
	})
}
func (r *Repository) ClubMembers(ctx context.Context, a Actor, id string, p SocialPage) (SocialResult, error) {
	if err := socialUUID(id); err != nil {
		return SocialResult{}, err
	}
	var yes bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1 AND status='approved' AND (owner_id=$2 OR $3))`, id, a.ID, a.Admin).Scan(&yes)
	if err != nil {
		return SocialResult{}, err
	}
	if !yes {
		return SocialResult{}, ErrNotFound
	}
	rows, err := r.db.Query(ctx, `SELECT to_jsonb(x) FROM (SELECT pr.user_id AS id,pr.display_name,m.created_at FROM organization_members m JOIN profiles pr ON pr.user_id=m.user_id
JOIN organizations o ON o.id=m.organization_id WHERE o.id=$1 AND o.status='approved' AND (o.owner_id=$2 OR $3) AND (m.created_at,pr.user_id)<($4,$5)
ORDER BY m.created_at DESC,pr.user_id DESC LIMIT $6) x`, id, a.ID, a.Admin, p.Before, p.ID, p.Limit+1)
	if err != nil {
		return SocialResult{}, err
	}
	return socialRows(rows, p)
}
