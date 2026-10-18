// Package copilot — ИИ-помощник проверки: правила (детерминированно) + совет LLM (опционально).
// Решение всегда остаётся за человеком: пакет только возвращает рекомендацию.
package copilot

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Flag — найденное правилом нарушение: Code — для интерфейса и фильтров, Message — для человека.
type Flag struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	moneyWords = regexp.MustCompile(`(?i)(переведите|перевод\s+на|на\s+карт|сбер|тинькофф|реквизит|донат|пожертвован|кошел[её]к|\d+\s*(руб|₽|р\.))`)
	cardNumber = regexp.MustCompile(`\b(?:\d[ -]?){16}\b`)
	linkRe     = regexp.MustCompile(`(?i)(https?://|www\.|t\.me/)`)
	contactRe  = regexp.MustCompile(`(?i)([\w.+-]+@[\w-]+\.[\w.]+|\+?\d[\d\s()-]{9,})`)
)

// Check — правила модерации. Флаги ставят только они: ответ LLM на них не влияет,
// поэтому текст задачи с «инструкциями для модели» не может скрыть нарушение.
func Check(title string) []Flag {
	var flags []Flag
	add := func(code, msg string) { flags = append(flags, Flag{code, msg}) }
	if moneyWords.MatchString(title) || cardNumber.MatchString(title) {
		add("money", "Похоже на сбор или перевод денег")
	}
	if linkRe.MatchString(title) {
		add("link", "В тексте есть ссылка")
	}
	if contactRe.MatchString(title) && !cardNumber.MatchString(title) {
		add("contact", "В тексте есть телефон или e-mail")
	}
	letters, upper := 0, 0
	for _, r := range title {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsUpper(r) {
				upper++
			}
		}
	}
	if letters >= 10 && upper*100/letters > 60 {
		add("shouting", "Текст написан заглавными буквами")
	}
	if utf8.RuneCountInString(strings.TrimSpace(title)) < 5 {
		add("too_short", "Слишком короткое название")
	}
	return flags
}
