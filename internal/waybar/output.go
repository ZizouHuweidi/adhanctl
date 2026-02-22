package waybar

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/zizouhuweidi/adhanctl/internal/api"
	"github.com/zizouhuweidi/adhanctl/internal/prayer"
)

type Output struct {
	Text    string `json:"text"`
	Tooltip string `json:"tooltip,omitempty"`
	Class   string `json:"class,omitempty"`
}

func Build(resp *api.Response, ampm, arabic, short bool) Output {
	loc := prayer.TimezoneFromResp(resp)
	events := prayer.ParseTimes(resp, loc)
	now := time.Now().In(loc)

	next := prayer.NextEventAfter(events, now)

	var text string
	var tooltipLines []string

	hijri := prayer.HijriString(resp, arabic)
	if hijri != "" {
		tooltipLines = append(tooltipLines, fmt.Sprintf("📅 %s", hijri))
	}

	if next != nil {
		rem := prayer.HumanDuration(next.When.Sub(now))
		timeStr := prayer.FormatTime(next.When, ampm)
		if short {
			text = fmt.Sprintf("%s %s", next.Name, timeStr)
		} else {
			text = fmt.Sprintf("%s %s (%s)", next.Name, timeStr, rem)
		}
		tooltipLines = append(tooltipLines, fmt.Sprintf("Next: %s — %s", next.Name, rem))
	} else {
		text = "Internal Error"
		tooltipLines = append(tooltipLines, "Internal Error")
	}

	tooltipLines = append(tooltipLines, "", "Today's Schedule:")
	for _, name := range prayer.StandardOrder {
		for _, e := range events {
			if e.Name == name {
				marker := ""
				if now.After(e.When) {
					marker = " ✓"
				} else if next != nil && e.Name == next.Name {
					marker = " ←"
				}
				tooltipLines = append(tooltipLines,
					fmt.Sprintf("  %-8s %s%s", name, prayer.FormatTime(e.When, ampm), marker))
				break
			}
		}
	}

	return Output{
		Text:    text,
		Tooltip: strings.Join(tooltipLines, "\n"),
		Class:   "adhan",
	}
}

func Print(out Output) error {
	data, err := json.Marshal(out)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
