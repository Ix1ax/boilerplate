package events

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestSocialValidation(t *testing.T) {
	for _, n := range []int{-1, 0, 101} {
		if _, err := parseSocialPage("", n); err == nil {
			t.Fatalf("limit %d accepted", n)
		}
	}
	for _, cursor := range []string{"!", strings.Repeat("a", 513), base64.RawURLEncoding.EncodeToString([]byte(`{"at":"2026-10-19T00:00:00Z"}`))} {
		if _, err := parseSocialPage(cursor, 30); err == nil {
			t.Fatal("bad cursor accepted")
		}
	}
	if _, err := socialBody(strings.Repeat("я", 1001), 1000); err == nil {
		t.Fatal("long body accepted")
	}
	if body, err := socialBody("  Привет  ", 1000); err != nil || body != "Привет" {
		t.Fatal(body, err)
	}
	if err := socialPeer(Actor{ID: "a"}, "a"); err == nil {
		t.Fatal("self friendship accepted")
	}
}
