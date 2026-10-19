//go:build integration

package events

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"example.com/hackathon/server/events/internal/database"
	"example.com/hackathon/server/events/internal/platform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Шесть базовых сценариев кейса (docs/case.md) на уровне хранилища: правила доступа живут в SQL,
// поэтому проверяем их на настоящем Postgres, а не на фейке.
func TestCaseScenarios(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Fatal("Укажите DATABASE_URL для интеграционного теста")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := database.Migrate(ctx, dsn, "up"); err != nil {
		t.Fatal(err)
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(NewRepository(db))
	person := func(name string) Actor {
		a := Actor{ID: "it-" + uuid.NewString(), Name: name}
		if err := s.EnsureProfile(ctx, a); err != nil {
			t.Fatal(err)
		}
		return a
	}
	anna, igor, oleg, admin := person("Анна"), person("Игорь"), person("Олег"), person("Модератор")
	admin.Admin = true
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	expect := func(err, want error) {
		t.Helper()
		if !errors.Is(err, want) {
			t.Fatalf("ожидалась ошибка %v, получено %v", want, err)
		}
	}
	future := time.Now().Add(48 * time.Hour)
	sportName := "Тестовый спорт " + uuid.NewString()[:8]

	// Сценарий 1: своё направление без правки кода, событие видно другому пользователю.
	run, err := s.CreateEvent(ctx, anna, EventInput{Title: "Фрисби в парке", SportName: sportName, StartsAt: future, Location: "Остров Татышев", JoinMode: "open", Capacity: ptr(1)})
	must(err)
	if !run.Sport.IsCustom || run.Organizer.Type != "user" || run.Organizer.Name != "Анна" {
		t.Fatalf("сценарий 1: %+v", run)
	}
	again, err := s.CreateEvent(ctx, igor, EventInput{Title: "Ещё фрисби", SportName: "  " + sportName + " ", StartsAt: future, Location: "Парк", JoinMode: "open"})
	must(err)
	if again.Sport.ID != run.Sport.ID {
		t.Fatal("сценарий 1: одинаковое направление завело дубль в справочнике")
	}
	page, err := s.ListEvents(ctx, igor, EventFilter{Query: sportName, Limit: 10})
	must(err)
	if page.Total != 2 {
		t.Fatalf("сценарий 1: поиск по направлению нашёл %d", page.Total)
	}

	// Сценарий 2: до допуска публиковать от организации нельзя, после — сразу.
	_, err = s.CreateEvent(ctx, oleg, EventInput{Title: "Турнир", SportID: run.Sport.ID, StartsAt: future, Location: "Клуб", JoinMode: "request", AsOrganization: true})
	expect(err, ErrOrgNotApproved)
	org, err := s.SubmitOrganization(ctx, oleg, OrganizationInput{Name: "Ладья " + uuid.NewString()[:6], Description: "Шахматный клуб для всех"})
	must(err)
	_, err = s.SubmitOrganization(ctx, oleg, OrganizationInput{Name: "Ещё одна", Description: "Шахматный клуб для всех"})
	expect(err, ErrOrgExists)
	if _, err = s.GetOrganization(ctx, anna, org.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("сценарий 2: страница недопущенной организации видна посторонним")
	}
	_, err = s.ReviewOrganization(ctx, admin, org.ID, "rejected", "")
	expect(err, platform.ErrCommentRequired)
	rejected, err := s.ReviewOrganization(ctx, admin, org.ID, "rejected", "Нет контактов")
	must(err)
	mine, err := s.MyOrganization(ctx, oleg)
	must(err)
	if rejected.Status != "rejected" || mine.ReviewReason != "Нет контактов" {
		t.Fatalf("сценарий 2: представитель не видит причину отказа: %+v", mine)
	}
	resubmitted, err := s.UpdateOrganization(ctx, oleg, OrganizationInput{Name: org.Name, Description: "Шахматный клуб, контакты: +7 900", Contacts: "+7 900"})
	must(err)
	if resubmitted.Status != "pending" {
		t.Fatalf("сценарий 2: правка после отказа не вернула заявку на рассмотрение: %s", resubmitted.Status)
	}
	_, err = s.ReviewOrganization(ctx, admin, org.ID, "approved", "")
	must(err)
	tournament, err := s.CreateEvent(ctx, oleg, EventInput{Title: "Турнир выходного дня", SportID: run.Sport.ID, StartsAt: future, Location: "Клуб", JoinMode: "request", Capacity: ptr(1), PrivateInfo: "Код домофона 24", AsOrganization: true})
	must(err)
	if tournament.Status != "published" || tournament.Organizer.Type != "organization" || tournament.Organizer.ID != org.ID {
		t.Fatalf("сценарий 2: %+v", tournament)
	}
	orgEvents, err := s.ListEvents(ctx, anna, EventFilter{OrganizationID: org.ID, Limit: 10})
	must(err)
	if orgEvents.Total != 1 {
		t.Fatal("сценарий 2: пользователь не видит событие организации")
	}

	// Сценарий 3: свободная запись — сразу участник; лимит мест — лист ожидания и автоподтверждение.
	joined, err := s.Join(ctx, igor, run.ID, "")
	must(err)
	if joined.Status != "confirmed" {
		t.Fatalf("сценарий 3: %s", joined.Status)
	}
	if again, err := s.Join(ctx, igor, run.ID, ""); err != nil || again.ID != joined.ID {
		t.Fatal("сценарий 3: повторная запись создала дубль")
	}
	waiting, err := s.Join(ctx, oleg, run.ID, "")
	must(err)
	if waiting.Status != "waitlist" {
		t.Fatalf("сценарий 3: при полном лимите ожидался лист ожидания, получено %s", waiting.Status)
	}
	_, err = s.Join(ctx, anna, run.ID, "")
	expect(err, ErrOwnEvent)
	people, err := s.Participants(ctx, anna, run.ID)
	must(err)
	if len(people) != 2 || people[0].User.Name != "Игорь" {
		t.Fatalf("сценарий 3: организатор не видит участника: %+v", people)
	}
	must(s.Leave(ctx, igor, run.ID))
	promoted, err := s.GetEvent(ctx, oleg, run.ID)
	must(err)
	if promoted.MyParticipation == nil || promoted.MyParticipation.Status != "confirmed" {
		t.Fatal("сценарий 3: место не перешло первому из листа ожидания")
	}

	// Сценарий 4: заявка ждёт решения; решает только организатор; принятие и отказ.
	application, err := s.Join(ctx, anna, tournament.ID, "Играю два года")
	must(err)
	viewed, err := s.GetEvent(ctx, anna, tournament.ID)
	must(err)
	if application.Status != "pending" || viewed.PrivateInfo != "" || !viewed.HasPrivateInfo {
		t.Fatal("сценарий 4: до принятия закрытая информация видна или статус неверный")
	}
	_, err = s.Decide(ctx, admin, tournament.ID, application.ID, true, "")
	expect(err, ErrNotOrganizer)
	_, err = s.Decide(ctx, oleg, tournament.ID, application.ID, true, "")
	must(err)
	viewed, err = s.GetEvent(ctx, anna, tournament.ID)
	must(err)
	if viewed.MyParticipation.Status != "confirmed" || viewed.PrivateInfo == "" {
		t.Fatal("сценарий 4: после принятия статус или закрытая информация не обновились")
	}
	second, err := s.Join(ctx, igor, tournament.ID, "")
	must(err)
	_, err = s.Decide(ctx, oleg, tournament.ID, second.ID, true, "")
	expect(err, ErrFull)
	rejectedP, err := s.Decide(ctx, oleg, tournament.ID, second.ID, false, "Мест нет")
	must(err)
	if rejectedP.Status != "rejected" || rejectedP.DecisionReason != "Мест нет" {
		t.Fatalf("сценарий 4: отказ: %+v", rejectedP)
	}
	_, err = s.Decide(ctx, oleg, tournament.ID, second.ID, true, "")
	expect(err, platform.ErrTransition)

	// Сценарий 5: вопрос и ответ организатора различимы по авторству.
	question, err := s.AddComment(ctx, anna, tournament.ID, nil, "Нужна ли своя доска?")
	must(err)
	answer, err := s.AddComment(ctx, oleg, tournament.ID, &question.ID, "Нет, доски есть")
	must(err)
	_, err = s.AddComment(ctx, anna, tournament.ID, &answer.ID, "Ответ на ответ")
	expect(err, ErrBadParent)
	thread, err := s.Comments(ctx, igor, tournament.ID)
	must(err)
	if len(thread) != 2 || thread[0].IsOrganizer || !thread[1].IsOrganizer || thread[1].OrganizationName != org.Name {
		t.Fatalf("сценарий 5: %+v", thread)
	}
	hidden, err := s.HideComment(ctx, admin, question.ID, "Проверка")
	must(err)
	thread, err = s.Comments(ctx, igor, tournament.ID)
	must(err)
	if hidden.Status != "hidden" || thread[0].Body != "" || len(thread) != 2 {
		t.Fatal("сценарий 5: скрытие сообщения затронуло обсуждение или текст виден")
	}

	// Сценарий 6: снятое событие пропадает из выдачи и закрыто для записи даже по прямой ссылке.
	_, err = s.ModerateEvent(ctx, admin, tournament.ID, "unpublished", "")
	expect(err, platform.ErrCommentRequired)
	_, err = s.ModerateEvent(ctx, admin, tournament.ID, "unpublished", "Некорректное место")
	must(err)
	orgEvents, err = s.ListEvents(ctx, igor, EventFilter{OrganizationID: org.ID, Limit: 10})
	must(err)
	if orgEvents.Total != 0 {
		t.Fatal("сценарий 6: снятое событие осталось в выдаче")
	}
	_, err = s.Join(ctx, person("Новичок"), tournament.ID, "")
	expect(err, ErrEventClosed)
	stranger := person("Посторонний")
	_, err = s.GetEvent(ctx, stranger, tournament.ID)
	expect(err, ErrNotFound)
	banner, err := s.GetEvent(ctx, anna, tournament.ID)
	must(err)
	if banner.Status != "unpublished" || banner.ModerationReason != "Некорректное место" {
		t.Fatal("сценарий 6: участник не видит причину снятия")
	}
	var notices int
	must(db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE type='notification.requested' AND user_id=$1", anna.ID).Scan(&notices))
	if notices < 2 {
		t.Fatalf("уведомления участнику: %d (ожидались о принятии заявки и о снятии)", notices)
	}
}

func ptr[T any](v T) *T { return &v }
