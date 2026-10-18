// Package llm — домен «генерация текста»: проверка запроса, лимит параллелизма, вызов модели.
package llm

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"example.com/hackathon/server/llm/internal/ollama"
)

var (
	ErrPrompt = errors.New("Запрос должен содержать от 1 до заданного предела символов")
	ErrBusy   = errors.New("Модель занята, повторите через несколько секунд")
)

// Completer — порт к модели; в тестах подменяется фейком.
type Completer interface {
	Model() string
	Generate(ctx context.Context, r ollama.Request) (string, error)
	Stream(ctx context.Context, r ollama.Request, onChunk func(string) error) error
	Status(ctx context.Context) ollama.Status
}

// Service защищает модель от перегрузки: проверяет длину запроса и параметры и не запускает
// больше maxConcurrent генераций сразу — лишние сразу получают ErrBusy, а не ждут в очереди.
type Service struct {
	model     Completer
	slots     chan struct{}
	maxPrompt int
}

func New(model Completer, maxConcurrent, maxPromptRunes int) *Service {
	return &Service{model: model, slots: make(chan struct{}, maxConcurrent), maxPrompt: maxPromptRunes}
}

func (s *Service) validate(r ollama.Request) (ollama.Request, error) {
	r.Prompt, r.System = strings.TrimSpace(r.Prompt), strings.TrimSpace(r.System)
	if r.Prompt == "" || !utf8.ValidString(r.Prompt+r.System) || utf8.RuneCountInString(r.Prompt)+utf8.RuneCountInString(r.System) > s.maxPrompt {
		return r, ErrPrompt
	}
	if r.Temperature != nil && (*r.Temperature < 0 || *r.Temperature > 2) {
		return r, ErrPrompt
	}
	return r, nil
}
func (s *Service) acquire() (release func(), err error) {
	select {
	case s.slots <- struct{}{}:
		return func() { <-s.slots }, nil
	default:
		return nil, ErrBusy
	}
}

func (s *Service) Generate(ctx context.Context, r ollama.Request) (string, error) {
	r, err := s.validate(r)
	if err != nil {
		return "", err
	}
	release, err := s.acquire()
	if err != nil {
		return "", err
	}
	defer release()
	return s.model.Generate(ctx, r)
}
func (s *Service) Stream(ctx context.Context, r ollama.Request, onChunk func(string) error) error {
	r, err := s.validate(r)
	if err != nil {
		return err
	}
	release, err := s.acquire()
	if err != nil {
		return err
	}
	defer release()
	return s.model.Stream(ctx, r, onChunk)
}
func (s *Service) Model() string                            { return s.model.Model() }
func (s *Service) Status(ctx context.Context) ollama.Status { return s.model.Status(ctx) }
