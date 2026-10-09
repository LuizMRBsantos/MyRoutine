package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakePush records what would be sent, per endpoint. Endpoints in gone
// answer like a push service whose subscription expired (404/410).
type fakePush struct {
	mu   sync.Mutex
	sent map[string][]PushMessage
	gone map[string]bool
}

func newFakePush() *fakePush {
	return &fakePush{sent: map[string][]PushMessage{}, gone: map[string]bool{}}
}

func (f *fakePush) Send(_ context.Context, target PushTarget, payload []byte) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.gone[target.Endpoint] {
		return true, nil
	}
	var msg PushMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return false, err
	}
	f.sent[target.Endpoint] = append(f.sent[target.Endpoint], msg)
	return false, nil
}

func (f *fakePush) to(endpoint string) []PushMessage {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sent[endpoint]
}

var saoPaulo = func() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		panic(err)
	}
	return loc
}()

// subscribedUser creates a user with one device (product defaults) and
// returns the user id and the device endpoint.
func subscribedUser(t *testing.T) (string, string) {
	t.Helper()
	userID := createTestUser(t)
	endpoint := uniqueEndpoint("web.push.apple.com")
	if err := NewNotificationService(requireDB(t)).Subscribe(context.Background(), userID, pushSub(endpoint), ""); err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	return userID, endpoint
}

func addTask(t *testing.T, userID, title, date, start, status string) {
	t.Helper()
	if _, err := requireDB(t).Exec(context.Background(),
		`INSERT INTO tasks (user_id, title, date, start_time, status) VALUES ($1, $2, $3, NULLIF($4, '')::time, $5)`,
		userID, title, date, start, status,
	); err != nil {
		t.Fatalf("adding task: %v", err)
	}
}

func addHabit(t *testing.T, userID, name string) string {
	t.Helper()
	var id string
	if err := requireDB(t).QueryRow(context.Background(),
		`INSERT INTO habits (user_id, name) VALUES ($1, $2) RETURNING id::text`, userID, name,
	).Scan(&id); err != nil {
		t.Fatalf("adding habit: %v", err)
	}
	return id
}

// at is a moment in São Paulo local time.
func at(date, clock string) time.Time {
	ts, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, saoPaulo)
	if err != nil {
		panic(err)
	}
	return ts
}

func dispatch(t *testing.T, push PushSender, now time.Time) DispatchResult {
	t.Helper()
	res, err := NewNotificationService(requireDB(t)).Dispatch(context.Background(), push, now)
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	return res
}

func TestTaskReminderArrivesBeforeTheAppointmentOnlyOnce(t *testing.T) {
	userID, device := subscribedUser(t)
	addTask(t, userID, "Dentista", "2030-03-12", "14:30", "planned")
	addTask(t, userID, "Já feito", "2030-03-12", "14:30", "done")
	addTask(t, userID, "Sem horário", "2030-03-12", "", "planned")
	addTask(t, userID, "Mais tarde", "2030-03-12", "16:00", "planned")
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "14:00")) // 30 min before: too early
	if got := push.to(device); len(got) != 0 {
		t.Fatalf("30 min before the default 15-min lead: got %+v, want nothing", got)
	}

	dispatch(t, push, at("2030-03-12", "14:17"))
	dispatch(t, push, at("2030-03-12", "14:22")) // next cron tick: no repeat
	got := push.to(device)
	if len(got) != 1 {
		t.Fatalf("got %d notifications, want exactly 1: %+v", len(got), got)
	}
	if got[0].Title != "Dentista" || !strings.Contains(got[0].Body, "14:30") || got[0].URL != "/planner" {
		t.Fatalf("reminder = %+v, want the Dentista task at 14:30 linking to the Planner", got[0])
	}
}

func TestAppointmentsStartingTogetherBecomeOneNotification(t *testing.T) {
	userID, device := subscribedUser(t)
	addTask(t, userID, "Reunião", "2030-03-12", "09:00", "planned")
	addTask(t, userID, "Ligação", "2030-03-12", "09:05", "in_progress")
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "08:52"))
	got := push.to(device)
	if len(got) != 1 {
		t.Fatalf("got %d notifications, want 1 consolidated: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Body, "Reunião") || !strings.Contains(got[0].Body, "Ligação") {
		t.Fatalf("consolidated body %q should list both appointments", got[0].Body)
	}
}

func TestRemindersFollowTheUsersTimezone(t *testing.T) {
	userID, device := subscribedUser(t)
	if _, err := requireDB(t).Exec(context.Background(),
		"UPDATE users SET timezone = 'Europe/Lisbon' WHERE id = $1", userID); err != nil {
		t.Fatal(err)
	}
	addTask(t, userID, "Aula", "2030-03-12", "10:00", "planned") // 10:00 in Lisbon = 07:00 in São Paulo
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "06:50"))
	if got := push.to(device); len(got) != 1 || got[0].Title != "Aula" {
		t.Fatalf("Lisbon user at 09:50 local: got %+v, want the Aula reminder", got)
	}
}

func TestTaskRemindersCanBeTurnedOff(t *testing.T) {
	userID, device := subscribedUser(t)
	off := DefaultNotificationSettings
	off.TaskReminders = false
	if _, err := NewNotificationService(requireDB(t)).Update(context.Background(), userID, off); err != nil {
		t.Fatal(err)
	}
	addTask(t, userID, "Dentista", "2030-03-12", "14:30", "planned")
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "14:20"))
	if got := push.to(device); len(got) != 0 {
		t.Fatalf("reminders off: got %+v, want nothing", got)
	}
}

func TestMorningDigestSummarizesTheDayOnce(t *testing.T) {
	userID, device := subscribedUser(t)
	addTask(t, userID, "Reunião", "2030-03-12", "10:00", "planned")
	addTask(t, userID, "Academia", "2030-03-12", "18:00", "planned")
	addHabit(t, userID, "Ler")
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "06:55")) // before 07:00
	if got := push.to(device); len(got) != 0 {
		t.Fatalf("before 07:00: got %+v", got)
	}
	dispatch(t, push, at("2030-03-12", "07:02"))
	dispatch(t, push, at("2030-03-12", "07:07"))
	got := push.to(device)
	if len(got) != 1 {
		t.Fatalf("got %d morning digests, want 1: %+v", len(got), got)
	}
	body := got[0].Body
	if !strings.Contains(body, "2 compromissos") || !strings.Contains(body, "10:00") || !strings.Contains(body, "1 hábito") {
		t.Fatalf("morning body %q should mention 2 appointments, the first at 10:00, and 1 habit", body)
	}
}

func TestNothingUsefulMeansNoNotification(t *testing.T) {
	_, device := subscribedUser(t) // no tasks, no habits
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "07:01"))
	if got := push.to(device); len(got) != 0 {
		t.Fatalf("empty day: got %+v, want nothing", got)
	}
}

func TestEveningDigestIsOptIn(t *testing.T) {
	userID, device := subscribedUser(t)
	read := addHabit(t, userID, "Ler")
	addHabit(t, userID, "Meditar")
	if _, err := requireDB(t).Exec(context.Background(),
		"INSERT INTO habit_logs (habit_id, user_id, logged_date) VALUES ($1, $2, '2030-03-12')", read, userID); err != nil {
		t.Fatal(err)
	}
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "21:01"))
	if got := push.to(device); len(got) != 0 {
		t.Fatalf("evening is off by default: got %+v", got)
	}

	on := DefaultNotificationSettings
	on.EveningDigest = true
	if _, err := NewNotificationService(requireDB(t)).Update(context.Background(), userID, on); err != nil {
		t.Fatal(err)
	}
	dispatch(t, push, at("2030-03-12", "21:06"))
	got := push.to(device)
	if len(got) != 1 || !strings.Contains(got[0].Body, "1 hábito") {
		t.Fatalf("evening digest = %+v, want one mentioning 1 open habit", got)
	}
}

func TestExpiredDeviceIsForgotten(t *testing.T) {
	userID, device := subscribedUser(t)
	addTask(t, userID, "Dentista", "2030-03-12", "14:30", "planned")
	push := newFakePush()
	push.gone[device] = true

	res := dispatch(t, push, at("2030-03-12", "14:20"))
	if res.Removed < 1 {
		t.Fatalf("result %+v, want the expired device removed", res)
	}
	if v, _ := NewNotificationService(requireDB(t)).Get(context.Background(), userID); v.Devices != 0 {
		t.Fatalf("devices = %d, want 0 after the push service said it is gone", v.Devices)
	}
}

func TestTasksMarkedNotToNotifyStayQuiet(t *testing.T) {
	userID, device := subscribedUser(t)
	addTask(t, userID, "Aula de Cálculo", "2030-03-12", "14:30", "planned")
	if _, err := requireDB(t).Exec(context.Background(),
		"UPDATE tasks SET notify = false WHERE user_id = $1", userID); err != nil {
		t.Fatal(err)
	}
	push := newFakePush()

	dispatch(t, push, at("2030-03-12", "14:20"))
	if got := push.to(device); len(got) != 0 {
		t.Fatalf("task with notify off: got %+v, want no reminder", got)
	}
}
