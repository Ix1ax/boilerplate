// Package events — домен кейса «Платформа совместных спортивных активностей»: справочник
// направлений, организации с допуском администратором, мероприятия с двумя режимами записи,
// заявки участников и обсуждение. Модель и правила доступа — docs/data-model.md.
//
// Правила, от которых зависят баллы сценариев (допуск организации, запись только на
// опубликованное будущее событие, решение по заявке только организатором), проверяются
// в SQL внутри одной транзакции (repository.go), а не в клиенте и не отдельным запросом.
package events

import (
	"errors"
	"time"

	"example.com/hackathon/server/events/internal/platform"
)

var (
	ErrNotFound       = errors.New("Не найдено")
	ErrValidation     = errors.New("Проверьте поля формы")
	ErrOrgNotApproved = errors.New("Организация ещё не допущена администратором: публиковать события от её имени пока нельзя")
	ErrOrgExists      = errors.New("Заявка организации уже подана")
	ErrEventClosed    = errors.New("Мероприятие недоступно для записи: оно снято с публикации или отменено")
	ErrEventPast      = errors.New("Мероприятие уже началось или прошло")
	ErrOwnEvent       = errors.New("Вы организатор этого мероприятия")
	ErrFull           = errors.New("Свободных мест нет")
	ErrNotOrganizer   = errors.New("Решение по заявкам принимает только организатор мероприятия")
	ErrBadParent      = errors.New("Отвечать можно только на сообщение верхнего уровня этого обсуждения")
	ErrNotParticipant = errors.New("Вы не записаны на это мероприятие")
	ErrSportHidden    = errors.New("Это направление недоступно, выберите другое")
)

// FieldError — ошибка конкретного поля: клиент подсвечивает его в форме.
type FieldError struct{ Field, Message string }

func (e FieldError) Error() string { return e.Message }
func (e FieldError) Unwrap() error { return ErrValidation }

// Переходы статусов. Причина обязательна там, где её должен увидеть другой человек:
// отказ организации, снятие и удаление события админом, отмена организатором.
var (
	orgWorkflow = platform.NewWorkflow(map[string][]string{
		"pending":  {"approved", "rejected"},
		"rejected": {"pending"},
	}).RequireComment("rejected")
	eventWorkflow = platform.NewWorkflow(map[string][]string{
		"published":   {"unpublished", "deleted", "cancelled"},
		"unpublished": {"published", "deleted"},
	}).RequireComment("unpublished", "deleted", "cancelled")
)

// Actor — кто выполняет запрос (личность проверил gateway).
type Actor struct {
	ID, Name, Email string
	Admin           bool
}

type Sport struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsCustom bool   `json:"is_custom"`
}

// Organizer — кто проводит событие: человек (ссылка на /profiles/{id}) или организация
// (ссылка на /organizations/{id}).
type Organizer struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MyParticipation — запись текущего пользователя на событие.
type MyParticipation struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	DecisionReason string `json:"decision_reason"`
	// FirstTime — «иду впервые»; IsBuddy — «готов встретить новичка».
	FirstTime bool `json:"first_time"`
	IsBuddy   bool `json:"is_buddy"`
	// Buddy — кто встретит меня, если я новичок; Newcomers — кого встречаю я (только в карточке).
	Buddy     *Person  `json:"buddy"`
	Newcomers []Person `json:"newcomers,omitempty"`
}

type Event struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Sport        Sport      `json:"sport"`
	StartsAt     time.Time  `json:"starts_at"`
	EndsAt       *time.Time `json:"ends_at"`
	Location     string     `json:"location"`
	Lat          *float64   `json:"lat"`
	Lng          *float64   `json:"lng"`
	Level        string     `json:"level"`
	Requirements string     `json:"requirements"`
	PriceNote    string     `json:"price_note"`
	// PrivateInfo видят только организатор, подтверждённые участники и админ (иначе пусто).
	PrivateInfo    string `json:"private_info"`
	HasPrivateInfo bool   `json:"has_private_info"`
	JoinMode       string `json:"join_mode"`
	Capacity       *int   `json:"capacity"`
	ConfirmedCount int    `json:"confirmed_count"`
	PendingCount   int    `json:"pending_count"`
	WaitlistCount  int    `json:"waitlist_count"`
	// CompanyMatches — сколько людей из «Ищу компанию» получили это событие в подборке.
	CompanyMatches int `json:"company_matches"`
	// SpotsLeft — свободные места при лимите (nil — без ограничения).
	SpotsLeft *int `json:"spots_left"`
	// FirstTimersCount и BuddiesCount — подтверждённые новички и готовые их встретить.
	FirstTimersCount int `json:"first_timers_count"`
	BuddiesCount     int `json:"buddies_count"`
	// «Соберём состав»: минимум подтверждённых участников к сроку, иначе автоотмена.
	MinParticipants *int       `json:"min_participants"`
	QuorumDeadline  *time.Time `json:"quorum_deadline"`
	QuorumStatus    string     `json:"quorum_status"`
	// Outdoor — занятие на улице: для него есть погода (GET /events/{id}/weather).
	Outdoor          bool             `json:"outdoor"`
	Status           string           `json:"status"`
	ModerationReason string           `json:"moderation_reason"`
	Organizer        Organizer        `json:"organizer"`
	IsOrganizer      bool             `json:"is_organizer"`
	MyParticipation  *MyParticipation `json:"my_participation"`
	CreatedAt        time.Time        `json:"created_at"`
}

type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Person struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level string `json:"level"`
}

// Participant — запись или заявка в списке организатора.
type Participant struct {
	ID             string    `json:"id"`
	User           Person    `json:"user"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	DecisionReason string    `json:"decision_reason"`
	CreatedAt      time.Time `json:"created_at"`
	FirstTime      bool      `json:"first_time"`
	IsBuddy        bool      `json:"is_buddy"`
	// Buddy — кто встречает этого новичка (видит организатор).
	Buddy *Person `json:"buddy"`
}

// MyEntry — запись пользователя вместе с кратким описанием события («Мои записи»).
type MyEntry struct {
	MyParticipation
	Event EventBrief `json:"event"`
}

type EventBrief struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Sport            string    `json:"sport"`
	StartsAt         time.Time `json:"starts_at"`
	Location         string    `json:"location"`
	JoinMode         string    `json:"join_mode"`
	Status           string    `json:"status"`
	ModerationReason string    `json:"moderation_reason"`
}

type Comment struct {
	ID       string  `json:"id"`
	EventID  string  `json:"event_id"`
	ParentID *string `json:"parent_id"`
	Author   Person  `json:"author"`
	// IsOrganizer — автор организует событие; OrganizationName — от чьего имени он отвечает.
	IsOrganizer      bool      `json:"is_organizer"`
	OrganizationName string    `json:"organization_name"`
	Body             string    `json:"body"`
	Status           string    `json:"status"`
	HiddenReason     string    `json:"hidden_reason"`
	CreatedAt        time.Time `json:"created_at"`
}

type Organization struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	City        string  `json:"city"`
	Contacts    string  `json:"contacts"`
	Sports      []Sport `json:"sports"`
	Status      string  `json:"status"`
	// ReviewReason — причина отказа; видна представителю и админу.
	ReviewReason string     `json:"review_reason"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
	Owner        *Person    `json:"owner,omitempty"`
	EventsCount  int        `json:"events_count"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Profile struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	About  string  `json:"about"`
	City   string  `json:"city"`
	Level  string  `json:"level"`
	Events []Event `json:"events"`
}

// EventInput — поля формы создания. Направление — SportID из справочника или SportName
// (своё, заводится в справочнике без правки кода).
type EventInput struct {
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	SportID        string     `json:"sport_id"`
	SportName      string     `json:"sport_name"`
	StartsAt       time.Time  `json:"starts_at"`
	EndsAt         *time.Time `json:"ends_at"`
	Location       string     `json:"location"`
	Lat            *float64   `json:"lat"`
	Lng            *float64   `json:"lng"`
	Level          string     `json:"level"`
	Requirements   string     `json:"requirements"`
	PriceNote      string     `json:"price_note"`
	PrivateInfo    string     `json:"private_info"`
	JoinMode       string     `json:"join_mode"`
	Capacity       *int       `json:"capacity"`
	AsOrganization bool       `json:"as_organization"`
	// MinParticipants и QuorumDeadline — «Соберём состав»; срок по умолчанию — за 3 часа до начала.
	MinParticipants *int       `json:"min_participants"`
	QuorumDeadline  *time.Time `json:"quorum_deadline"`
	Outdoor         bool       `json:"outdoor"`
}

type OrganizationInput struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	City        string   `json:"city"`
	Contacts    string   `json:"contacts"`
	SportIDs    []string `json:"sport_ids"`
}

// EventFilter — параметры выдачи. Level «beginner» показывает и события «для всех».
type EventFilter struct {
	Query, Level, Mode, OrganizationID, OrganizerID string
	// MatchedFor — id моего запроса «Ищу компанию»: только подобранные под него события.
	MatchedFor string
	SportIDs   []string
	From, To   *time.Time
	// Statuses пусто — только опубликованные будущие (пользовательская выдача).
	Statuses      []string
	Limit, Offset int
}
