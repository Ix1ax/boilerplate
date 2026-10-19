package events

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Store — хранилище домена. Все проверки «кто может» выполняются внутри его методов в SQL
// (см. repository.go): Service только валидирует ввод и передаёт личность.
type Store interface {
	EnsureProfile(ctx context.Context, a Actor) error
	Sports(ctx context.Context) ([]Sport, error)
	ListEvents(ctx context.Context, a Actor, f EventFilter) (Page[Event], error)
	GetEvent(ctx context.Context, a Actor, id string) (Event, error)
	CreateEvent(ctx context.Context, a Actor, in EventInput) (Event, error)
	CancelEvent(ctx context.Context, a Actor, id, reason string) (Event, error)
	Join(ctx context.Context, a Actor, eventID, message string, firstTime bool) (MyParticipation, error)
	SetBuddy(ctx context.Context, a Actor, eventID string, on bool) (MyParticipation, error)
	FailQuorums(ctx context.Context) error
	DueReminders(ctx context.Context) ([]ReminderTarget, error)
	ReminderTargetFor(ctx context.Context, a Actor, eventID string) (ReminderTarget, error)
	SendReminder(ctx context.Context, t ReminderTarget, kind, weatherLine string) (int, error)
	AdminOrganization(ctx context.Context, a Actor, id string) (OrganizationDetail, error)
	EventHistory(ctx context.Context, a Actor, id string) ([]HistoryEntry, error)
	AdminSports(ctx context.Context) ([]AdminSport, error)
	SetSportHidden(ctx context.Context, a Actor, id string, hidden bool, reason string) (AdminSport, error)
	RepublishDomain(ctx context.Context) (int, error)
	UpdateProfile(ctx context.Context, a Actor, p ProfilePatch) (Profile, error)
	OrganizationStats(ctx context.Context, id string) (OrganizerStats, error)
	UpdateEvent(ctx context.Context, a Actor, id string, in EventInput, changed EventChanges) (Event, error)
	Alternatives(ctx context.Context, a Actor, id string, limit int) ([]Event, error)
	Leave(ctx context.Context, a Actor, eventID string) error
	Participants(ctx context.Context, a Actor, eventID string) ([]Participant, error)
	Decide(ctx context.Context, a Actor, eventID, participationID string, approve bool, reason string) (Participant, error)
	Comments(ctx context.Context, a Actor, eventID string) ([]Comment, error)
	AddComment(ctx context.Context, a Actor, eventID string, parentID *string, body string) (Comment, error)
	MyParticipations(ctx context.Context, a Actor) ([]MyEntry, error)
	MyOrganization(ctx context.Context, a Actor) (Organization, error)
	SubmitOrganization(ctx context.Context, a Actor, in OrganizationInput) (Organization, error)
	UpdateOrganization(ctx context.Context, a Actor, in OrganizationInput) (Organization, error)
	ListOrganizations(ctx context.Context, a Actor, q string, statuses []string, limit, offset int) (Page[Organization], error)
	GetOrganization(ctx context.Context, a Actor, id string) (Organization, error)
	ReviewOrganization(ctx context.Context, a Actor, id, to, reason string) (Organization, error)
	ModerateEvent(ctx context.Context, a Actor, id, to, reason string) (Event, error)
	HideComment(ctx context.Context, a Actor, id, reason string) (Comment, error)
	GetProfile(ctx context.Context, a Actor, id string) (Profile, error)
	CreateCompanyRequest(ctx context.Context, a Actor, in CompanyRequestInput) (CompanyRequest, error)
	MyCompanyRequest(ctx context.Context, a Actor, id string) (CompanyRequest, error)
	MyCompanyRequests(ctx context.Context, a Actor) ([]CompanyRequest, error)
	CloseCompanyRequest(ctx context.Context, a Actor, id string) error
	Demand(ctx context.Context, a Actor, sportID, sportName, level string) (Demand, error)
}

// Service — валидация ввода поверх Store. Weather необязателен: без него погода не показывается.
type Service struct {
	Store
	Weather Forecaster
}

func New(store Store) *Service { return &Service{Store: store} }

var levels = map[string]bool{"any": true, "beginner": true, "amateur": true, "pro": true}

// ValidateEvent нормализует и проверяет форму события. now передаётся явно ради тестов.
func ValidateEvent(in *EventInput, now time.Time) error {
	in.Title, in.Description, in.Location = strings.TrimSpace(in.Title), strings.TrimSpace(in.Description), strings.TrimSpace(in.Location)
	in.SportID, in.SportName = strings.TrimSpace(in.SportID), strings.Join(strings.Fields(in.SportName), " ")
	in.Requirements, in.PriceNote, in.PrivateInfo = strings.TrimSpace(in.Requirements), strings.TrimSpace(in.PriceNote), strings.TrimSpace(in.PrivateInfo)
	if in.Level == "" {
		in.Level = "any"
	}
	switch {
	case !between(in.Title, 3, 120):
		return FieldError{"title", "Название — от 3 до 120 символов"}
	case utf8.RuneCountInString(in.Description) > 5000:
		return FieldError{"description", "Описание — не длиннее 5000 символов"}
	case in.SportID == "" && in.SportName == "":
		return FieldError{"sport", "Выберите направление или впишите своё"}
	case in.SportID != "" && uuid.Validate(in.SportID) != nil:
		return FieldError{"sport", "Неизвестное направление"}
	case in.SportID == "" && !between(in.SportName, 2, 60):
		return FieldError{"sport", "Название направления — от 2 до 60 символов"}
	case in.StartsAt.IsZero() || !in.StartsAt.After(now):
		return FieldError{"starts_at", "Укажите дату и время в будущем"}
	case in.EndsAt != nil && !in.EndsAt.After(in.StartsAt):
		return FieldError{"ends_at", "Окончание должно быть позже начала"}
	case !between(in.Location, 3, 300):
		return FieldError{"location", "Укажите адрес или понятное описание места"}
	case (in.Lat == nil) != (in.Lng == nil) || in.Lat != nil && (*in.Lat < -90 || *in.Lat > 90 || *in.Lng < -180 || *in.Lng > 180):
		return FieldError{"location", "Некорректные координаты"}
	case !levels[in.Level]:
		return FieldError{"level", "Уровень: any, beginner, amateur или pro"}
	case utf8.RuneCountInString(in.Requirements) > 2000 || utf8.RuneCountInString(in.PrivateInfo) > 2000:
		return FieldError{"requirements", "Условия — не длиннее 2000 символов"}
	case utf8.RuneCountInString(in.PriceNote) > 200:
		return FieldError{"price_note", "Стоимость — не длиннее 200 символов"}
	case in.JoinMode != "open" && in.JoinMode != "request":
		return FieldError{"join_mode", "Выберите режим записи: свободно или по заявке"}
	case in.Capacity != nil && (*in.Capacity < 1 || *in.Capacity > 10000):
		return FieldError{"capacity", "Мест — от 1 до 10000 или без ограничения"}
	}
	return validateQuorum(in, now)
}

// quorumLead — срок сбора состава по умолчанию: за 3 часа до начала успевают и отменить, и предупредить.
// quorumMinLead — позже собирать бессмысленно: люди уже в пути.
const (
	quorumLead    = 3 * time.Hour
	quorumMinLead = 30 * time.Minute
)

// validateQuorum проверяет «Соберём состав»: минимум не больше лимита мест, срок — между сейчас и началом.
func validateQuorum(in *EventInput, now time.Time) error {
	if in.MinParticipants == nil {
		if in.QuorumDeadline != nil {
			return FieldError{"min_participants", "Срок сбора состава задаётся вместе с минимумом участников"}
		}
		return nil
	}
	if *in.MinParticipants < 2 || *in.MinParticipants > 1000 {
		return FieldError{"min_participants", "Минимум участников — от 2 до 1000"}
	}
	if in.Capacity != nil && *in.MinParticipants > *in.Capacity {
		return FieldError{"min_participants", "Минимум не может быть больше числа мест"}
	}
	if in.QuorumDeadline == nil {
		deadline := in.StartsAt.Add(-quorumLead)
		if earliest := now.Add(quorumMinLead); deadline.Before(earliest) {
			deadline = earliest
		}
		in.QuorumDeadline = &deadline
	}
	switch {
	case !in.QuorumDeadline.After(now):
		return FieldError{"quorum_deadline", "Срок сбора состава должен быть в будущем"}
	case in.QuorumDeadline.After(in.StartsAt):
		return FieldError{"quorum_deadline", "Состав нужно собрать до начала мероприятия"}
	}
	return nil
}

func ValidateOrganization(in *OrganizationInput) error {
	in.Name, in.Description = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	in.City, in.Contacts = strings.TrimSpace(in.City), strings.TrimSpace(in.Contacts)
	switch {
	case !between(in.Name, 2, 120):
		return FieldError{"name", "Название — от 2 до 120 символов"}
	case !between(in.Description, 10, 3000):
		return FieldError{"description", "Расскажите о деятельности: от 10 до 3000 символов"}
	case utf8.RuneCountInString(in.City) > 100 || utf8.RuneCountInString(in.Contacts) > 500:
		return FieldError{"contacts", "Слишком длинные город или контакты"}
	case len(in.SportIDs) > 20:
		return FieldError{"sport_ids", "Не больше 20 направлений"}
	}
	for _, id := range in.SportIDs {
		if uuid.Validate(id) != nil {
			return FieldError{"sport_ids", "Неизвестное направление"}
		}
	}
	return nil
}

func (s *Service) CreateEvent(ctx context.Context, a Actor, in EventInput) (Event, error) {
	if err := ValidateEvent(&in, time.Now()); err != nil {
		return Event{}, err
	}
	return s.Store.CreateEvent(ctx, a, in)
}

func (s *Service) Join(ctx context.Context, a Actor, eventID, message string, firstTime bool) (MyParticipation, error) {
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) > 500 {
		return MyParticipation{}, FieldError{"message", "Сообщение к заявке — не длиннее 500 символов"}
	}
	return s.Store.Join(ctx, a, eventID, message, firstTime)
}

func (s *Service) Decide(ctx context.Context, a Actor, eventID, pid string, approve bool, reason string) (Participant, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 500 {
		return Participant{}, FieldError{"reason", "Комментарий — не длиннее 500 символов"}
	}
	return s.Store.Decide(ctx, a, eventID, pid, approve, reason)
}

func (s *Service) AddComment(ctx context.Context, a Actor, eventID string, parentID *string, body string) (Comment, error) {
	body = strings.TrimSpace(body)
	if !between(body, 1, 2000) {
		return Comment{}, FieldError{"body", "Сообщение — от 1 до 2000 символов"}
	}
	if parentID != nil && uuid.Validate(*parentID) != nil {
		return Comment{}, FieldError{"parent_id", "Некорректный идентификатор сообщения"}
	}
	return s.Store.AddComment(ctx, a, eventID, parentID, body)
}

func (s *Service) SubmitOrganization(ctx context.Context, a Actor, in OrganizationInput) (Organization, error) {
	if err := ValidateOrganization(&in); err != nil {
		return Organization{}, err
	}
	return s.Store.SubmitOrganization(ctx, a, in)
}

func (s *Service) UpdateOrganization(ctx context.Context, a Actor, in OrganizationInput) (Organization, error) {
	if err := ValidateOrganization(&in); err != nil {
		return Organization{}, err
	}
	return s.Store.UpdateOrganization(ctx, a, in)
}

// Причина решения: проверка обязательности — в Workflow, здесь только длина.
func cleanReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 1000 {
		return "", FieldError{"reason", "Причина — не длиннее 1000 символов"}
	}
	return reason, nil
}

func (s *Service) ReviewOrganization(ctx context.Context, a Actor, id, to, reason string) (Organization, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Organization{}, err
	}
	return s.Store.ReviewOrganization(ctx, a, id, to, reason)
}

func (s *Service) ModerateEvent(ctx context.Context, a Actor, id, to, reason string) (Event, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Event{}, err
	}
	return s.Store.ModerateEvent(ctx, a, id, to, reason)
}

func (s *Service) CancelEvent(ctx context.Context, a Actor, id, reason string) (Event, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Event{}, err
	}
	return s.Store.CancelEvent(ctx, a, id, reason)
}

func (s *Service) HideComment(ctx context.Context, a Actor, id, reason string) (Comment, error) {
	reason, err := cleanReason(reason)
	if err != nil {
		return Comment{}, err
	}
	if reason == "" {
		return Comment{}, FieldError{"reason", "Укажите причину: её увидит автор сообщения"}
	}
	return s.Store.HideComment(ctx, a, id, reason)
}

func between(s string, min, max int) bool {
	n := utf8.RuneCountInString(s)
	return utf8.ValidString(s) && n >= min && n <= max
}
