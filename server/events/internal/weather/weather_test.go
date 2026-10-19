package weather

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientPicksHourAndCaches(t *testing.T) {
	var calls atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("start_date") != "2026-10-24" || r.URL.Query().Get("latitude") != "56.02" {
			t.Errorf("запрос: %s", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"hourly":{"time":["2026-10-24T02:00","2026-10-24T03:00"],
"temperature_2m":[2.1,4.4],"precipitation_probability":[10,70],"weather_code":[3,61]}}`))
	}))
	defer api.Close()
	c := New(api.URL)
	c.now = func() time.Time { return time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC) }
	// 10:10 по Красноярску = 03:10 UTC → ближайший час 03:00.
	at := time.Date(2026, 10, 24, 10, 10, 0, 0, time.FixedZone("KRAT", 7*3600))
	f, err := c.At(context.Background(), 56.0153, 92.8932, at)
	if err != nil {
		t.Fatal(err)
	}
	if f.TemperatureC != 4.4 || f.PrecipitationProbability != 70 || f.Description != "дождь" || f.Advice != "возьмите дождевик и сменную обувь" {
		t.Fatalf("%+v", f)
	}
	if got := f.Summary(); got != "+4°, дождь, осадки 70% — возьмите дождевик и сменную обувь" {
		t.Fatal(got)
	}
	if _, err = c.At(context.Background(), 56.0179, 92.8911, at); err != nil || calls.Load() != 1 {
		t.Fatalf("соседняя точка не взята из кэша: %v, запросов %d", err, calls.Load())
	}
}

func TestClientErrors(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(503) }))
	defer down.Close()
	c := New(down.URL)
	now := time.Now()
	if _, err := c.At(context.Background(), 56, 92, now.Add(20*24*time.Hour)); !errors.Is(err, ErrTooFar) {
		t.Fatalf("дальше горизонта: %v", err)
	}
	if _, err := c.At(context.Background(), 56, 92, now.Add(time.Hour)); err == nil {
		t.Fatal("ошибка API не возвращена")
	}
}

func TestAdvise(t *testing.T) {
	cases := []struct {
		temp         float64
		precip, code int
		want         string
	}{
		{20, 0, 95, "гроза: следите за сообщениями организатора"},
		{-25, 0, 0, "сильный мороз: одевайтесь слоями, закройте лицо"},
		{-3, 80, 71, "одевайтесь теплее, возьмите шапку и перчатки"},
		{12, 60, 3, "возьмите дождевик и сменную обувь"},
		{30, 0, 0, "возьмите воду и головной убор"},
		{2.5, 0, 3, "одевайтесь теплее, возьмите шапку и перчатки"},
		{6, 0, 1, "прохладно: возьмите кофту"},
		{18, 10, 1, ""},
	}
	for _, tc := range cases {
		if got := Advise(tc.temp, tc.precip, tc.code); got != tc.want {
			t.Errorf("%v°, %d%%, код %d: %q", tc.temp, tc.precip, tc.code, got)
		}
	}
}
