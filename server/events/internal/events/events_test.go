package events

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/hackathon/server/events/internal/platform"
	"github.com/gin-gonic/gin"
)

func validEvent(now time.Time) EventInput {
	return EventInput{Title: "Пробежка", SportName: "  Бег  ", StartsAt: now.Add(time.Hour), Location: "Набережная", JoinMode: "open"}
}

func TestValidateEvent(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	one, zero, past := 1, 0, now.Add(-time.Minute)
	lat := 56.0
	cases := []struct {
		name  string
		edit  func(*EventInput)
		field string
	}{
		{"корректное", func(*EventInput) {}, ""},
		{"короткое название", func(e *EventInput) { e.Title = "Йо" }, "title"},
		{"нет направления", func(e *EventInput) { e.SportName = " " }, "sport"},
		{"чужой sport_id", func(e *EventInput) { e.SportID = "not-uuid" }, "sport"},
		{"в прошлом", func(e *EventInput) { e.StartsAt = past }, "starts_at"},
		{"конец раньше начала", func(e *EventInput) { e.EndsAt = &past }, "ends_at"},
		{"нет места", func(e *EventInput) { e.Location = "" }, "location"},
		{"одна координата", func(e *EventInput) { e.Lat = &lat }, "location"},
		{"неизвестный режим", func(e *EventInput) { e.JoinMode = "invite" }, "join_mode"},
		{"неизвестный уровень", func(e *EventInput) { e.Level = "master" }, "level"},
		{"ноль мест", func(e *EventInput) { e.Capacity = &zero }, "capacity"},
		{"одно место", func(e *EventInput) { e.Capacity = &one }, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := validEvent(now)
			tc.edit(&in)
			err := ValidateEvent(&in, now)
			var fe FieldError
			if tc.field == "" {
				if err != nil {
					t.Fatalf("ожидалось без ошибки, получено %v", err)
				}
				return
			}
			if !errors.As(err, &fe) || fe.Field != tc.field || !errors.Is(err, ErrValidation) {
				t.Fatalf("ожидалась ошибка поля %s, получено %v", tc.field, err)
			}
		})
	}
	// Своё направление нормализуется: лишние пробелы не создают дубль в справочнике.
	in := validEvent(now)
	in.SportName = "  Алтимат   фрисби "
	if err := ValidateEvent(&in, now); err != nil || in.SportName != "Алтимат фрисби" || in.Level != "any" {
		t.Fatalf("нормализация: %q %q %v", in.SportName, in.Level, err)
	}
}

func TestValidateOrganization(t *testing.T) {
	ok := OrganizationInput{Name: "Ладья", Description: "Шахматный клуб для всех"}
	if err := ValidateOrganization(&ok); err != nil {
		t.Fatal(err)
	}
	for name, in := range map[string]OrganizationInput{
		"name":        {Name: "Л", Description: "Шахматный клуб для всех"},
		"description": {Name: "Ладья", Description: "коротко"},
		"sport_ids":   {Name: "Ладья", Description: "Шахматный клуб для всех", SportIDs: []string{"x"}},
	} {
		var fe FieldError
		if err := ValidateOrganization(&in); !errors.As(err, &fe) || fe.Field != name {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

// Причина обязательна там, где решение увидит другая сторона.
func TestWorkflowsRequireReason(t *testing.T) {
	cases := []struct {
		w             platform.Workflow
		from, to, why string
		want          error
	}{
		{orgWorkflow, "pending", "approved", "", nil},
		{orgWorkflow, "pending", "rejected", "", platform.ErrCommentRequired},
		{orgWorkflow, "pending", "rejected", "Нет контактов", nil},
		{orgWorkflow, "approved", "rejected", "x", platform.ErrTransition},
		{orgWorkflow, "rejected", "pending", "", nil},
		{eventWorkflow, "published", "unpublished", "", platform.ErrCommentRequired},
		{eventWorkflow, "published", "deleted", "Спам", nil},
		{eventWorkflow, "published", "cancelled", "", platform.ErrCommentRequired},
		{eventWorkflow, "unpublished", "published", "", nil},
		{eventWorkflow, "deleted", "published", "", platform.ErrTransition},
		{eventWorkflow, "cancelled", "published", "", platform.ErrTransition},
	}
	for _, tc := range cases {
		err := tc.w.Check(tc.from, tc.to, tc.why)
		if (tc.want == nil) != (err == nil) || tc.want != nil && !errors.Is(err, tc.want) {
			t.Errorf("%s → %s (%q): %v, ожидалось %v", tc.from, tc.to, tc.why, err, tc.want)
		}
	}
}

// Ошибки домена переводятся в понятные коды ответа; внутренние детали не утекают.
func TestHandleMapsErrors(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	cases := []struct {
		err  error
		code int
	}{
		{FieldError{"title", "x"}, 422},
		{ErrNotFound, 404},
		{ErrOrgNotApproved, 403},
		{ErrNotOrganizer, 403},
		{ErrEventClosed, 409},
		{ErrFull, 409},
		{fmt.Errorf("%w: уже", platform.ErrTransition), 409},
		{platform.ErrCommentRequired, 422},
		{errors.New("pq: connection refused"), 500},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		if !handle(c, tc.err) || w.Code != tc.code {
			t.Errorf("%v: код %d, ожидался %d", tc.err, w.Code, tc.code)
		}
		if tc.code == 500 && strings.Contains(w.Body.String(), "pq:") {
			t.Errorf("внутренняя ошибка попала в ответ: %s", w.Body.String())
		}
	}
}
