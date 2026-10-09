package api_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
	AdminEmails:          []string{testAdminEmail},
}

// testAdminEmail may register without an invite and becomes admin.
const testAdminEmail = "router-admin@test.local"

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

// seedInvite stores a usable invite for email straight in the database and
// returns its raw token (only the SHA-256 hash is stored, like the app does).
func seedInvite(t *testing.T, email string) string {
	t.Helper()
	token := fmt.Sprintf("router-invite-%d", time.Now().UnixNano())
	sum := sha256.Sum256([]byte(token))
	_, err := testPool.Exec(context.Background(),
		`INSERT INTO invites (token_hash, email, expires_at) VALUES ($1, $2, NOW() + INTERVAL '1 day')`,
		hex.EncodeToString(sum[:]), email,
	)
	if err != nil {
		t.Fatalf("seeding invite: %v", err)
	}
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(),
			"DELETE FROM invites WHERE token_hash = $1", hex.EncodeToString(sum[:]))
	})
	return token
}

// newAuthedClient registers a fresh (invited) user and returns a client
// holding its token.
func newAuthedClient(t *testing.T) *apiClient {
	t.Helper()
	base := requireServer(t)
	c := &apiClient{t: t, base: base}

	email := fmt.Sprintf("router-%d@test.local", time.Now().UnixNano())
	res, raw := c.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Router Test", "email": email, "password": "testpassword123",
		"invite_token": seedInvite(t, email),
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

// Invite-only registration over HTTP: the admin creates an invite, the
// invitee looks it up and registers with it; nobody else gets in.
func TestInviteFlowOverHTTP(t *testing.T) {
	base := requireServer(t)

	// The admin (in ADMIN_EMAILS) registers without an invite.
	admin := &apiClient{t: t, base: base}
	res, raw := admin.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Admin", "email": testAdminEmail, "password": "testpassword123",
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("admin register status = %d, body = %s", res.StatusCode, raw)
	}
	var adminAuth struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID      string `json:"id"`
			IsAdmin bool   `json:"is_admin"`
		} `json:"user"`
	}
	admin.decode(raw, &adminAuth)
	admin.token = adminAuth.AccessToken
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM invites WHERE invited_by = $1", adminAuth.User.ID)
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", adminAuth.User.ID)
	})
	if !adminAuth.User.IsAdmin {
		t.Fatal("admin register: is_admin = false, want true")
	}

	// Without an invite, a stranger cannot register.
	stranger := &apiClient{t: t, base: base}
	res, raw = stranger.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Estranho", "email": fmt.Sprintf("stranger-%d@test.local", time.Now().UnixNano()), "password": "testpassword123",
	})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("register without invite status = %d, want 403 (body %s)", res.StatusCode, raw)
	}

	// The admin invites someone.
	guestEmail := fmt.Sprintf("guest-%d@test.local", time.Now().UnixNano())
	res, raw = admin.do(http.MethodPost, "/api/v1/admin/invites", map[string]string{"email": guestEmail})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create invite status = %d, body = %s", res.StatusCode, raw)
	}
	var inv struct {
		Token string `json:"token"`
	}
	admin.decode(raw, &inv)

	// The registration page learns which email the link is for.
	guest := &apiClient{t: t, base: base}
	res, raw = guest.do(http.MethodGet, "/api/v1/auth/invites/"+inv.Token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("lookup status = %d, body = %s", res.StatusCode, raw)
	}

	res, raw = guest.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Convidado", "email": guestEmail, "password": "testpassword123", "invite_token": inv.Token,
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("invited register status = %d, body = %s", res.StatusCode, raw)
	}
	var guestAuth struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID      string `json:"id"`
			IsAdmin bool   `json:"is_admin"`
		} `json:"user"`
	}
	guest.decode(raw, &guestAuth)
	guest.token = guestAuth.AccessToken
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", guestAuth.User.ID)
	})
	if guestAuth.User.IsAdmin {
		t.Fatal("invited user must not be admin")
	}

	// A used link no longer resolves.
	res, _ = guest.do(http.MethodGet, "/api/v1/auth/invites/"+inv.Token, nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("lookup of used invite status = %d, want 404", res.StatusCode)
	}

	// A regular user cannot reach the admin area.
	res, _ = guest.do(http.MethodGet, "/api/v1/admin/invites", nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("non-admin GET /admin/invites status = %d, want 403", res.StatusCode)
	}

	// The admin sees the invite as used.
	res, raw = admin.do(http.MethodGet, "/api/v1/admin/invites", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("list invites status = %d, body = %s", res.StatusCode, raw)
	}
	var list struct {
		Invites []struct {
			Email  string `json:"email"`
			Status string `json:"status"`
		} `json:"invites"`
	}
	admin.decode(raw, &list)
	found := false
	for _, i := range list.Invites {
		if i.Email == guestEmail {
			found = true
			if i.Status != "used" {
				t.Fatalf("invite status = %q, want used", i.Status)
			}
		}
	}
	if !found {
		t.Fatal("created invite not listed")
	}
}

// newAdminClient registers the ADMIN_EMAILS account (no invite needed) and
// returns a client holding its token.
func newAdminClient(t *testing.T) *apiClient {
	t.Helper()
	admin := &apiClient{t: t, base: requireServer(t)}
	res, raw := admin.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Admin", "email": testAdminEmail, "password": "testpassword123",
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("admin register status = %d, body = %s", res.StatusCode, raw)
	}
	var auth struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	admin.decode(raw, &auth)
	admin.token = auth.AccessToken
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", auth.User.ID)
	})
	return admin
}

// Admin-issued password reset over HTTP: the admin generates the link, the
// user sets a new password with it, and a regular user cannot generate links.
func TestPasswordResetOverHTTP(t *testing.T) {
	admin := newAdminClient(t)
	base := requireServer(t)

	email := fmt.Sprintf("forgot-%d@test.local", time.Now().UnixNano())
	user := &apiClient{t: t, base: base}
	res, raw := user.do(http.MethodPost, "/api/v1/auth/register", map[string]string{
		"name": "Esquecido", "email": email, "password": "senha-antiga-1", "invite_token": seedInvite(t, email),
	})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", res.StatusCode, raw)
	}
	var reg struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	user.decode(raw, &reg)
	user.token = reg.AccessToken
	t.Cleanup(func() {
		_, _ = testPool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", reg.User.ID)
	})

	// Only admins can generate reset links.
	res, _ = user.do(http.MethodPost, "/api/v1/admin/password-resets", map[string]string{"email": email})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("non-admin create reset status = %d, want 403", res.StatusCode)
	}

	res, raw = admin.do(http.MethodPost, "/api/v1/admin/password-resets", map[string]string{"email": email})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create reset status = %d, body = %s", res.StatusCode, raw)
	}
	var reset struct {
		Token string `json:"token"`
	}
	admin.decode(raw, &reset)

	anon := &apiClient{t: t, base: base}
	res, raw = anon.do(http.MethodGet, "/api/v1/auth/password-resets/"+reset.Token, nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("lookup status = %d, body = %s", res.StatusCode, raw)
	}

	res, _ = anon.do(http.MethodPost, "/api/v1/auth/password-resets/"+reset.Token, map[string]string{"password": "curta"})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("short password status = %d, want 400", res.StatusCode)
	}
	res, raw = anon.do(http.MethodPost, "/api/v1/auth/password-resets/"+reset.Token, map[string]string{"password": "senha-nova-22"})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("reset status = %d, body = %s", res.StatusCode, raw)
	}

	res, _ = anon.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": email, "password": "senha-nova-22"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login with new password status = %d, want 200", res.StatusCode)
	}
	res, _ = anon.do(http.MethodPost, "/api/v1/auth/password-resets/"+reset.Token, map[string]string{"password": "outra-senha-33"})
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("reused link status = %d, want 404", res.StatusCode)
	}
}

// LGPD over HTTP: download everything, then erase the account; a wrong
// password is refused with 403 (never 401, which the web app reads as an
// expired session).
func TestAccountExportAndDeleteOverHTTP(t *testing.T) {
	c := newAuthedClient(t)

	res, raw := c.do(http.MethodPost, "/api/v1/habits", map[string]any{"name": "Ler"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create habit status = %d, body = %s", res.StatusCode, raw)
	}

	res, raw = c.do(http.MethodGet, "/api/v1/me/export", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("export status = %d, body = %s", res.StatusCode, raw)
	}
	if cd := res.Header.Get("Content-Disposition"); !strings.Contains(cd, "attachment") {
		t.Fatalf("export Content-Disposition = %q, want an attachment", cd)
	}
	var export struct {
		Habits []struct {
			Name string `json:"name"`
		} `json:"habits"`
	}
	c.decode(raw, &export)
	if len(export.Habits) != 1 || export.Habits[0].Name != "Ler" {
		t.Fatalf("exported habits = %+v, want one 'Ler'", export.Habits)
	}

	res, _ = c.do(http.MethodPut, "/api/v1/me/password", map[string]string{
		"current_password": "errada-123", "new_password": "nova-senha-123",
	})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("change password with wrong current = %d, want 403", res.StatusCode)
	}

	res, _ = c.do(http.MethodDelete, "/api/v1/me", map[string]string{"password": "errada-123"})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("delete with wrong password = %d, want 403", res.StatusCode)
	}

	res, raw = c.do(http.MethodDelete, "/api/v1/me", map[string]string{"password": "testpassword123"})
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", res.StatusCode, raw)
	}

	// The still-unexpired access token no longer opens anything.
	res, _ = c.do(http.MethodGet, "/api/v1/habits", nil)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("access after deletion = %d, want 403", res.StatusCode)
	}
}

// Track Day over HTTP: write the day, register what the web parser read,
// undo it; bad dates and oversized text are refused.
func TestJournalOverHTTP(t *testing.T) {
	c := newAuthedClient(t)
	const day = "/api/v1/journal/2026-03-10"
	text := "$ 50 Almoço #alimentacao\ncorrida 5km 30min"

	res, raw := c.do(http.MethodPut, day, map[string]string{"content": text})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("save status = %d, body = %s", res.StatusCode, raw)
	}

	res, raw = c.do(http.MethodPost, day+"/items", map[string]any{
		"line": "$ 50 Almoço #alimentacao", "kind": "transaction",
		"expense": map[string]any{"amount_cents": 5000, "category": "alimentacao", "description": "Almoço"},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("register expense status = %d, body = %s", res.StatusCode, raw)
	}

	res, raw = c.do(http.MethodPost, "/api/v1/habits", map[string]any{"name": "Correr", "category": "health"})
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create habit status = %d, body = %s", res.StatusCode, raw)
	}
	var habit struct {
		ID string `json:"id"`
	}
	c.decode(raw, &habit)

	res, raw = c.do(http.MethodPost, day+"/items", map[string]any{
		"line": "corrida 5km 30min", "kind": "workout",
		"workout": map[string]any{"habit_id": habit.ID, "metrics": map[string]any{"km": 5}, "time_minutes": 30},
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("register workout status = %d, body = %s", res.StatusCode, raw)
	}
	var view struct {
		LineIDs map[string]string `json:"line_ids"`
		Items   []struct {
			SourceID string `json:"source_id"`
			Kind     string `json:"kind"`
		} `json:"items"`
	}
	c.decode(raw, &view)
	if len(view.Items) != 2 || len(view.LineIDs) != 2 {
		t.Fatalf("view = %+v, want 2 items and 2 line ids", view)
	}

	res, raw = c.do(http.MethodDelete, day+"/items/"+view.LineIDs["$ 50 Almoço #alimentacao"], nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("undo status = %d, body = %s", res.StatusCode, raw)
	}
	c.decode(raw, &view)
	if len(view.Items) != 1 || view.Items[0].Kind != "workout" {
		t.Fatalf("after undo items = %+v, want only the workout", view.Items)
	}

	if res, _ = c.do(http.MethodGet, "/api/v1/journal/10-03-2026", nil); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad date status = %d, want 400", res.StatusCode)
	}
	huge := strings.Repeat("a", 20001)
	if res, _ = c.do(http.MethodPut, day, map[string]string{"content": huge}); res.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized text status = %d, want 413", res.StatusCode)
	}
}

// Auth routes are rate-limited per IP (no nginx in front in production).
func TestAuthRoutesAreRateLimited(t *testing.T) {
	requireServer(t)
	cfg := *testCfg
	cfg.AuthRateLimitPerMinute = 3
	srv := httptest.NewServer(api.NewRouter(&cfg, testPool, zap.NewNop()))
	defer srv.Close()

	c := &apiClient{t: t, base: srv.URL}
	login := func() int {
		res, _ := c.do(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "x@test.local", "password": "errada-123"})
		return res.StatusCode
	}
	// The window is the database's clock minute; on a slow CI run the four
	// attempts can straddle a minute and the count restarts. Then try again:
	// two boundaries within one run cannot happen.
	minute := func() (m time.Time) {
		if err := testPool.QueryRow(context.Background(), "SELECT date_trunc('minute', now())").Scan(&m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	for try := 1; ; try++ {
		// Start from zero: earlier requests from this IP count too.
		if _, err := testPool.Exec(context.Background(), "DELETE FROM auth_rate_limits"); err != nil {
			t.Fatal(err)
		}
		start := minute()
		statuses := []int{login(), login(), login(), login()}
		crossed := !minute().Equal(start)
		if crossed && try == 1 {
			continue
		}
		for i, got := range statuses[:3] {
			if got != http.StatusUnauthorized {
				t.Fatalf("attempt %d = %d, want 401 (wrong password, within the limit)", i+1, got)
			}
		}
		if statuses[3] != http.StatusTooManyRequests {
			t.Fatalf("4th attempt = %d, want 429", statuses[3])
		}
		break
	}
	// Protected (non-auth) routes are not affected by the auth limiter.
	if res, _ := c.do(http.MethodGet, "/health", nil); res.StatusCode == http.StatusTooManyRequests {
		t.Fatal("/health must not be rate limited")
	}
}

// Notifications over HTTP: off without VAPID keys; with them, a device can
// subscribe and the settings round-trip.
func TestNotificationsOverHTTP(t *testing.T) {
	c := newAuthedClient(t)

	res, raw := c.do(http.MethodGet, "/api/v1/notifications/config", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(raw), `"enabled":false`) {
		t.Fatalf("config without keys = %d %s, want enabled:false", res.StatusCode, raw)
	}
	sub := map[string]any{
		"endpoint": fmt.Sprintf("https://web.push.apple.com/test/%d", time.Now().UnixNano()),
		"keys":     map[string]string{"p256dh": "BNcRdreALRFXTkOOUHK1EtK2wtaz5Ry4YfYCA_0QTpQtUbVlUls0VJXg7A8u-Ts1XbjhazAkj7I99e8QcYP7DkM", "auth": "tBHItJI5svbpez7KI4CCXg"}, // gitleaks:allow — public example push key (test fixture)
	}
	if res, _ = c.do(http.MethodPost, "/api/v1/notifications/subscriptions", sub); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("subscribe without keys = %d, want 503", res.StatusCode)
	}

	cfg := *testCfg
	cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject = "BPublicKeyForTests", "private", "mailto:test@test.local"
	srv := httptest.NewServer(api.NewRouter(&cfg, testPool, zap.NewNop()))
	defer srv.Close()
	on := &apiClient{t: t, base: srv.URL, token: c.token}

	res, raw = on.do(http.MethodGet, "/api/v1/notifications/config", nil)
	if res.StatusCode != http.StatusOK || !strings.Contains(string(raw), `"vapid_public_key":"BPublicKeyForTests"`) {
		t.Fatalf("config with keys = %d %s", res.StatusCode, raw)
	}
	if res, raw = on.do(http.MethodPost, "/api/v1/notifications/subscriptions", sub); res.StatusCode != http.StatusNoContent {
		t.Fatalf("subscribe = %d %s, want 204", res.StatusCode, raw)
	}
	res, raw = on.do(http.MethodPut, "/api/v1/notifications/settings", map[string]any{
		"task_reminders": true, "task_lead_minutes": 30, "morning_digest": false,
		"morning_time": "07:00", "evening_digest": true, "evening_time": "21:30",
	})
	if res.StatusCode != http.StatusOK || !strings.Contains(string(raw), `"devices":1`) || !strings.Contains(string(raw), `"task_lead_minutes":30`) {
		t.Fatalf("update settings = %d %s", res.StatusCode, raw)
	}
	if res, _ = on.do(http.MethodPost, "/api/v1/notifications/subscriptions", map[string]any{
		"endpoint": "https://evil.example.com/x", "keys": map[string]string{"p256dh": "abc", "auth": "def"},
	}); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("evil endpoint = %d, want 400", res.StatusCode)
	}
}

func TestNotificationDispatchNeedsTheCronSecret(t *testing.T) {
	const path = "/api/v1/internal/notifications/dispatch"
	post := func(srv *httptest.Server, auth string) int {
		req, _ := http.NewRequest(http.MethodPost, srv.URL+path, nil)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}

	off := httptest.NewServer(api.NewRouter(testCfg, testPool, zap.NewNop()))
	defer off.Close()
	if got := post(off, "Bearer anything"); got != http.StatusNotFound {
		t.Fatalf("without CRON_SECRET = %d, want 404 (endpoint off)", got)
	}

	cfg := *testCfg
	cfg.CronSecret = strings.Repeat("s", 40)
	srv := httptest.NewServer(api.NewRouter(&cfg, testPool, zap.NewNop()))
	defer srv.Close()
	for _, bad := range []string{"", "Bearer wrong", cfg.CronSecret} {
		if got := post(srv, bad); got != http.StatusUnauthorized {
			t.Fatalf("auth %q = %d, want 401", bad, got)
		}
	}
	if got := post(srv, "Bearer "+cfg.CronSecret); got != http.StatusServiceUnavailable {
		t.Fatalf("right secret but no VAPID keys = %d, want 503", got)
	}
}

func TestWeeklyGoalsOverHTTP(t *testing.T) {
	c := newAuthedClient(t)

	res, raw := c.do(http.MethodPost, "/api/v1/weekly-goals", map[string]string{"title": "Lista 3 de C2", "date": "2030-03-14"})
	if res.StatusCode != http.StatusCreated || !strings.Contains(string(raw), `"week":"2030-03-11"`) {
		t.Fatalf("create = %d %s, want 201 in the week of 2030-03-11", res.StatusCode, raw)
	}
	var goal struct{ ID string }
	if err := json.Unmarshal(raw, &goal); err != nil {
		t.Fatal(err)
	}

	if res, raw = c.do(http.MethodPatch, "/api/v1/weekly-goals/"+goal.ID, map[string]bool{"done": true}); res.StatusCode != http.StatusOK || !strings.Contains(string(raw), `"done":true`) {
		t.Fatalf("mark done = %d %s", res.StatusCode, raw)
	}
	if res, raw = c.do(http.MethodGet, "/api/v1/weekly-goals?date=2030-03-17", nil); res.StatusCode != http.StatusOK || !strings.Contains(string(raw), "Lista 3 de C2") {
		t.Fatalf("list = %d %s", res.StatusCode, raw)
	}
	if res, _ = c.do(http.MethodGet, "/api/v1/weekly-goals", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("list this week = %d", res.StatusCode)
	}
	if res, _ = c.do(http.MethodPost, "/api/v1/weekly-goals", map[string]string{"title": "  "}); res.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty title = %d, want 400", res.StatusCode)
	}
	if res, _ = c.do(http.MethodPatch, "/api/v1/weekly-goals/not-a-uuid", map[string]bool{"done": true}); res.StatusCode != http.StatusNotFound {
		t.Fatalf("bad id = %d, want 404", res.StatusCode)
	}
	if res, _ = c.do(http.MethodDelete, "/api/v1/weekly-goals/"+goal.ID, nil); res.StatusCode != http.StatusNoContent {
		t.Fatalf("delete = %d, want 204", res.StatusCode)
	}
}
