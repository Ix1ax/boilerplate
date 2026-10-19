//go:build integration

package events

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOnlineRepository(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db := isolatedDB(t, ctx)
	repo := NewRepository(db)
	service := New(repo)
	owner := Actor{ID: "video-owner", Name: "Олег"}
	member := Actor{ID: "video-member", Name: "Анна"}
	outsider := Actor{ID: "video-outsider", Admin: true}
	for _, a := range []Actor{owner, member, outsider} {
		if err := repo.EnsureProfile(ctx, a); err != nil {
			t.Fatal(err)
		}
	}
	event, err := service.CreateEvent(ctx, owner, EventInput{Title: "Онлайн зарядка", SportName: "Онлайн спорт", StartsAt: time.Now().Add(time.Hour), Location: "Онлайн", JoinMode: "open"})
	if err != nil {
		t.Fatal(err)
	}
	format, err := repo.GetEventFormat(ctx, member, event.ID)
	if err != nil || format.Format != "offline" {
		t.Fatalf("default: %+v %v", format, err)
	}
	if _, err := repo.SetEventFormat(ctx, member, event.ID, "online"); !errors.Is(err, ErrNotEventOwner) {
		t.Fatalf("not owner: %v", err)
	}
	if _, err := repo.SetEventFormat(ctx, owner, event.ID, "online"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetVideoAccess(ctx, outsider, event.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("admin bypass: %v", err)
	}
	if _, err := service.Join(ctx, member, event.ID, "", false); err != nil {
		t.Fatal(err)
	}
	access, err := repo.GetVideoAccess(ctx, member, event.ID)
	if err != nil || !access.Confirmed || access.Format != "online" {
		t.Fatalf("access: %+v %v", access, err)
	}
	if _, err := db.Exec(ctx, "UPDATE events SET starts_at=now()+interval '5 minutes' WHERE id=$1", event.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SetEventFormat(ctx, owner, event.ID, "offline"); !errors.Is(err, ErrEventPast) {
		t.Fatalf("format switched after entry opened: %v", err)
	}
	if err := service.Leave(ctx, member, event.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetVideoAccess(ctx, member, event.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("left member: %v", err)
	}
}
