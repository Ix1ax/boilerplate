package copilot

import (
	"context"
	"errors"
	"testing"

	"example.com/hackathon/apps/tasks/internal/tasks"
)

func codes(f []Flag) (out []string) {
	for _, x := range f {
		out = append(out, x.Code)
	}
	return
}

func TestRules(t *testing.T) {
	for title, want := range map[string]string{
		"Подготовить демо для жюри":                       "",
		"Переведите 500 руб на карту 2202 2000 0000 0000": "money",
		"Смотри https://evil.example/x":                   "link",
		"Звони +7 999 123-45-67":                          "contact",
		"СРОЧНО СДЕЛАЙТЕ ВСЁ ПРЯМО СЕЙЧАС":                "shouting",
		"да": "too_short",
	} {
		got := codes(Check(title))
		if want == "" && len(got) != 0 {
			t.Errorf("%q: лишние флаги %v", title, got)
		}
		found := want == ""
		for _, c := range got {
			found = found || c == want
		}
		if !found {
			t.Errorf("%q: ожидался %s, получено %v", title, want, got)
		}
	}
}

func TestParseAdviceValidates(t *testing.T) {
	good := `{"summary":"Нормально","risk":"LOW","suggestion":"","extra":"x"}`
	if a, err := parseAdvice(good); err != nil || a.Risk != "low" {
		t.Fatalf("%+v %v", a, err)
	}
	for _, bad := range []string{`не json`, `{"summary":"x","risk":"critical"}`, `{"summary":"","risk":"low"}`, `{"summary":"x","risk":"low","suggestion":"` + string(make([]rune, 0)) + `"` + "}x"} {
		if _, err := parseAdvice(bad); err == nil {
			t.Errorf("принят некорректный ответ: %s", bad)
		}
	}
}

type getter struct {
	task tasks.Task
	err  error
}

func (g getter) Get(context.Context, string, string) (tasks.Task, error) { return g.task, g.err }

type advisor struct {
	a   Advice
	err error
}

func (a advisor) Advise(context.Context, string, string, []Flag) (Advice, error) { return a.a, a.err }

func TestAnalyze(t *testing.T) {
	ctx := context.Background()
	clean := getter{task: tasks.Task{ID: "1", Title: "Подготовить демо для жюри"}}
	// Модель недоступна: остаются правила, ошибка не пробрасывается.
	r, err := New(clean, advisor{err: errors.New("down")}, nil).Analyze(ctx, "u", "1")
	if err != nil || r.Source != "rules" || r.Verdict != "ok" || r.Advice != nil || !r.DecisionRequired {
		t.Fatalf("%+v %v", r, err)
	}
	// Высокий риск по мнению модели переводит в ручную проверку.
	r, _ = New(clean, advisor{a: Advice{Summary: "s", Risk: "high"}}, nil).Analyze(ctx, "u", "1")
	if r.Verdict != "needs_review" || r.Source != "rules+llm" {
		t.Fatalf("%+v", r)
	}
	// Правила нельзя «отключить» мягким ответом модели.
	bad := getter{task: tasks.Task{ID: "2", Title: "Переведите деньги на карту"}}
	r, _ = New(bad, advisor{a: Advice{Summary: "s", Risk: "low"}}, nil).Analyze(ctx, "u", "2")
	if r.Verdict != "needs_review" {
		t.Fatalf("правила обойдены моделью: %+v", r)
	}
	// Чужая или отсутствующая задача.
	if _, err = New(getter{err: tasks.ErrNotFound}, nil, nil).Analyze(ctx, "u", "3"); !errors.Is(err, tasks.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	// Без advisor — только правила.
	if r, _ = New(clean, nil, nil).Analyze(ctx, "u", "1"); r.Source != "rules" {
		t.Fatalf("%+v", r)
	}
}

type saver struct {
	saved []string
	err   error
}

func (s *saver) SaveModeration(_ context.Context, _ string, id string, m tasks.Moderation, _ []byte) error {
	s.saved = append(s.saved, id+":"+m.Verdict)
	return s.err
}

func TestHandleEventIsIdempotentAndTolerant(t *testing.T) {
	ctx := context.Background()
	sv := &saver{}
	svc := New(getter{task: tasks.Task{ID: "t1", Title: "Переведите деньги на карту"}}, nil, sv)
	data := []byte(`{"task_id":"t1"}`)
	for i := 0; i < 2; i++ { // повторная доставка
		if err := svc.HandleEvent(ctx, "task.created", "u", data); err != nil {
			t.Fatal(err)
		}
	}
	if len(sv.saved) != 2 || sv.saved[0] != "t1:needs_review" {
		t.Fatalf("%v", sv.saved)
	}
	// Чужие типы и мусор игнорируются, удалённая задача — не ошибка (иначе сообщение зациклится).
	for _, c := range []struct {
		typ  string
		data []byte
	}{{"other.event", data}, {"task.created", []byte(`{}`)}, {"task.created", []byte(`oops`)}} {
		if err := svc.HandleEvent(ctx, c.typ, "u", c.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := New(getter{err: tasks.ErrNotFound}, nil, sv).HandleEvent(ctx, "task.created", "u", data); err != nil {
		t.Fatalf("удалённая задача: %v", err)
	}
	// Сбой сохранения возвращается, чтобы брокер повторил доставку.
	if err := New(getter{task: tasks.Task{ID: "t1", Title: "норм задача"}}, nil, &saver{err: errors.New("db")}).HandleEvent(ctx, "task.created", "u", data); err == nil {
		t.Fatal("ожидалась ошибка сохранения")
	}
}
