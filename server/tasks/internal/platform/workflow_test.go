package platform

import (
	"errors"
	"testing"
)

func TestWorkflow(t *testing.T) {
	w := NewWorkflow(map[string][]string{"draft": {"review"}, "review": {"published", "draft"}}).RequireComment("draft")
	if err := w.Check("draft", "review", ""); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][2]string{{"draft", "published"}, {"published", "draft"}, {"unknown", "draft"}} {
		if err := w.Check(c[0], c[1], "x"); !errors.Is(err, ErrTransition) {
			t.Errorf("%v: переход должен быть запрещён", c)
		}
	}
	// Возврат на доработку без причины запрещён, с причиной — разрешён.
	if err := w.Check("review", "draft", "  "); !errors.Is(err, ErrCommentRequired) {
		t.Fatalf("ожидалось требование комментария: %v", err)
	}
	if err := w.Check("review", "draft", "Уточните сроки"); err != nil {
		t.Fatal(err)
	}
	if len(w.Allowed("review")) != 2 || len(w.Allowed("published")) != 0 {
		t.Fatal("неверный список переходов")
	}
}
