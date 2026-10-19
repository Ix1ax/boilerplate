package moderation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestLogged(t *testing.T) {
	cases := []struct {
		name string
		e    Envelope
		want bool
	}{
		{"отказ организации", Envelope{Entity: "organization", Kind: "organization", From: "pending", To: "rejected", Actor: "Модератор"}, true},
		{"подача заявки", Envelope{Entity: "organization", Kind: "organization", From: "none", To: "pending"}, false},
		{"снятие события", Envelope{Entity: "event", Kind: "event", From: "published", To: "unpublished", Actor: "Модератор"}, true},
		{"автоотмена — тоже решение", Envelope{Entity: "event", Kind: "event", From: "published", To: "cancelled", Actor: "Платформа"}, true},
		{"состав собран — не модерация", Envelope{Entity: "event", Kind: "event_quorum", From: "gathering", To: "reached"}, false},
		{"скрытие сообщения", Envelope{Entity: "comment", Kind: "comment", From: "visible", To: "hidden"}, true},
		{"новое сообщение", Envelope{Entity: "comment", Kind: "comment", From: "none", To: "visible"}, false},
		{"запись участника", Envelope{Entity: "participation", Kind: "participation", From: "pending", To: "confirmed"}, false},
		{"переотправка снимка", Envelope{Entity: "event", Kind: "event", Actor: "snapshot"}, false},
		{"отказ по жалобе", Envelope{Entity: "report", Kind: "report", From: "open", To: "dismissed", Actor: "Модератор"}, true},
		{"жалоба принята — уже в журнале как скрытие цели", Envelope{Entity: "report", Kind: "report", From: "open", To: "resolved", Actor: "Модератор"}, false},
		{"новая жалоба", Envelope{Entity: "report", Kind: "report", From: "none", To: "open", Actor: "Анна"}, false},
		{"скрытие направления", Envelope{Entity: "sport", Kind: "sport", From: "visible", To: "hidden", Actor: "Модератор"}, true},
		{"возврат направления", Envelope{Entity: "sport", Kind: "sport", From: "hidden", To: "visible", Actor: "Модератор"}, true},
		{"новое направление", Envelope{Entity: "sport", Kind: "sport", From: "none", To: "visible", Actor: "Анна"}, false},
	}
	for _, tc := range cases {
		if got := Logged(tc.e); got != tc.want {
			t.Errorf("%s: %v", tc.name, got)
		}
	}
}

type spyStore struct{ applied []Envelope }

func (s *spyStore) Apply(_ context.Context, _ string, e Envelope) error {
	s.applied = append(s.applied, e)
	return nil
}

func TestHandleEventSkipsBadMessages(t *testing.T) {
	spy := &spyStore{}
	s := New(spy)
	id, msg := uuid.NewString(), uuid.NewString()
	at := time.Now().UTC().Format(time.RFC3339Nano)
	good := `{"entity":"event","id":"` + id + `","kind":"event","at":"` + at + `","snapshot":{"id":"` + id + `"}}`
	for name, tc := range map[string]struct{ typ, data, msg string }{
		"чужой тип":     {"notification.requested", good, msg},
		"не JSON":       {"domain.event.changed", "{", msg},
		"без снимка":    {"domain.event.changed", strings.Replace(good, `"snapshot":{"id":"`+id+`"}`, `"snapshot":null`, 1), msg},
		"плохой id":     {"domain.event.changed", strings.Replace(good, id, "x", 2), msg},
		"плохое письмо": {"domain.event.changed", good, "not-uuid"},
	} {
		if err := s.HandleEvent(context.Background(), tc.msg, tc.typ, []byte(tc.data)); err != nil || len(spy.applied) != 0 {
			t.Fatalf("%s: err=%v, применено %d", name, err, len(spy.applied))
		}
	}
	if err := s.HandleEvent(context.Background(), msg, "domain.event.changed", []byte(good)); err != nil || len(spy.applied) != 1 {
		t.Fatalf("корректное событие не применено: %v", err)
	}
}

func TestUpsertKeepsNewerSnapshot(t *testing.T) {
	sql := upsertSQL("events", []string{"title", "status"})
	for _, want := range []string{"jsonb_populate_record(NULL::events", "SET (title, status, changed_at, domain_version) = ROW(EXCLUDED.title, EXCLUDED.status, EXCLUDED.changed_at, EXCLUDED.domain_version)", "WHERE events.domain_version < EXCLUDED.domain_version", "events.domain_version = 0 AND EXCLUDED.domain_version = 0 AND events.changed_at <= EXCLUDED.changed_at"} {
		if !strings.Contains(sql, want) {
			t.Fatalf("нет %q в\n%s", want, sql)
		}
	}
}
