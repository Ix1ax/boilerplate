package audit

import (
	"context"
	"testing"
)

type store struct{ n int }

func (s *store) Insert(context.Context, string, Record) error { s.n++; return nil }

func TestHandleEventValidates(t *testing.T) {
	st := &store{}
	s := New(st)
	ok := []byte(`{"actor":"u","method":"POST","route":"/api/v1/tasks","status":201,"occurred_at":"2026-10-18T10:00:00Z"}`)
	const id = "7f3b8a52-1c1e-4c8e-9a43-3a1d2f0b9c01"
	for _, c := range []struct {
		id, typ string
		data    []byte
	}{{id, "audit.recorded", ok}, {id, "other", ok}, {"not-uuid", "audit.recorded", ok}, {id, "audit.recorded", []byte(`{"actor":""}`)}, {id, "audit.recorded", []byte(`x`)}} {
		if err := s.HandleEvent(context.Background(), c.id, c.typ, c.data); err != nil {
			t.Fatal(err)
		}
	}
	if st.n != 1 {
		t.Fatalf("сохранено %d записей, ожидалась 1", st.n)
	}
}
