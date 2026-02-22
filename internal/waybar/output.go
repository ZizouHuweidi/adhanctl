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

func Build(resp *api.Response, nextEvent *prayer.Event, events []prayer.Event, ampm, arabic, short bool) Output {
	loc := prayer.TimezoneFromResp(resp)
	now := time.Now().In(loc)

	var text string
	var tooltipLines []string

	hijri := prayer.HijriString(resp, arabic)
	if hijri != "" {
		tooltipLines = append(tooltipLines, fmt.Sprintf("📅 %s", hijri))
	}

	if nextEvent != nil {
		rem := prayer.HumanDuration(nextEvent.When.Sub(now))
		timeStr := prayer.FormatTime(nextEvent.When, ampm)
		if short {
			text = fmt.Sprintf("%s %s", nextEvent.Name, timeStr)
		} else {
			text = fmt.Sprintf("%s %s (%s)", nextEvent.Name, timeStr, rem)
		}
		tooltipLines = append(tooltipLines, fmt.Sprintf("Next: %s — %s", nextEvent.Name, rem))
	} else {
		text = "No upcoming prayer"
		tooltipLines = append(tooltipLines, "No upcoming prayer")
	}

	tooltipLines = append(tooltipLines, "", "Today's Schedule:")
	for _, name := range prayer.StandardOrder {
		for _, e := range events {
			if e.Name == name {
				marker := ""
				if now.After(e.When) {
					marker = " ✓"
				} else if nextEvent != nil && e.Name == nextEvent.Name {
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
