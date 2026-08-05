package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/myroutine/backend/internal/api"
	"github.com/myroutine/backend/internal/config"
	"github.com/myroutine/backend/internal/testutil"
)

// These tests drive the real router over httptest: routing, JWT middleware,
// response envelopes and status-code mapping — the contract the web and
// mobile clients depend on.

var (
	testPool   *pgxpool.Pool
	testServer *httptest.Server
)

var testCfg = &config.Config{
	AppEnv:               "test",
	JWTSecret:            "test-secret-with-at-least-32-characters!",
	JWTExpiryHours:       "24",
	JWTRefreshExpiryDays: "30",
	CORSAllowedOrigins:   "http://localhost:5173",
}

func TestMain(m *testing.M) {
	pool, cleanup, err := testutil.StartPostgres(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "test setup failed: %v\n", err)
		os.Exit(1)
	}
	testPool = pool

	if pool != nil {
		testServer = httptest.NewServer(api.NewRouter(testCfg, pool, zap.NewNop()))
	}

	code := m.Run()

	if testServer != nil {
		testServer.Close()
	}
	cleanup()
	os.Exit(code)
}

func requireServer(t *testing.T) string {
	t.Helper()
	if testServer == nil {
		t.Skip("integration test requires Docker")
	}
	return testServer.URL
}

// ─── HTTP helpers ─────────────────────────────────────────────────────────────

type apiClient struct {
	t     *testing.T
	base  string
	token string
}

func (c *apiClient) do(method, path string, body any) (*http.Response, []byte) {
	c.t.Helper()

	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshaling body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("building request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(res.Body); err != nil {
		c.t.Fatalf("reading body: %v", err)
	}
	return res, buf.Bytes()
}

func (c *apiClient) decode(raw []byte, target any) {
	c.t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		c.t.Fatalf("decoding %s: %v", raw, err)
	}
}

// newAuthedClient registers a fresh user and returns a client holding its token.
func newAuthedClient(t *testing.T) *apiClient {
	t.Helper()
	base := requireServer(t)
	c := &apiClient{t: t, base: base}

	email := fmt.Sprintf("router-%d@test.local", time.Now().UnixNano())
	res, raw := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Router Test", "email": email, "password": "testpassword123",
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", res.StatusCode, raw)
	}

	var auth struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	c.decode(raw, &auth)
	c.token = auth.AccessToken

	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", auth.User.ID)
	})
	return c
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHealthCheckIsPublic(t *testing.T) {
	base := requireServer(t)
	res, err := http.Get(base + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}

func TestProtectedRoutesRequireJWT(t *testing.T) {
	base := requireServer(t)
	c := &apiClient{t: t, base: base}

	protected := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/habits"},
		{http.MethodGet, "/api/v1/habits/stats"},
		{http.MethodGet, "/api/v1/me"},
		{http.MethodGet, "/api/v1/tasks?date=2026-08-05"},
		{http.MethodGet, "/api/v1/finance/summary"},
		{http.MethodGet, "/api/v1/health-module/activities"},
		{http.MethodGet, "/api/v1/study/sessions"},
	}

	for _, route := range protected {
		res, _ := c.do(route.method, route.path, nil)
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401 without a token",
				route.method, route.path, res.StatusCode)
		}
	}
}

func TestRejectsGarbageToken(t *testing.T) {
	base := requireServer(t)
	c := &apiClient{t: t, base: base, token: "not-a-real-jwt"}

	res, _ := c.do(http.MethodGet, "/api/v1/habits", nil)
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 for a malformed token", res.StatusCode)
	}
}

// The web client reads data out of a named envelope on these routes; a change
// here breaks the UI silently, so it is pinned by a test.
func TestListEnvelopes(t *testing.T) {
	c := newAuthedClient(t)

	cases := []struct{ path, key string }{
		{"/api/v1/habits", "habits"},
		{"/api/v1/habits/heatmap", "heatmap"},
		{"/api/v1/reviews/missed", "missed_days"},
		{"/api/v1/tasks?date=2026-08-05", "tasks"},
		{"/api/v1/goals?month=2026-08-01", "goals"},
		{"/api/v1/finance/transactions", "transactions"},
		{"/api/v1/finance/budgets", "budgets"},
		{"/api/v1/health-module/activities", "activities"},
		{"/api/v1/health-module/body-metrics", "body_metrics"},
		{"/api/v1/study/sessions", "sessions"},
	}

	for _, tc := range cases {
		res, raw := c.do(http.MethodGet, tc.path, nil)
		if res.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, body = %s", tc.path, res.StatusCode, raw)
			continue
		}

		var envelope map[string]json.RawMessage
		c.decode(raw, &envelope)

		value, ok := envelope[tc.key]
		if !ok {
			t.Errorf("GET %s: missing %q key, got %s", tc.path, tc.key, raw)
			continue
		}
		// Empty collections must serialize as [] so the UI can map over them.
		if string(value) != "[]" {
			t.Errorf("GET %s: %s = %s, want [] for an empty collection", tc.path, tc.key, value)
		}
	}
}

func TestUnknownIDsReturn404(t *testing.T) {
	c := newAuthedClient(t)
	missing := "00000000-0000-0000-0000-000000000000"

	cases := []struct {
		name, method, path string
		body               any
	}{
		{"get habit", http.MethodGet, "/api/v1/habits/" + missing, nil},
		{"update habit", http.MethodPut, "/api/v1/habits/" + missing, map[string]string{"name": "x"}},
		{"delete habit", http.MethodDelete, "/api/v1/habits/" + missing, nil},
		{"check in", http.MethodPost, "/api/v1/habits/" + missing + "/checkin", map[string]string{}},
		{"review day", http.MethodPost, "/api/v1/habits/" + missing + "/review", map[string]string{"status": "discarded"}},
		{"patch task", http.MethodPatch, "/api/v1/tasks/" + missing, map[string]string{"title": "x"}},
		{"delete task", http.MethodDelete, "/api/v1/tasks/" + missing, nil},
		{"advance task", http.MethodPost, "/api/v1/tasks/" + missing + "/advance", nil},
		{"delete goal", http.MethodDelete, "/api/v1/goals/" + missing, nil},
		{"delete transaction", http.MethodDelete, "/api/v1/finance/transactions/" + missing, nil},
		{"delete study session", http.MethodDelete, "/api/v1/study/sessions/" + missing, nil},
	}

	for _, tc := range cases {
		res, raw := c.do(tc.method, tc.path, tc.body)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, want 404 (body: %s)", tc.name, res.StatusCode, raw)
		}
	}
}

func TestValidationErrorsReturn400(t *testing.T) {
	c := newAuthedClient(t)

	cases := []struct {
		name, method, path string
		body               any
	}{
		{"habit without name", http.MethodPost, "/api/v1/habits", map[string]string{}},
		{"task without title", http.MethodPost, "/api/v1/tasks", map[string]string{"date": "2026-08-05"}},
		{"tasks without date param", http.MethodGet, "/api/v1/tasks", nil},
		{"goal without month", http.MethodPost, "/api/v1/goals", map[string]string{"title": "x"}},
		{"negative amount", http.MethodPost, "/api/v1/finance/transactions",
			map[string]any{"amount_cents": -1, "description": "x", "occurred_on": "2026-08-05"}},
		{"study without subject", http.MethodPost, "/api/v1/study/sessions",
			map[string]any{"duration_minutes": 30}},
	}

	for _, tc := range cases {
		res, raw := c.do(tc.method, tc.path, tc.body)
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400 (body: %s)", tc.name, res.StatusCode, raw)
		}

		// Errors always use the {"error": "..."} envelope.
		var errBody struct {
			Error string `json:"error"`
		}
		c.decode(raw, &errBody)
		if errBody.Error == "" {
			t.Errorf("%s: missing error message in %s", tc.name, raw)
		}
	}
}

// Full happy path over HTTP: create a habit, check in, and see it reflected
// in the list and stats.
func TestHabitFlowOverHTTP(t *testing.T) {
	c := newAuthedClient(t)

	res, raw := c.do(http.MethodPost, "/api/v1/habits", map[string]any{
		"name": "Correr", "icon": "🏃", "time_of_day": "morning", "category": "health",
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create habit status = %d, body = %s", res.StatusCode, raw)
	}
	var habit struct {
		ID string `json:"id"`
	}
	c.decode(raw, &habit)

	res, raw = c.do(http.MethodPost, "/api/v1/habits/"+habit.ID+"/checkin", map[string]any{})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("check-in status = %d, body = %s", res.StatusCode, raw)
	}

	res, raw = c.do(http.MethodGet, "/api/v1/habits", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", res.StatusCode)
	}
	var list struct {
		Habits []struct {
			ID             string `json:"id"`
			CompletedToday bool   `json:"completed_today"`
			CurrentStreak  int    `json:"current_streak"`
		} `json:"habits"`
	}
	c.decode(raw, &list)
	if len(list.Habits) != 1 {
		t.Fatalf("habit count = %d, want 1", len(list.Habits))
	}
	if !list.Habits[0].CompletedToday || list.Habits[0].CurrentStreak != 1 {
		t.Errorf("habit = completed:%v streak:%d, want true/1",
			list.Habits[0].CompletedToday, list.Habits[0].CurrentStreak)
	}

	res, raw = c.do(http.MethodGet, "/api/v1/habits/stats", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("stats status = %d", res.StatusCode)
	}
	var stats struct {
		TotalHabits   int `json:"total_habits"`
		CurrentStreak int `json:"current_streak"`
		HabitStats    []struct {
			HabitID string `json:"habit_id"`
		} `json:"habit_stats"`
	}
	c.decode(raw, &stats)
	if stats.TotalHabits != 1 || stats.CurrentStreak != 1 || len(stats.HabitStats) != 1 {
		t.Errorf("stats = %+v, want 1 habit / streak 1 / 1 habit_stat", stats)
	}
}

// Users must not see or touch each other's data through the API.
func TestCrossUserAccessIsBlocked(t *testing.T) {
	owner := newAuthedClient(t)
	other := newAuthedClient(t)

	res, raw := owner.do(http.MethodPost, "/api/v1/habits", map[string]any{"name": "Privado"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create habit status = %d, body = %s", res.StatusCode, raw)
	}
	var habit struct {
		ID string `json:"id"`
	}
	owner.decode(raw, &habit)

	for _, tc := range []struct {
		name, method, path string
		body               any
	}{
		{"read", http.MethodGet, "/api/v1/habits/" + habit.ID, nil},
		{"update", http.MethodPut, "/api/v1/habits/" + habit.ID, map[string]string{"name": "hack"}},
		{"delete", http.MethodDelete, "/api/v1/habits/" + habit.ID, nil},
		{"review", http.MethodPost, "/api/v1/habits/" + habit.ID + "/review", map[string]string{"status": "discarded"}},
	} {
		res, _ := other.do(tc.method, tc.path, tc.body)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s another user's habit: status = %d, want 404", tc.name, res.StatusCode)
		}
	}
}
