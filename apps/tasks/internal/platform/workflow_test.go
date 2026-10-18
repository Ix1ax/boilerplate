package platform

import (
	"errors"
	"testing"
)

func TestWorkflow(t *testing.T) {
	w := NewWorkflow(map[string][]string{"draft": {"review"}, "review": {"published", "draft"}})
	if err := w.Check("draft", "review"); err != nil {
		t.Fatal(err)
	}
	for _, c := range [][2]string{{"draft", "published"}, {"published", "draft"}, {"unknown", "draft"}} {
		if err := w.Check(c[0], c[1]); !errors.Is(err, ErrTransition) {
			t.Errorf("%v: переход должен быть запрещён", c)
		}
	}
	if len(w.Allowed("review")) != 2 || len(w.Allowed("published")) != 0 {
		t.Fatal("неверный список переходов")
	}
}
