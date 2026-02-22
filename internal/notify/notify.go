package notify

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"github.com/zizouhuweidi/adhanctl/internal/prayer"
)

func Desktop(summary, body string) error {
	if cmd, err := exec.LookPath("notify-send"); err == nil {
		return exec.Command(cmd, summary, body).Run()
	}

	if cmd, err := exec.LookPath("dunstify"); err == nil {
		return exec.Command(cmd, summary, body).Run()
	}

	fmt.Fprintf(os.Stderr, "NOTIFY: %s - %s\n", summary, body)
	return nil
}

func Prayer(ev prayer.Event, hijri string) {
	title := fmt.Sprintf("🕌 %s", ev.Name)
	body := fmt.Sprintf("%s at %s", ev.Name, ev.When.Format(time.Kitchen))

	if hijri != "" {
		body = fmt.Sprintf("%s\n%s", hijri, body)
	}

	if err := Desktop(title, body); err != nil {
		slog.Default().Debug("notification error", "error", err)
	}
}
