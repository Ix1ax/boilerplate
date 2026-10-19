//go:build integration

package events

import (
	"context"
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestRecommendationsPrivacyAndFeedback(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	r := NewRepository(db)
	s := New(r)
	person := func() Actor {
		a := Actor{ID: uuid.NewString(), Name: "Участник"}
		if err := r.EnsureProfile(ctx, a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	owner, user := person(), person()
	create := func(title, sport string) Event {
		e, err := s.CreateEvent(ctx, owner, EventInput{Title: title, SportName: sport, Location: "Красноярск, парк", StartsAt: time.Now().Add(24 * time.Hour), JoinMode: "open", PrivateInfo: "Код двери"})
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	run := create("Пробежка для всех", "Бег")
	chess := create("Шахматная встреча", "Шахматы")
	page, err := r.RecommendEvents(ctx, user, 20, 0)
	if err != nil || len(page.Items) != 2 || page.Personalized {
		t.Fatalf("cold start: %+v %v", page, err)
	}
	_, err = r.SaveRecommendationPreferences(ctx, user, RecommendationPreferences{SportIDs: []string{run.Sport.ID}, City: "Красноярск"})
	if err != nil {
		t.Fatal(err)
	}
	page, err = r.RecommendEvents(ctx, user, 20, 0)
	if err != nil || page.Items[0].Event.ID != run.ID || !page.Personalized {
		t.Fatalf("интересы: %+v %v", page, err)
	}
	for _, x := range page.Items {
		if x.Event.PrivateInfo != "" {
			t.Fatal("закрытая информация раскрыта в подборке")
		}
	}
	if err = r.RecommendationFeedback(ctx, user, run.ID, "dismiss"); err != nil {
		t.Fatal(err)
	}
	if err = r.RecommendationFeedback(ctx, user, run.ID, "view"); err != nil {
		t.Fatal(err)
	}
	page, err = r.RecommendEvents(ctx, user, 20, 0)
	if err != nil || len(page.Items) != 1 || page.Items[0].Event.ID != chess.ID {
		t.Fatalf("скрытие сброшено просмотром: %+v %v", page, err)
	}
	if err = r.RecommendationFeedback(ctx, user, run.ID, "reset"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Join(ctx, user, chess.ID, "", false); err != nil {
		t.Fatal(err)
	}
	page, err = r.RecommendEvents(ctx, user, 20, 0)
	if err != nil || len(page.Items) != 1 || page.Items[0].Event.ID != run.ID {
		t.Fatalf("уже записанная встреча предложена: %+v %v", page, err)
	}
	if _, err = s.CancelEvent(ctx, owner, run.ID, "Проверка отмены"); err != nil {
		t.Fatal(err)
	}
	page, err = r.RecommendEvents(ctx, user, 20, 0)
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("отменённое предложено: %+v %v", page, err)
	}
	progress, err := r.PersonalProgress(ctx, user)
	if err != nil || progress.Points != 10 {
		t.Fatalf("прогресс: %+v %v", progress, err)
	}
	again, err := r.PersonalProgress(ctx, user)
	if err != nil || again.Points != progress.Points {
		t.Fatal("повторное чтение начислило очки")
	}
}
