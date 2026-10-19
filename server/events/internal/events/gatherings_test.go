package events

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestValidateQuorum(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	starts := now.Add(48 * time.Hour)
	n := func(v int) *int { return &v }
	at := func(d time.Duration) *time.Time { v := now.Add(d); return &v }
	cases := []struct {
		name     string
		min, cap *int
		deadline *time.Time
		field    string
	}{
		{"без минимума", nil, nil, nil, ""},
		{"срок без минимума", nil, nil, at(time.Hour), "min_participants"},
		{"минимум один", n(1), nil, nil, "min_participants"},
		{"минимум больше мест", n(12), n(10), nil, "min_participants"},
		{"срок в прошлом", n(6), nil, at(-time.Minute), "quorum_deadline"},
		{"срок после начала", n(6), nil, at(49 * time.Hour), "quorum_deadline"},
		{"корректно", n(10), n(14), at(24 * time.Hour), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := EventInput{StartsAt: starts, MinParticipants: tc.min, Capacity: tc.cap, QuorumDeadline: tc.deadline}
			err := validateQuorum(&in, now)
			var fe FieldError
			switch {
			case tc.field == "" && err != nil:
				t.Fatal(err)
			case tc.field != "" && (!errors.As(err, &fe) || fe.Field != tc.field):
				t.Fatalf("ожидалась ошибка поля %s, получено %v", tc.field, err)
			}
		})
	}
}

func TestValidateQuorumDefaultDeadline(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	six := 6
	in := EventInput{StartsAt: now.Add(48 * time.Hour), MinParticipants: &six}
	if err := validateQuorum(&in, now); err != nil || !in.QuorumDeadline.Equal(in.StartsAt.Add(-3*time.Hour)) {
		t.Fatalf("срок по умолчанию: %v %v", in.QuorumDeadline, err)
	}
	// Встреча через час: за 3 часа уже поздно, срок — через 30 минут.
	soon := EventInput{StartsAt: now.Add(time.Hour), MinParticipants: &six}
	if err := validateQuorum(&soon, now); err != nil || !soon.QuorumDeadline.Equal(now.Add(30*time.Minute)) {
		t.Fatalf("срок для близкой встречи: %v %v", soon.QuorumDeadline, err)
	}
}

func TestReminderBody(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, localZone)
	tgt := ReminderTarget{Title: "Пробежка", Location: "Остров Татышев", PrivateInfo: "У главного входа", Requirements: "Вода", StartsAt: time.Date(2026, 10, 24, 10, 0, 0, 0, localZone)}
	body := reminderBody(tgt, "+4°, дождь — возьмите дождевик", "Игорь", "", now)
	for _, want := range []string{"Завтра в 10:00 · Остров Татышев.", "Точка сбора: У главного входа", "Возьмите: Вода", "Погода: +4°, дождь", "Вас встретит Игорь."} {
		if !strings.Contains(body, want) {
			t.Errorf("нет %q в\n%s", want, body)
		}
	}
	bare := reminderBody(ReminderTarget{Location: "Зал", StartsAt: now.Add(3 * time.Hour)}, "", "", "Анна, Пётр", now)
	if bare != "Сегодня в 15:00 · Зал.\nВы встречаете: Анна, Пётр." {
		t.Fatal(bare)
	}
	if got := whenText(time.Date(2026, 10, 30, 9, 5, 0, 0, localZone), now); got != "пт, 30 октября в 09:05" {
		t.Fatal(got)
	}
}
