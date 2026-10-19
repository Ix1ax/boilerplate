// Package copilot — ИИ-подсказка модератору по жалобе: правила (детерминированно) + совет LLM
// (опционально). Решение всегда остаётся за админом: пакет ничего не скрывает и не удаляет,
// он только возвращает рекомендацию и адреса обычных админских действий.
package copilot

import (
	"regexp"
	"strings"
	"unicode"
)

// Flag — найденный правилом признак нарушения: Code — для интерфейса и фильтров, Message — для человека.
type Flag struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	moneyWords = regexp.MustCompile(`(?i)(переведите|перевод\s+на|на\s+карт|реквизит|донат|пожертвован|кошел[её]к|предоплат|\d+\s*(руб|₽|р\.))`)
	cardNumber = regexp.MustCompile(`\b(?:\d[ -]?){16}\b`)
	linkRe     = regexp.MustCompile(`(?i)(https?://|www\.|t\.me/|wa\.me/)`)
	contactRe  = regexp.MustCompile(`(?i)([\w.+-]+@[\w-]+\.[\w.]+|\+?\d[\d\s()-]{9,})`)
	commerceRe = regexp.MustCompile(`(?i)(прода[мюё]|куплю|скидк|заработ|доход|казино|ставк[аи]\s+на|промокод|пиши(те)?\s+в\s+личк)`)
	// Короткий список: правило ловит очевидное, остальное — совет модели и решение админа.
	insultRe = regexp.MustCompile(`(?i)(идиот|дебил|урод|тупиц|тупой|придур|кретин|мраз|сволоч)`)
)

// manyReports — с этого числа открытых жалоб на одну цель она поднимается в приоритете.
const manyReports = 3

// Check — правила по тексту цели. Флаги ставят только они: ответ модели на них не влияет,
// поэтому текст с «инструкциями для модели» не может скрыть нарушение.
func Check(text string) []Flag {
	flags := []Flag{}
	add := func(code, msg string) { flags = append(flags, Flag{code, msg}) }
	if moneyWords.MatchString(text) || cardNumber.MatchString(text) {
		add("money", "Похоже на сбор или перевод денег")
	}
	if commerceRe.MatchString(text) {
		add("commerce", "Похоже на рекламу или продажу")
	}
	if insultRe.MatchString(text) {
		add("insult", "Есть оскорбительные слова")
	}
	if linkRe.MatchString(text) {
		add("link", "В тексте есть ссылка")
	}
	if contactRe.MatchString(text) && !cardNumber.MatchString(text) {
		add("contact", "В тексте есть телефон или e-mail")
	}
	letters, upper := 0, 0
	for _, r := range text {
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
	return flags
}

// Priority — насколько срочно разбирать жалобу: high при серьёзной причине или многих жалобах.
func Priority(reason string, openOnTarget int) string {
	if reason == "fraud" || reason == "dangerous" || openOnTarget >= manyReports {
		return "high"
	}
	return "normal"
}

// clip обрезает текст для модели: длинный ввод дорог и не добавляет смысла подсказке.
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}
