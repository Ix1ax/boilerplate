package events

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestVideoPermissionsAndWindow(t *testing.T) {
	now := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	cfg := VideoConfig{"wss://video.example.test", "key", strings.Repeat("s", 32)}
	base := VideoAccess{EventFormat: EventFormat{"28b17314-b332-42d0-b867-b2de5747dbbd", "online"}, OrganizerID: "owner", Status: "published", StartsAt: now, Confirmed: true}
	for _, tc := range []struct {
		name  string
		actor Actor
		alter func(*VideoAccess)
		at    time.Time
		want  error
	}{
		{"confirmed", Actor{ID: "member"}, nil, now, nil},
		{"organizer", Actor{ID: "owner"}, func(v *VideoAccess) { v.Confirmed = false }, now, nil},
		{"admin_not_member", Actor{ID: "admin", Admin: true}, func(v *VideoAccess) { v.Confirmed = false }, now, ErrNotParticipant},
		{"anonymous", Actor{}, nil, now, ErrNotParticipant},
		{"pending", Actor{ID: "pending"}, func(v *VideoAccess) { v.Confirmed = false }, now, ErrNotParticipant},
		{"offline", Actor{ID: "member"}, func(v *VideoAccess) { v.Format = "offline" }, now, ErrVideoOffline},
		{"cancelled", Actor{ID: "owner"}, func(v *VideoAccess) { v.Status = "cancelled" }, now, ErrEventClosed},
		{"unpublished", Actor{ID: "member"}, func(v *VideoAccess) { v.Status = "unpublished" }, now, ErrEventClosed},
		{"early", Actor{ID: "member"}, nil, now.Add(-15*time.Minute - time.Nanosecond), ErrVideoWindow},
		{"opening_boundary", Actor{ID: "member"}, nil, now.Add(-15 * time.Minute), nil},
		{"ended", Actor{ID: "member"}, nil, now.Add(90 * time.Minute), ErrVideoWindow},
		{"explicit_end", Actor{ID: "member"}, func(v *VideoAccess) { e := now.Add(time.Hour); v.EndsAt = &e }, now.Add(time.Hour), ErrVideoWindow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := base
			if tc.alter != nil {
				tc.alter(&v)
			}
			_, err := IssueVideoToken(cfg, tc.actor, v, tc.at)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestVideoTokenGrantsAndSignature(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	end := now.Add(30 * time.Second)
	cfg := VideoConfig{"wss://video.example.test", "key", strings.Repeat("s", 32)}
	v := VideoAccess{EventFormat: EventFormat{"id", "online"}, OrganizerID: "owner", Status: "published", StartsAt: now, EndsAt: &end, Confirmed: true}
	out, err := IssueVideoToken(cfg, Actor{ID: "member", Name: "Олег"}, v, now)
	if err != nil {
		t.Fatal(err)
	}
	if !out.ExpiresAt.Equal(end) {
		t.Fatal("token outlives meeting")
	}
	parts := strings.Split(out.Token, ".")
	mac := hmac.New(sha256.New, []byte(cfg.APISecret))
	_, _ = mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if !hmac.Equal(mac.Sum(nil), sig) {
		t.Fatal("bad signature")
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims struct {
		Iss   string         `json:"iss"`
		Sub   string         `json:"sub"`
		Exp   int64          `json:"exp"`
		Video map[string]any `json:"video"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatal(err)
	}
	if claims.Iss != "key" || claims.Sub != "member" || claims.Exp != end.Unix() || claims.Video["room"] != "event_id" || claims.Video["roomJoin"] != true {
		t.Fatalf("bad claims: %s", payload)
	}
	for _, grant := range []string{"roomAdmin", "roomCreate", "roomList", "roomRecord", "ingressAdmin"} {
		if _, ok := claims.Video[grant]; ok {
			t.Fatalf("unexpected grant %s", grant)
		}
	}
	if claims.Video["canPublishData"] != false {
		t.Fatal("unmoderated data channel permitted")
	}
}

func TestVideoConfig(t *testing.T) {
	good := VideoConfig{"wss://video.example.test", "key", strings.Repeat("s", 32)}
	for _, u := range []string{"http://example.test", "ws://example.test", "wss://user:pass@example.test", "wss://example.test?secret=value", ""} {
		cfg := good
		cfg.URL = u
		if cfg.Valid() {
			t.Fatalf("accepted %q", u)
		}
	}
	if !good.Valid() {
		t.Fatal("valid config rejected")
	}
	good.APISecret = ""
	if good.Valid() {
		t.Fatal("empty secret")
	}
}
