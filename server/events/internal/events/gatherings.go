package events

import (
	"context"
	"errors"
	"fmt"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/jackc/pgx/v5"
)

// Два улучшения основного пути, которые снимают типичные причины «так и не пошёл»:
//
//   - «Соберём состав»: футбол вчетвером не случится. Организатор задаёт минимум и срок;
//     набрали — всем «встреча состоится», не набрали — автоотмена с понятной причиной.
//   - «Встречу новичка»: страшно прийти в незнакомую компанию. Новичок отмечает «иду впервые»,
//     опытный участник — «готов встретить», и платформа назначает пару; оба получают имя друг друга.
//
// Всё меняется в транзакции записи или решения по заявке, по заблокированной строке события.

// maxNewcomersPerBuddy — больше трёх человек одному не встретить и не познакомить с остальными.
const maxNewcomersPerBuddy = 3

var (
	ErrNotConfirmed  = errors.New("Встречать новичков могут только подтверждённые участники")
	ErrNewcomerBuddy = errors.New("Вы сами идёте впервые — вас встретят. Проводником можно стать в следующий раз")
)

// systemActor — автор автоматических решений (автоотмена по сроку), виден в истории и уведомлениях.
var systemActor = Actor{ID: "system", Name: "Платформа"}

func quorumStatusFor(minParticipants *int) string {
	if minParticipants == nil {
		return "none"
	}
	return "gathering"
}

// afterConfirmed вызывается, когда запись pid стала подтверждённой: подбираем проводника новичку
// и проверяем, не собран ли состав.
func afterConfirmed(ctx context.Context, tx pgx.Tx, eventID, pid string, e lockedEvent) error {
	if err := assignBuddies(ctx, tx, eventID, e); err != nil {
		return err
	}
	var firstTime, alone bool
	var name string
	err := tx.QueryRow(ctx, `SELECT p.first_time, p.buddy_id IS NULL, COALESCE(pr.display_name,'')
FROM participations p LEFT JOIN profiles pr ON pr.user_id=p.user_id WHERE p.id=$1`, pid).Scan(&firstTime, &alone, &name)
	if err != nil {
		return err
	}
	if firstTime && alone {
		if err = notify(ctx, tx, e.OrganizerID, "Новичка некому встретить",
			name+" идёт на «"+e.Title+"» впервые, а готовых встретить пока нет. Встретьте сами или попросите опытных участников.",
			"/events/"+eventID); err != nil {
			return err
		}
	}
	return checkQuorum(ctx, tx, eventID, e)
}

// checkQuorum отмечает «состав собран», как только подтверждённых стало не меньше минимума.
// Обратно в «набираем» не возвращаем: людям уже сказали, что встреча будет.
func checkQuorum(ctx context.Context, tx pgx.Tx, eventID string, e lockedEvent) error {
	if e.QuorumStatus != "gathering" || e.MinParticipants == nil {
		return nil
	}
	n, err := confirmedCount(ctx, tx, eventID)
	if err != nil || n < *e.MinParticipants {
		return err
	}
	tag, err := tx.Exec(ctx, "UPDATE events SET quorum_status='reached', updated_at=now() WHERE id=$1 AND quorum_status='gathering'", eventID)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	if err = record(ctx, tx, "event_quorum", eventID, platform.Transition{From: "gathering", To: "reached", Actor: actorLabel(systemActor)}); err != nil {
		return err
	}
	body := fmt.Sprintf("«%s», %s: записались %d из %d нужных. До встречи!", e.Title, humanTime(e.StartsAt), n, *e.MinParticipants)
	recipients, err := confirmedUsers(ctx, tx, eventID)
	if err != nil {
		return err
	}
	for _, u := range append(recipients, e.OrganizerID) {
		if err = notify(ctx, tx, u, "Состав собран — встреча состоится", body, "/events/"+eventID); err != nil {
			return err
		}
	}
	return nil
}

func confirmedUsers(ctx context.Context, tx pgx.Tx, eventID string) ([]string, error) {
	rows, err := tx.Query(ctx, "SELECT user_id FROM participations WHERE event_id=$1 AND status='confirmed'", eventID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

type person struct{ pid, userID, name string }

// assignBuddies раздаёт подтверждённых новичков без проводника тем, у кого меньше всего подопечных
// (до трёх). Обоим сразу уходит уведомление с именем пары.
func assignBuddies(ctx context.Context, tx pgx.Tx, eventID string, e lockedEvent) error {
	rows, err := tx.Query(ctx, `SELECT p.id,p.user_id,COALESCE(pr.display_name,'') FROM participations p
LEFT JOIN profiles pr ON pr.user_id=p.user_id
WHERE p.event_id=$1 AND p.status='confirmed' AND p.first_time AND p.buddy_id IS NULL ORDER BY p.created_at, p.id`, eventID)
	if err != nil {
		return err
	}
	newcomers, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (person, error) {
		var p person
		return p, row.Scan(&p.pid, &p.userID, &p.name)
	})
	if err != nil {
		return err
	}
	for _, n := range newcomers {
		var b person
		err := tx.QueryRow(ctx, `SELECT b.id,b.user_id,COALESCE(pr.display_name,'') FROM participations b
LEFT JOIN profiles pr ON pr.user_id=b.user_id
WHERE b.event_id=$1 AND b.status='confirmed' AND b.buddy
  AND (SELECT count(*) FROM participations x WHERE x.buddy_id=b.id AND x.status='confirmed') < $2
ORDER BY (SELECT count(*) FROM participations x WHERE x.buddy_id=b.id AND x.status='confirmed'), b.created_at, b.id
LIMIT 1`, eventID, maxNewcomersPerBuddy).Scan(&b.pid, &b.userID, &b.name)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // проводники закончились — остальные новички ждут следующего добровольца
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, "UPDATE participations SET buddy_id=$2, updated_at=now() WHERE id=$1", n.pid, b.pid); err != nil {
			return err
		}
		link := "/events/" + eventID
		if err = notify(ctx, tx, n.userID, "Вас встретит "+b.name,
			"«"+e.Title+"»: "+b.name+" уже бывал на таких встречах и встретит вас на месте. Найдите друг друга у точки сбора.", link); err != nil {
			return err
		}
		if err = notify(ctx, tx, b.userID, n.name+" придёт впервые",
			"«"+e.Title+"»: встретьте "+n.name+" у точки сбора и познакомьте с остальными. Спасибо, что помогаете!", link); err != nil {
			return err
		}
	}
	return nil
}

// releaseNewcomers снимает подопечных с проводника buddyPID (он ушёл или передумал) и пробует
// передать их другим. Оставшихся без пары предупреждаем вместе с организатором.
func releaseNewcomers(ctx context.Context, tx pgx.Tx, eventID, buddyPID string, e lockedEvent) error {
	rows, err := tx.Query(ctx, "UPDATE participations SET buddy_id=NULL, updated_at=now() WHERE buddy_id=$1 AND status='confirmed' RETURNING id", buddyPID)
	if err != nil {
		return err
	}
	released, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil || len(released) == 0 {
		return err
	}
	if err = assignBuddies(ctx, tx, eventID, e); err != nil {
		return err
	}
	rows, err = tx.Query(ctx, `SELECT p.id,p.user_id,COALESCE(pr.display_name,'') FROM participations p
LEFT JOIN profiles pr ON pr.user_id=p.user_id WHERE p.id = ANY($1::uuid[]) AND p.buddy_id IS NULL`, released)
	if err != nil {
		return err
	}
	alone, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (person, error) {
		var p person
		return p, row.Scan(&p.pid, &p.userID, &p.name)
	})
	if err != nil {
		return err
	}
	link := "/events/" + eventID
	for _, n := range alone {
		if err = notify(ctx, tx, n.userID, "Проводник не сможет прийти",
			"«"+e.Title+"»: мы предупредили организатора — вас встретят на месте.", link); err != nil {
			return err
		}
		if err = notify(ctx, tx, e.OrganizerID, "Новичка некому встретить",
			n.name+" идёт на «"+e.Title+"» впервые, а проводник отказался. Встретьте сами или попросите опытных участников.", link); err != nil {
			return err
		}
	}
	return nil
}

func newcomersOf(ctx context.Context, q querier, buddyPID string) ([]Person, error) {
	rows, err := q.Query(ctx, `SELECT p.user_id,COALESCE(pr.display_name,''),COALESCE(pr.level,'') FROM participations p
LEFT JOIN profiles pr ON pr.user_id=p.user_id WHERE p.buddy_id=$1 AND p.status='confirmed' ORDER BY p.created_at`, buddyPID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Person, error) {
		var p Person
		return p, row.Scan(&p.ID, &p.Name, &p.Level)
	})
}

// SetBuddy — «готов встретить новичка» (on) или «передумал». Доступно подтверждённым участникам
// предстоящего опубликованного события, кроме самих новичков.
func (r *Repository) SetBuddy(ctx context.Context, a Actor, eventID string, on bool) (MyParticipation, error) {
	err := r.inTx(ctx, func(tx pgx.Tx) error {
		e, err := lockEvent(ctx, tx, eventID)
		if err != nil {
			return err
		}
		switch {
		case e.Status != "published":
			return ErrEventClosed
		case !e.StartsAt.After(time.Now()):
			return ErrEventPast
		}
		var pid, status string
		var firstTime bool
		err = tx.QueryRow(ctx, "SELECT id,status,first_time FROM participations WHERE event_id=$1 AND user_id=$2 FOR UPDATE", eventID, a.ID).Scan(&pid, &status, &firstTime)
		switch {
		case errors.Is(err, pgx.ErrNoRows) || err == nil && status != "confirmed":
			return ErrNotConfirmed
		case err != nil:
			return err
		case firstTime && on:
			return ErrNewcomerBuddy
		}
		if _, err = tx.Exec(ctx, "UPDATE participations SET buddy=$2, updated_at=now() WHERE id=$1", pid, on); err != nil {
			return err
		}
		if on {
			return assignBuddies(ctx, tx, eventID, e)
		}
		return releaseNewcomers(ctx, tx, eventID, pid, e)
	})
	if err != nil {
		return MyParticipation{}, err
	}
	ev, err := r.GetEvent(ctx, a, eventID)
	if err != nil {
		return MyParticipation{}, err
	}
	return *ev.MyParticipation, nil
}

// FailQuorums — фоновое задание: события, не собравшие состав к сроку, отменяются автоматически,
// записавшиеся и организатор получают причину. Каждое событие — в своей транзакции.
func (r *Repository) FailQuorums(ctx context.Context) error {
	rows, err := r.db.Query(ctx, `SELECT id FROM events WHERE quorum_status='gathering' AND status='published'
AND quorum_deadline <= now() ORDER BY quorum_deadline LIMIT 100`)
	if err != nil {
		return err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := r.inTx(ctx, func(tx pgx.Tx) error { return failQuorum(ctx, tx, id) }); err != nil {
			return fmt.Errorf("событие %s: %w", id, err)
		}
	}
	return nil
}

func failQuorum(ctx context.Context, tx pgx.Tx, id string) error {
	e, err := lockEvent(ctx, tx, id)
	if err != nil || e.Status != "published" || e.QuorumStatus != "gathering" || e.MinParticipants == nil {
		return err // уже решено другой транзакцией
	}
	n, err := confirmedCount(ctx, tx, id)
	if err != nil {
		return err
	}
	if n >= *e.MinParticipants {
		return checkQuorum(ctx, tx, id, e) // успели в последний момент
	}
	if _, err = tx.Exec(ctx, "UPDATE events SET quorum_status='failed', updated_at=now() WHERE id=$1", id); err != nil {
		return err
	}
	reason := fmt.Sprintf("Не набрался минимальный состав к сроку: записались %d из %d нужных", n, *e.MinParticipants)
	return changeEventStatus(ctx, tx, systemActor, id, e, "cancelled", reason, "Встреча не состоится: не собрали состав")
}
