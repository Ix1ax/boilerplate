//go:build integration

package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDemandClaimAndArrival(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	repo := NewRepository(db)
	svc := New(repo)
	actors := []Actor{{ID: "demand-a", Name: "Олег"}, {ID: "demand-b", Name: "Анна"}, {ID: "demand-c", Name: "Борис"}}
	for _, a := range actors {
		if err := repo.EnsureProfile(ctx, a); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.CreateCompanyRequest(ctx, a, CompanyRequestInput{SportName: "Бег по району", Level: "beginner", Area: "Октябрьский район", DayParts: []string{"morning"}, Weekdays: []int{6, 7}}); err != nil {
			t.Fatal(err)
		}
	}
	proposals, err := repo.DemandProposals(ctx, actors[0])
	if err != nil || len(proposals) != 1 {
		t.Fatalf("предложения: %+v %v", proposals, err)
	}
	if proposals[0].MembersCount != 3 {
		t.Fatal("нужны три человека")
	}
	if _, err = repo.ClaimDemandProposal(ctx, Actor{ID: "outsider"}, proposals[0].ID, ProposalClaim{Location: "У входа в парк"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("чужому недоступно: %v", err)
	}
	var wg sync.WaitGroup
	wins := make(chan Event, 2)
	errs := make(chan error, 2)
	for _, a := range actors[:2] {
		wg.Add(1)
		go func(a Actor) {
			defer wg.Done()
			e, err := repo.ClaimDemandProposal(ctx, a, proposals[0].ID, ProposalClaim{Location: "У входа в парк"})
			if err == nil {
				wins <- e
			} else {
				errs <- err
			}
		}(a)
	}
	wg.Wait()
	close(wins)
	close(errs)
	if len(wins) != 1 || len(errs) != 1 {
		t.Fatalf("ровно один организатор: %d %d", len(wins), len(errs))
	}
	e := <-wins
	var n int
	if err = db.QueryRow(ctx, "SELECT count(*) FROM participations WHERE event_id=$1", e.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("автозаписи нет: %d %v", n, err)
	}
	var organizer string
	if err = db.QueryRow(ctx, "SELECT organizer_id FROM events WHERE id=$1", e.ID).Scan(&organizer); err != nil {
		t.Fatal(err)
	}
	var participant Actor
	for _, a := range actors {
		if a.ID != organizer {
			participant = a
			break
		}
	}
	if _, err = svc.Join(ctx, participant, e.ID, "", false); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SetArrival(ctx, participant, e.ID, ArrivalInput{Status: "here"}); err == nil {
		t.Fatal("до окна отметка недоступна")
	}
	if _, err = db.Exec(ctx, "UPDATE events SET starts_at=$2,ends_at=$3 WHERE id=$1", e.ID, time.Now().Add(10*time.Minute), time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SetArrival(ctx, participant, e.ID, ArrivalInput{Status: "late", ETAMinutes: 10}); err != nil {
		t.Fatal(err)
	}
	roster, err := repo.Arrivals(ctx, Actor{ID: organizer}, e.ID)
	if err != nil || len(roster) != 2 {
		t.Fatalf("состав: %+v %v", roster, err)
	}
	if _, err = repo.Arrivals(ctx, Actor{ID: "outsider"}, e.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("приватность: %v", err)
	}
	if err = repo.Leave(ctx, participant, e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = repo.SetArrival(ctx, participant, e.ID, ArrivalInput{Status: "here"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ушедший не может отмечаться: %v", err)
	}
}

func TestExpiredCompanyRequestCanBeReplaced(t *testing.T) {
	ctx := context.Background()
	db := isolatedDB(t, ctx)
	repo := NewRepository(db)
	svc := New(repo)
	a := Actor{ID: "replace", Name: "Олег"}
	if err := repo.EnsureProfile(ctx, a); err != nil {
		t.Fatal(err)
	}
	in := CompanyRequestInput{SportName: "Пешая прогулка", Area: "Центр", Level: "any"}
	old, err := svc.CreateCompanyRequest(ctx, a, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, "UPDATE company_requests SET expires_at=now()-interval '1 day' WHERE id=$1", old.ID); err != nil {
		t.Fatal(err)
	}
	next, err := svc.CreateCompanyRequest(ctx, a, in)
	if err != nil || next.ID == old.ID {
		t.Fatalf("новый запрос: %+v %v", next, err)
	}
}
