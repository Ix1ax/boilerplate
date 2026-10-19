//go:build integration

package moderation

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/server/moderation/internal/database"
	"example.com/hackathon/server/moderation/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// isolatedDB — отдельная временная база на сервере из DATABASE_URL: тест не трогает данные демо.
func isolatedDB(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("Укажите DATABASE_URL для интеграционного теста")
	}
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	name := "moderation_it_" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		admin.Close()
	})
	u, _ := url.Parse(dsn)
	u.Path = "/" + name
	if err = database.Migrate(ctx, u.String(), "up"); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	return db
}

func TestProjections(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	repo := NewRepository(isolatedDB(t, ctx))
	s := New(repo)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	at := time.Now().Add(-time.Hour)
	send := func(entity, id, kind, from, to, actor, comment string, when time.Time, snapshot map[string]any) string {
		t.Helper()
		msg := uuid.NewString()
		data, _ := json.Marshal(map[string]any{"entity": entity, "id": id, "kind": kind, "from": from, "to": to,
			"actor": actor, "comment": comment, "at": when, "snapshot": snapshot})
		must(s.HandleEvent(ctx, msg, "domain."+entity+".changed", data))
		return msg
	}
	org := uuid.NewString()
	orgSnap := func(status, reason string) map[string]any {
		return map[string]any{"id": org, "name": "Клуб «Сосны»", "city": "Красноярск", "owner_id": "u1", "owner_name": "Мария",
			"status": status, "review_reason": reason, "created_at": at}
	}
	send("organization", org, "organization", "none", "pending", "Мария", "", at, orgSnap("pending", ""))
	rejected := send("organization", org, "organization", "pending", "rejected", "Модератор", "Нет расписания", at.Add(time.Minute), orgSnap("rejected", "Нет расписания"))
	// Опоздавший старый снимок не откатывает статус, а повтор того же события не дублирует журнал.
	send("organization", org, "organization", "none", "pending", "Мария", "", at.Add(-time.Minute), orgSnap("pending", ""))
	data, _ := json.Marshal(map[string]any{"entity": "organization", "id": org, "kind": "organization", "from": "pending", "to": "rejected",
		"actor": "Модератор", "comment": "Нет расписания", "at": at.Add(time.Minute), "snapshot": orgSnap("rejected", "Нет расписания")})
	must(s.HandleEvent(ctx, rejected, "domain.organization.changed", data))

	event := uuid.NewString()
	eventSnap := func(status, quorum string) map[string]any {
		return map[string]any{"id": event, "title": "Футбол во дворе", "sport": "Футбол", "organizer_id": "u2", "organizer_name": "Игорь",
			"organization_id": nil, "organization_name": "", "starts_at": time.Now().Add(48 * time.Hour), "location": "Стадион",
			"join_mode": "open", "capacity": 10, "min_participants": 4, "quorum_status": quorum, "outdoor": true,
			"status": status, "moderation_reason": "", "created_at": time.Now()}
	}
	send("event", event, "event", "none", "published", "Игорь", "", at, eventSnap("published", "gathering"))
	for i, first := range []bool{true, false} {
		p := uuid.NewString()
		send("participation", p, "participation", "none", "confirmed", "u", "", at.Add(time.Duration(i)*time.Second),
			map[string]any{"id": p, "event_id": event, "event_title": "Футбол во дворе", "user_id": "x", "user_name": "Анна",
				"status": "confirmed", "first_time": first, "created_at": time.Now()})
	}
	comment := uuid.NewString()
	send("comment", comment, "comment", "visible", "hidden", "Модератор", "Реклама", at.Add(2*time.Minute),
		map[string]any{"id": comment, "event_id": event, "event_title": "Футбол во дворе", "author_id": "u3", "author_name": "Спамер",
			"body": "Продам абонемент", "status": "hidden", "hidden_reason": "Реклама", "created_at": at})

	d, err := repo.Dashboard(ctx)
	must(err)
	if d.Organizations.Rejected != 1 || d.Organizations.Pending != 0 {
		t.Fatalf("старый снимок откатил организацию: %+v", d.Organizations)
	}
	if d.Events.Upcoming != 1 || d.Participations.Confirmed != 2 || d.Participations.FirstTime != 1 || d.Comments.Hidden != 1 || d.Quorum.Gathering != 1 {
		t.Fatalf("дашборд: %+v", d)
	}
	if len(d.Series) != 14 || len(d.TopSports) != 1 || d.TopSports[0].Sport != "Футбол" || d.UpdatedAt == nil {
		t.Fatalf("ряд и направления: %+v %+v", d.Series, d.TopSports)
	}

	log, err := repo.Log(ctx, platform.ListQuery{Limit: 10, OrderBy: " ORDER BY at DESC"})
	must(err)
	if log.Total != 2 {
		t.Fatalf("в журнале %d записей, ожидалось 2 (отказ и скрытие): %+v", log.Total, log.Items)
	}
	if log.Items[0].Entity != "comment" || !strings.Contains(log.Items[0].Title, "Продам абонемент") || log.Items[1].Comment != "Нет расписания" {
		t.Fatalf("журнал: %+v", log.Items)
	}
	rows, err := repo.EventRows(ctx)
	must(err)
	if len(rows) != 1 || rows[0][0] != "Футбол во дворе" || rows[0][9] != "2" || rows[0][11] != "1" || rows[0][13] != "4" {
		t.Fatalf("строка отчёта: %v", rows)
	}
	orgRows, err := repo.OrganizationRows(ctx)
	must(err)
	if len(orgRows) != 1 || orgRows[0][3] != "rejected" || orgRows[0][4] != "Нет расписания" {
		t.Fatalf("отчёт по организациям: %v", orgRows)
	}
}
