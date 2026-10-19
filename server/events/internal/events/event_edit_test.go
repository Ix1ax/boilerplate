package events

import (
	"errors"
	"testing"
	"time"
)

func TestEditPreservesElapsedQuorumDeadline(t *testing.T) {
	now := time.Now()
	in := validEvent(now)
	two := 2
	past := now.Add(-time.Hour)
	in.MinParticipants, in.QuorumDeadline = &two, &past
	if err := validateEvent(&in, now, false); err != nil {
		t.Fatal(err)
	}
	if !in.QuorumDeadline.Equal(past) {
		t.Fatal("срок изменён без переноса")
	}
	var field FieldError
	if err := ValidateEvent(&in, now); !errors.As(err, &field) || field.Field != "quorum_deadline" {
		t.Fatalf("создание должно отвергать истёкший срок: %v", err)
	}
	one := 1
	in.Capacity = &one
	if err := validateEvent(&in, now, false); !errors.As(err, &field) || field.Field != "min_participants" {
		t.Fatalf("правка должна проверять вместимость: %v", err)
	}
}
