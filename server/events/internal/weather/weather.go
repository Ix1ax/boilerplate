// Package weather — прогноз для уличных встреч через Open-Meteo (бесплатно, без ключа и регистрации).
// Прогноз — подсказка «что надеть и взять», а не условие встречи: при любой ошибке сервиса
// платформа просто не показывает погоду.
package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// DefaultURL — публичный API Open-Meteo. Он не требует ключа; ограничения — в README.
const DefaultURL = "https://api.open-meteo.com/v1/forecast"

// Horizon — дальше Open-Meteo не прогнозирует.
const Horizon = 15 * 24 * time.Hour

// cacheTTL — прогноз на час вперёд меняется медленно; кэш бережёт бесплатный API от карточек,
// которые открывают десятки раз.
const cacheTTL = 30 * time.Minute

var (
	ErrTooFar = errors.New("прогноз появится ближе к дате встречи")
	ErrNoData = errors.New("прогноз на это время недоступен")
)

type Forecast struct {
	Time                     time.Time `json:"time"`
	TemperatureC             float64   `json:"temperature_c"`
	PrecipitationProbability int       `json:"precipitation_probability"`
	Code                     int       `json:"code"`
	Description              string    `json:"description"`
	Advice                   string    `json:"advice"`
}

// Summary — одна строка для напоминания: «+4°, дождь, осадки 70% — возьмите дождевик».
func (f Forecast) Summary() string {
	s := fmt.Sprintf("%+.0f°, %s", f.TemperatureC, f.Description)
	if f.PrecipitationProbability >= 30 {
		s += fmt.Sprintf(", осадки %d%%", f.PrecipitationProbability)
	}
	if f.Advice != "" {
		s += " — " + f.Advice
	}
	return s
}

type Client struct {
	base string
	http *http.Client
	now  func() time.Time

	mu    sync.Mutex
	cache map[string]cached
}

type cached struct {
	body response
	at   time.Time
}

type response struct {
	Hourly struct {
		Time          []string  `json:"time"`
		Temperature   []float64 `json:"temperature_2m"`
		Precipitation []*int    `json:"precipitation_probability"`
		Code          []int     `json:"weather_code"`
	} `json:"hourly"`
}

func New(baseURL string) *Client {
	return &Client{base: baseURL, http: &http.Client{Timeout: 5 * time.Second}, now: time.Now, cache: map[string]cached{}}
}

// At — прогноз на час, ближайший к t, в точке lat/lng.
func (c *Client) At(ctx context.Context, lat, lng float64, t time.Time) (Forecast, error) {
	now := c.now()
	if t.After(now.Add(Horizon)) {
		return Forecast{}, ErrTooFar
	}
	hour := t.UTC().Round(time.Hour)
	day := hour.Format(time.DateOnly)
	// Координаты округляем до ~1 км: соседние события делят один запрос к API.
	key := fmt.Sprintf("%.2f,%.2f,%s", lat, lng, day)
	body, err := c.fetch(ctx, key, lat, lng, day)
	if err != nil {
		return Forecast{}, err
	}
	want := hour.Format("2006-01-02T15:04")
	for i, ts := range body.Hourly.Time {
		if ts != want || i >= len(body.Hourly.Temperature) || i >= len(body.Hourly.Code) {
			continue
		}
		f := Forecast{Time: hour, TemperatureC: body.Hourly.Temperature[i], Code: body.Hourly.Code[i]}
		if i < len(body.Hourly.Precipitation) && body.Hourly.Precipitation[i] != nil {
			f.PrecipitationProbability = *body.Hourly.Precipitation[i]
		}
		f.Description, f.Advice = Describe(f.Code), Advise(f.TemperatureC, f.PrecipitationProbability, f.Code)
		return f, nil
	}
	return Forecast{}, ErrNoData
}

func (c *Client) fetch(ctx context.Context, key string, lat, lng float64, day string) (response, error) {
	c.mu.Lock()
	hit, ok := c.cache[key]
	c.mu.Unlock()
	if ok && c.now().Sub(hit.at) < cacheTTL {
		return hit.body, nil
	}
	q := url.Values{
		"latitude": {fmt.Sprintf("%.2f", lat)}, "longitude": {fmt.Sprintf("%.2f", lng)},
		"hourly":   {"temperature_2m,precipitation_probability,weather_code"},
		"timezone": {"UTC"}, "start_date": {day}, "end_date": {day},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"?"+q.Encode(), nil)
	if err != nil {
		return response{}, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return response{}, fmt.Errorf("Open-Meteo недоступен: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return response{}, fmt.Errorf("Open-Meteo ответил %d", res.StatusCode)
	}
	var body response
	if err = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body); err != nil {
		return response{}, fmt.Errorf("Open-Meteo: непонятный ответ: %w", err)
	}
	c.mu.Lock()
	c.cache[key] = cached{body: body, at: c.now()}
	c.mu.Unlock()
	return body, nil
}

// Describe переводит код погоды WMO, который возвращает Open-Meteo, в слова.
func Describe(code int) string {
	switch {
	case code == 0:
		return "ясно"
	case code <= 2:
		return "переменная облачность"
	case code == 3:
		return "пасмурно"
	case code == 45 || code == 48:
		return "туман"
	case code >= 51 && code <= 57:
		return "морось"
	case code >= 61 && code <= 67:
		return "дождь"
	case code >= 71 && code <= 77:
		return "снег"
	case code >= 80 && code <= 82:
		return "ливень"
	case code == 85 || code == 86:
		return "снегопад"
	case code >= 95:
		return "гроза"
	}
	return "без осадков"
}

// Advise — один главный совет: что взять или надеть. Самое важное — первым.
func Advise(tempC float64, precipitation, code int) string {
	switch {
	case code >= 95:
		return "гроза: следите за сообщениями организатора"
	case tempC <= -20:
		return "сильный мороз: одевайтесь слоями, закройте лицо"
	// До +3° на улице без шапки и перчаток уже холодно, особенно когда стоишь между упражнениями.
	case tempC <= 3:
		return "одевайтесь теплее, возьмите шапку и перчатки"
	case precipitation >= 50 || code >= 51 && code <= 67 || code >= 80 && code <= 82:
		return "возьмите дождевик и сменную обувь"
	case code >= 71 && code <= 77 || code == 85 || code == 86:
		return "наденьте непромокаемую обувь"
	case tempC >= 27:
		return "возьмите воду и головной убор"
	case tempC <= 8:
		return "прохладно: возьмите кофту"
	}
	return ""
}
