//go:build integration

package events

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func socialObject(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err = json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestSocialPrivacyAndDurability(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := isolatedDB(t, ctx)
	r := NewRepository(db)
	a, b, outsider := Actor{ID: "social-a", Name: "Анна"}, Actor{ID: "social-b", Name: "Борис"}, Actor{ID: "social-outsider", Name: "Виктор"}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, person := range []Actor{a, b, outsider} {
		must(r.EnsureProfile(ctx, person))
	}
	page, _ := parseSocialPage("", 1)
	if _, err := r.OpenDirectChat(ctx, a, b.ID); !errors.Is(err, ErrSocialConflict) {
		t.Fatal("chat without friendship", err)
	}
	_, err := r.Friendship(ctx, a, b.ID, "request")
	must(err)
	if _, err = r.Friendship(ctx, b, a.ID, "request"); !errors.Is(err, ErrSocialConflict) {
		t.Fatal("cross request autoaccept", err)
	}
	if _, err = r.Friendship(ctx, a, b.ID, "accept"); !errors.Is(err, ErrSocialConflict) {
		t.Fatal("author accepted own request", err)
	}
	_, err = r.Friendship(ctx, b, a.ID, "accept")
	must(err)
	chat, err := r.OpenDirectChat(ctx, a, b.ID)
	must(err)
	id := socialObject(t, chat)["id"].(string)
	again, err := r.OpenDirectChat(ctx, b, a.ID)
	must(err)
	if socialObject(t, again)["id"] != id {
		t.Fatal("duplicate chat")
	}
	for i := 0; i < 3; i++ {
		_, err = r.SendDirectMessage(ctx, a, id, fmt.Sprintf("Сообщение %d", i))
		must(err)
	}
	// Реплика сервиса читает ту же историю; процесс не хранит сообщения в памяти.
	otherReplica := NewRepository(db)
	messages, err := otherReplica.DirectMessages(ctx, b, id, page)
	must(err)
	if len(messages.Items) != 1 || messages.NextCursor == "" {
		t.Fatal("missing pagination", messages)
	}
	secondPage, err := parseSocialPage(messages.NextCursor, 1)
	must(err)
	next, err := r.DirectMessages(ctx, b, id, secondPage)
	must(err)
	if string(next.Items[0]) == string(messages.Items[0]) {
		t.Fatal("cursor repeated message")
	}
	if _, err = r.DirectMessages(ctx, outsider, id, page); !errors.Is(err, ErrNotFound) {
		t.Fatal("history leak", err)
	}
	if _, err = r.SendDirectMessage(ctx, outsider, id, "Чужое сообщение"); !errors.Is(err, ErrNotFound) {
		t.Fatal("send ACL", err)
	}
	var signals int
	must(db.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE type='realtime.publish' AND payload->'data'->>'type'='chat.changed'`).Scan(&signals))
	if signals != 6 {
		t.Fatalf("transactional signals %d", signals)
	}
	_, err = r.CreateStory(ctx, a, "После пробежки")
	must(err)
	stories, err := r.Stories(ctx, b, page)
	must(err)
	if len(stories.Items) != 1 {
		t.Fatal("friend cannot see story")
	}
	private, err := r.Stories(ctx, outsider, page)
	must(err)
	if len(private.Items) != 0 {
		t.Fatal("story leak")
	}
	_, err = r.Friendship(ctx, a, b.ID, "remove")
	must(err)
	if _, err = r.SendDirectMessage(ctx, b, id, "После удаления"); !errors.Is(err, ErrSocialConflict) {
		t.Fatal("ex-friend can write", err)
	}
	stories, err = r.Stories(ctx, b, page)
	must(err)
	if len(stories.Items) != 0 {
		t.Fatal("ex-friend story leak")
	}
	messages, err = r.DirectMessages(ctx, b, id, page)
	must(err)
	if len(messages.Items) != 1 {
		t.Fatal("lost own history")
	}
	mustExec := func(query string, args ...any) { t.Helper(); _, err := db.Exec(ctx, query, args...); must(err) }
	mustExec(`UPDATE stories SET expires_at=now()-interval '1 second'`)
	stories, err = r.Stories(ctx, a, page)
	must(err)
	if len(stories.Items) != 0 {
		t.Fatal("expired story visible")
	}
}
func TestReviewRulesAndClubMembership(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := isolatedDB(t, ctx)
	r := NewRepository(db)
	s := New(r)
	owner, person, outsider := Actor{ID: "review-owner", Name: "Организатор"}, Actor{ID: "review-user", Name: "Участник"}, Actor{ID: "review-other", Name: "Посторонний"}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []Actor{owner, person, outsider} {
		must(r.EnsureProfile(ctx, a))
	}
	event, err := s.CreateEvent(ctx, owner, EventInput{Title: "Встреча для отзыва", SportName: "Бег отзыв", StartsAt: time.Now().Add(24 * time.Hour), Location: "Парк у реки", JoinMode: "open"})
	must(err)
	_, err = s.Join(ctx, person, event.ID, "", false)
	must(err)
	if _, err = r.CreateReview(ctx, person, event.ID, 5, "Рано"); !errors.Is(err, ErrSocialConflict) {
		t.Fatal("review before end", err)
	}
	_, err = db.Exec(ctx, `UPDATE events SET starts_at=now()-interval '4 hours' WHERE id=$1`, event.ID)
	must(err)
	if _, err = r.CreateReview(ctx, outsider, event.ID, 5, "Не был"); !errors.Is(err, ErrSocialConflict) {
		t.Fatal("outsider review", err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.CreateReview(ctx, person, event.ID, 5, "Спасибо за встречу")
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	success, conflict := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else if errors.Is(err, ErrSocialConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatal("duplicate review", success, conflict)
	}
	page, _ := parseSocialPage("", 10)
	reviews, err := r.ProfileReviews(ctx, outsider, owner.ID, page)
	must(err)
	if len(reviews.Items) != 1 {
		t.Fatal("missing review")
	}
	reviewID := socialObject(t, reviews.Items[0])["id"].(string)
	if err = r.HideReview(ctx, owner, reviewID, "Не нравится"); !errors.Is(err, ErrNotFound) {
		t.Fatal("organizer hides criticism", err)
	}
	must(r.HideReview(ctx, person, reviewID, "Отзываю свой отзыв"))
	reviews, err = r.ProfileReviews(ctx, outsider, owner.ID, page)
	must(err)
	if len(reviews.Items) != 0 {
		t.Fatal("hidden review visible")
	}
	orgID := uuid.NewString()
	_, err = db.Exec(ctx, `INSERT INTO organizations(id,owner_id,name,description,status) VALUES($1,$2,'Беговой клуб','Для друзей','approved')`, orgID, owner.ID)
	must(err)
	must(r.ClubMembership(ctx, person, orgID, true))
	must(r.ClubMembership(ctx, person, orgID, true))
	members, err := r.ClubMembers(ctx, owner, orgID, page)
	must(err)
	if len(members.Items) != 1 {
		t.Fatal("membership duplicate")
	}
	if _, err = r.ClubMembers(ctx, outsider, orgID, page); !errors.Is(err, ErrNotFound) {
		t.Fatal("member list leak", err)
	}
	must(r.ClubMembership(ctx, person, orgID, false))
	members, err = r.ClubMembers(ctx, owner, orgID, page)
	must(err)
	if len(members.Items) != 0 {
		t.Fatal("leave failed")
	}
	chats, err := r.Chats(ctx, person, page)
	must(err)
	if len(chats.Items) != 1 {
		t.Fatal("event chat not automatic", chats)
	}
	if socialObject(t, chats.Items[0])["kind"] != "event" {
		t.Fatal("wrong chat kind")
	}
}

// StoryBlobFake проверяет жизненный цикл метаданных независимо от конкретного S3-провайдера.
type storyBlobFake struct {
	data    map[string][]byte
	deleted []string
}

func (b *storyBlobFake) Put(_ context.Context, key, _ string, data []byte) error {
	b.data[key] = append([]byte(nil), data...)
	return nil
}
func (b *storyBlobFake) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	data, ok := b.data[key]
	if !ok {
		return nil, 0, errors.New("missing object")
	}
	return io.NopCloser(bytes.NewReader(data)), int64(len(data)), nil
}
func (b *storyBlobFake) Delete(_ context.Context, key string) error {
	delete(b.data, key)
	b.deleted = append(b.deleted, key)
	return nil
}
func TestStoryMediaAudienceAndCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	db := isolatedDB(t, ctx)
	r := NewRepository(db)
	owner, friend, other := Actor{ID: "story-owner"}, Actor{ID: "story-friend"}, Actor{ID: "story-other"}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, a := range []Actor{owner, friend, other} {
		must(r.EnsureProfile(ctx, a))
	}
	_, err := r.Friendship(ctx, owner, friend.ID, "request")
	must(err)
	_, err = r.Friendship(ctx, friend, owner.ID, "accept")
	must(err)
	story, err := r.CreateStory(ctx, owner, "Фото нашей команды")
	must(err)
	id := socialObject(t, story)["id"].(string)
	if _, err = r.ReserveStoryMedia(ctx, other, id); !errors.Is(err, ErrNotFound) {
		t.Fatal("stranger attached media", err)
	}
	key, err := r.ReserveStoryMedia(ctx, owner, id)
	must(err)
	blobs := &storyBlobFake{data: map[string][]byte{}}
	must(blobs.Put(ctx, key, "image/png", []byte("picture")))
	must(r.AttachStoryMedia(ctx, owner, id, key, "image/png", 7))
	info, err := r.StoryMediaAccess(ctx, friend, id)
	must(err)
	if info.Key != key || info.Size != 7 {
		t.Fatal("wrong media", info)
	}
	if _, err = r.StoryMediaAccess(ctx, other, id); !errors.Is(err, ErrNotFound) {
		t.Fatal("media leaks", err)
	}
	must(r.CleanupStoryMedia(ctx, blobs))
	if len(blobs.deleted) != 0 {
		t.Fatal("live media deleted")
	}
	// Незавершённая запись не получает ссылку на файл и впоследствии убирается.
	orphanStory, err := r.CreateStory(ctx, owner, "Незавершённая загрузка")
	must(err)
	orphanID := socialObject(t, orphanStory)["id"].(string)
	orphan, err := r.ReserveStoryMedia(ctx, owner, orphanID)
	must(err)
	must(blobs.Put(ctx, orphan, "image/png", []byte("orphan")))
	_, err = db.Exec(ctx, `UPDATE story_media_objects SET created_at=now()-interval '2 hours' WHERE object_key=$1`, orphan)
	must(err)
	must(r.CleanupStoryMedia(ctx, blobs))
	if len(blobs.deleted) != 1 || blobs.deleted[0] != orphan {
		t.Fatal("orphan not cleaned")
	}
	must(r.DeleteStory(ctx, owner, id, "Удаляю историю"))
	if _, err = r.StoryMediaAccess(ctx, friend, id); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted media visible", err)
	}
	must(r.CleanupStoryMedia(ctx, blobs))
	if len(blobs.data) != 0 {
		t.Fatal("deleted story media remains")
	}
	var objects int
	must(db.QueryRow(ctx, `SELECT count(*) FROM story_media_objects`).Scan(&objects))
	if objects != 0 {
		t.Fatal("cleanup ledger remains")
	}
}
