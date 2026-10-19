package service

import (
	"context"
	"strings"
	"time"

	"github.com/myroutine/backend/internal/appctx"
)

// Constância por hábito (bloco do Início): os últimos N dias de cada hábito,
// um caractere por dia, e contagens neutras — sem porcentagem, nota ou placar
// (product-constitution). Dia sem registro é estado neutro, não falha.
const (
	dayDone        = 'x' // registro feito
	dayOpen        = 'o' // programado, sem registro
	dayOff         = '-' // não é dia do hábito
	dayToday       = '.' // hoje, ainda sem registro: o dia não acabou
	dayBeforeHabit = ' ' // o hábito ainda não existia
)

type HabitConsistency struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
	// Days has one character per day, oldest first, ending today.
	Days              string `json:"days"`
	Done              int    `json:"done"`
	Scheduled         int    `json:"scheduled"`
	PreviousDone      int    `json:"previous_done"`
	PreviousScheduled int    `json:"previous_scheduled"`
}

type ConsistencyDTO struct {
	From   string             `json:"from"`
	To     string             `json:"to"`
	Habits []HabitConsistency `json:"habits"`
}

// Consistency returns the last 90 days of every active habit in the user's
// timezone.
func (s *HabitService) Consistency(ctx context.Context, userID string) (*ConsistencyDTO, error) {
	return s.consistency(ctx, userID, appctx.Today(ctx), 90)
}

func (s *HabitService) consistency(ctx context.Context, userID string, today time.Time, days int) (*ConsistencyDTO, error) {
	habits, err := s.listByUser(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	logs, err := s.logDatesByHabit(ctx, userID, today)
	if err != nil {
		return nil, err
	}

	today = calendarDay(today)
	from := today.AddDate(0, 0, -(days - 1))
	prevFrom := from.AddDate(0, 0, -days)
	out := &ConsistencyDTO{From: from.Format(dateLayout), To: today.Format(dateLayout), Habits: []HabitConsistency{}}

	for _, h := range habits {
		target := make(map[int32]bool, len(h.TargetDays))
		for _, d := range h.TargetDays {
			target[d] = true
		}
		logged := logs[h.ID]
		// The habit "exists" from its creation or its earliest check-in,
		// whichever comes first (check-ins can be registered for past days).
		created := calendarDay(h.CreatedAt.In(today.Location()))
		for d := range logged {
			if t, err := time.ParseInLocation(dateLayout, d, today.Location()); err == nil && t.Before(created) {
				created = t
			}
		}

		hc := HabitConsistency{ID: h.ID, Name: h.Name, Icon: h.Icon, Color: h.Color}
		var b strings.Builder
		for day := from; !day.After(today); day = day.AddDate(0, 0, 1) {
			key := day.Format(dateLayout)
			scheduled := target[isoWeekday(day)]
			switch {
			case day.Before(created):
				b.WriteRune(dayBeforeHabit)
			case logged[key]:
				b.WriteRune(dayDone)
				if scheduled {
					hc.Done++
					hc.Scheduled++
				}
			case day.Equal(today):
				if scheduled {
					b.WriteRune(dayToday)
				} else {
					b.WriteRune(dayOff)
				}
			case scheduled:
				b.WriteRune(dayOpen)
				hc.Scheduled++
			default:
				b.WriteRune(dayOff)
			}
		}
		hc.Days = b.String()

		for day := prevFrom; day.Before(from); day = day.AddDate(0, 0, 1) {
			if day.Before(created) || !target[isoWeekday(day)] {
				continue
			}
			hc.PreviousScheduled++
			if logged[day.Format(dateLayout)] {
				hc.PreviousDone++
			}
		}
		out.Habits = append(out.Habits, hc)
	}
	return out, nil
}
