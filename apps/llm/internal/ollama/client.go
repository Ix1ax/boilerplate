// Package ollama — тонкий клиент HTTP API Ollama (/api/generate, /api/tags).
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var ErrUnavailable = errors.New("Модель временно недоступна")

type Request struct {
	System, Prompt string
	JSON           bool
	Temperature    *float64
}
type Client struct {
	base, model string
	http        *http.Client
}

func New(base, model string, timeout time.Duration) *Client {
	return &Client{base: strings.TrimRight(base, "/"), model: model, http: &http.Client{Timeout: timeout}}
}
func (c *Client) Model() string { return c.model }

func (c *Client) body(r Request, stream bool) ([]byte, error) {
	p := map[string]any{"model": c.model, "prompt": r.Prompt, "stream": stream}
	if r.System != "" {
		p["system"] = r.System
	}
	if r.JSON {
		p["format"] = "json"
	}
	if r.Temperature != nil {
		p["options"] = map[string]any{"temperature": *r.Temperature}
	}
	return json.Marshal(p)
}
func (c *Client) post(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("%w: статус %d", ErrUnavailable, resp.StatusCode)
	}
	return resp, nil
}

// Generate возвращает ответ целиком.
func (c *Client) Generate(ctx context.Context, r Request) (string, error) {
	b, err := c.body(r, false)
	if err != nil {
		return "", err
	}
	resp, err := c.post(ctx, b)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Response string `json:"response"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return out.Response, nil
}

// Stream вызывает onChunk для каждого фрагмента ответа (NDJSON от Ollama).
func (c *Client) Stream(ctx context.Context, r Request, onChunk func(string) error) error {
	b, err := c.body(r, true)
	if err != nil {
		return err
	}
	resp, err := c.post(ctx, b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	for sc.Scan() {
		var line struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
			Error    string `json:"error"`
		}
		if json.Unmarshal(sc.Bytes(), &line) != nil {
			continue
		}
		if line.Error != "" {
			return fmt.Errorf("%w: %s", ErrUnavailable, line.Error)
		}
		if line.Response != "" {
			if err = onChunk(line.Response); err != nil {
				return err
			}
		}
		if line.Done {
			return nil
		}
	}
	if err = sc.Err(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	return nil
}

type Status struct {
	Available      bool   `json:"available"`
	Model          string `json:"model"`
	ModelInstalled bool   `json:"model_installed"`
}

// Status не возвращает ошибку: недоступность Ollama — штатное состояние (профиль llm не запущен).
func (c *Client) Status(ctx context.Context) Status {
	st := Status{Model: c.model}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/tags", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return st
	}
	defer resp.Body.Close()
	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tags) != nil {
		return st
	}
	st.Available = true
	for _, m := range tags.Models {
		if m.Name == c.model || strings.TrimSuffix(m.Name, ":latest") == c.model {
			st.ModelInstalled = true
		}
	}
	return st
}
