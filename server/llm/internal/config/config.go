// Package config читает настройки сервиса из переменных окружения (они описаны в .env.example)
// и сразу проверяет обязательные, чтобы сервис падал при старте, а не на первом запросе.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"
)

// Config — настройки llm: адрес Ollama, модель, таймаут и лимиты (переменные LLM_*).
type Config struct {
	Addr, InternalToken, OllamaURL, Model string
	Timeout                               time.Duration
	MaxConcurrent, MaxPromptRunes         int
}

func Load() (Config, error) {
	c := Config{
		Addr: Value("HTTP_ADDR", ":8080"), InternalToken: os.Getenv("INTERNAL_TOKEN"),
		OllamaURL: Value("LLM_URL", "http://ollama:11434"), Model: Value("LLM_MODEL", "qwen2.5:1.5b"),
	}
	if c.InternalToken == "" {
		return c, fmt.Errorf("Не задана переменная INTERNAL_TOKEN; выполните make setup")
	}
	if u, err := url.Parse(c.OllamaURL); err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return c, fmt.Errorf("Некорректное значение LLM_URL")
	}
	var err error
	if c.Timeout, err = time.ParseDuration(Value("LLM_TIMEOUT", "180s")); err != nil || c.Timeout <= 0 {
		return c, fmt.Errorf("Некорректное значение LLM_TIMEOUT")
	}
	// Параллельных генераций мало: модель работает на CPU, лишние запросы получают 429.
	if c.MaxConcurrent, err = strconv.Atoi(Value("LLM_MAX_CONCURRENT", "2")); err != nil || c.MaxConcurrent < 1 {
		return c, fmt.Errorf("Некорректное значение LLM_MAX_CONCURRENT")
	}
	if c.MaxPromptRunes, err = strconv.Atoi(Value("LLM_MAX_PROMPT_RUNES", "8000")); err != nil || c.MaxPromptRunes < 1 {
		return c, fmt.Errorf("Некорректное значение LLM_MAX_PROMPT_RUNES")
	}
	return c, nil
}
func Value(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
