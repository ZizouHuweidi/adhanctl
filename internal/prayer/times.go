package prayer

import (
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zizouhuweidi/adhanctl/internal/api"
)

type Event struct {
	Name string
	When time.Time
}

type PrayerOrder []string

var StandardOrder PrayerOrder = []string{"Fajr", "Sunrise", "Dhuhr", "Asr", "Maghrib", "Isha"}

func ParseTimes(resp *api.Response, loc *time.Location) []Event {
	var events []Event

	gregDate := resp.Data.Date.Gregorian.Date
	if gregDate == "" {
		now := time.Now().In(loc)
		gregDate = now.Format("02-01-2006")
	}

	for _, name := range StandardOrder {
		tstr, ok := resp.Data.Timings[name]
		if !ok {
			continue
		}

		tok := strings.Fields(tstr)
		if len(tok) == 0 {
			continue
		}

		ts := tok[0]
		if i := strings.Index(ts, "("); i >= 0 {
			ts = strings.TrimSpace(ts[:i])
		}

		dt, err := parseDateTime(gregDate, ts, loc)
		if err != nil {
			slog.Default().Debug("parse time error", "prayer", name, "error", err)
			continue
		}

		events = append(events, Event{Name: name, When: dt})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].When.Before(events[j].When)
	})

	return events
}

func parseDateTime(gregDate, timeStr string, loc *time.Location) (time.Time, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	h, _ := strconv.Atoi(parts[0])
	m, _ := strconv.Atoi(parts[1])

	day, month, year := 0, 0, 0
	dateParts := strings.Split(gregDate, "-")
	if len(dateParts) >= 3 {
		day, _ = strconv.Atoi(dateParts[0])
		month, _ = strconv.Atoi(dateParts[1])
		year, _ = strconv.Atoi(dateParts[2])
	}

	if day == 0 || month == 0 || year == 0 {
		now := time.Now().In(loc)
		day = now.Day()
		month = int(now.Month())
		year = now.Year()
	}

	return time.Date(year, time.Month(month), day, h, m, 0, 0, loc), nil
}

func TimezoneFromResp(resp *api.Response) *time.Location {
	tz := resp.Data.Meta.Timezone
	if tz == "" {
		return time.Local
	}

	loc, err := time.LoadLocation(tz)
	if err != nil {
		slog.Default().Debug("unknown timezone", "tz", tz, "error", err)
		return time.Local
	}

	return loc
}

func NextEventAfter(events []Event, after time.Time) *Event {
	var best *Event
	for i := range events {
		if events[i].When.After(after) {
			if best == nil || events[i].When.Before(best.When) {
				cp := events[i]
				best = &cp
			}
		}
	}
	return best
}

func UpcomingEvents(events []Event, from time.Time, within time.Duration) []Event {
	var result []Event
	limit := from.Add(within)

	for _, e := range events {
		if e.When.After(from) && e.When.Before(limit) {
			result = append(result, e)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].When.Before(result[j].When)
	})

	return result
}

func HumanDuration(d time.Duration) string {
	if d < 0 {
		return "passed"
	}

	totalMins := int(d.Minutes())
	if totalMins < 60 {
		return fmt.Sprintf("%dm", totalMins)
	}

	h := totalMins / 60
	m := totalMins % 60

	if m == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh%02dm", h, m)
}

func HijriString(resp *api.Response, arabic bool) string {
	h := resp.Data.Date.Hijri

	var parts []string

	if h.Date != "" {
		parts = append(parts, h.Date)
	}

	month := h.Month.En
	if arabic {
		month = h.Month.Ar
	}
	if month != "" {
		parts = append(parts, month)
	}

	weekday := h.Weekday.En
	if arabic {
		weekday = h.Weekday.Ar
	}
	if weekday != "" {
		parts = append(parts, weekday)
	}

	return strings.Join(parts, " ")
}

func FormatTime(t time.Time, ampm bool) string {
	if ampm {
		return t.Format("03:04 PM")
	}
	return t.Format("15:04")
}
