package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zizouhuweidi/adhanctl/internal/api"
	"github.com/zizouhuweidi/adhanctl/internal/cache"
	"github.com/zizouhuweidi/adhanctl/internal/config"
	"github.com/zizouhuweidi/adhanctl/internal/notify"
	"github.com/zizouhuweidi/adhanctl/internal/prayer"
	"github.com/zizouhuweidi/adhanctl/internal/waybar"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		runToday(os.Args[1:])
		return
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "today":
		runToday(args)
	case "next":
		runNext(args)
	case "notify":
		runNotify(args)
	case "serve":
		runServe(args)
	case "waybar":
		runWaybar(args)
	case "config":
		runConfig(args)
	case "version", "-v", "--version":
		fmt.Printf("adhanctl %s\n", version)
	case "help", "-h", "--help":
		printHelp()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", cmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`adhanctl - Prayer times CLI for Waybar and desktop notifications

Usage:
  adhanctl [command] [flags]

Commands:
  today       Show today's prayer schedule (default)
  next        Show next prayer with countdown
  notify      Send desktop notification for next prayer
  serve       Run background notifier daemon
  waybar      Output JSON for Waybar module
  config      Manage configuration
  version     Show version

Flags:
  -c, --city string          City name
  -C, --country string       Country name
      --lat, --latitude      Latitude (takes precedence over city)
      --lon, --longitude     Longitude (takes precedence over city)
  -m, --method int           Calculation method (default: 3)
  -s, --school int           Asr school: 0=Shafi, 1=Hanafi (default: 0)
      --ampm                 Use 12-hour format
      --ar                   Display Hijri in Arabic
  -v, --verbose              Enable debug logging
      --interval duration    Refresh interval for serve (default: 1m)

Waybar flags:
      --short                Short output (no countdown in text)

Run 'adhanctl config init' for first-time setup.`)
}

type flags struct {
	city      string
	country   string
	latitude  float64
	longitude float64
	method    int
	school    int
	ampm      bool
	arabic    bool
	verbose   bool
	interval  time.Duration
}

func parseFlags(args []string, cfg *config.Config) *flags {
	f := &flags{
		city:      cfg.City,
		country:   cfg.Country,
		latitude:  cfg.Latitude,
		longitude: cfg.Longitude,
		method:    cfg.Method,
		school:    cfg.School,
		ampm:      cfg.AmPm,
		arabic:    cfg.Arabic,
		interval:  cfg.Interval,
	}

	fs := flag.NewFlagSet("", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	fs.StringVar(&f.city, "city", f.city, "city name")
	fs.StringVar(&f.city, "c", f.city, "city name (shorthand)")
	fs.StringVar(&f.country, "country", f.country, "country name")
	fs.StringVar(&f.country, "C", f.country, "country name (shorthand)")
	fs.Float64Var(&f.latitude, "latitude", f.latitude, "latitude")
	fs.Float64Var(&f.latitude, "lat", f.latitude, "latitude (shorthand)")
	fs.Float64Var(&f.longitude, "longitude", f.longitude, "longitude")
	fs.Float64Var(&f.longitude, "lon", f.longitude, "longitude (shorthand)")
	fs.IntVar(&f.method, "method", f.method, "calculation method")
	fs.IntVar(&f.method, "m", f.method, "calculation method (shorthand)")
	fs.IntVar(&f.school, "school", f.school, "asr calculation school")
	fs.IntVar(&f.school, "s", f.school, "asr school (shorthand)")
	fs.BoolVar(&f.ampm, "ampm", f.ampm, "use 12-hour format")
	fs.BoolVar(&f.arabic, "ar", f.arabic, "display Hijri in Arabic")
	fs.BoolVar(&f.verbose, "verbose", false, "enable debug logging")
	fs.BoolVar(&f.verbose, "v", false, "enable debug logging (shorthand)")
	fs.DurationVar(&f.interval, "interval", f.interval, "refresh interval for serve")

	_ = fs.Parse(args)

	return f
}

func setupLogger(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

func buildParams(f *flags) api.TimingsParams {
	return api.TimingsParams{
		City:      f.city,
		Country:   f.country,
		Latitude:  f.latitude,
		Longitude: f.longitude,
		Method:    f.method,
		School:    f.school,
		Date:      time.Now(),
	}
}

func fetchWithCache(ctx context.Context, cfg *config.Config, f *flags) (*api.Response, error) {
	client := api.NewClient()
	c := cache.New(time.Duration(cfg.CacheSecs) * time.Second)

	params := buildParams(f)

	if resp, ok := c.Get(params); ok {
		return resp, nil
	}

	resp, err := client.FetchTimings(ctx, params)
	if err != nil {
		return nil, err
	}

	_ = c.Set(params, resp)
	return resp, nil
}

func validateLocation(f *flags) error {
	if f.latitude != 0 && f.longitude != 0 {
		return nil
	}
	if f.city != "" && f.country != "" {
		return nil
	}
	return fmt.Errorf("no location provided: use --city/--country or --lat/--lon, or run 'adhanctl config init'")
}

func runToday(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	f := parseFlags(args, cfg)
	setupLogger(f.verbose)

	if err := validateLocation(f); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	resp, err := fetchWithCache(ctx, cfg, f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching timings: %v\n", err)
		os.Exit(1)
	}

	loc := prayer.TimezoneFromResp(resp)
	events := prayer.ParseTimes(resp, loc)
	now := time.Now().In(loc)

	hijri := prayer.HijriString(resp, f.arabic)
	fmt.Printf("\n📅 %s\n\n", hijri)

	fmt.Println("Today's Prayer Schedule:")
	fmt.Println(strings.Repeat("-", 24))

	for _, name := range prayer.StandardOrder {
		for _, e := range events {
			if e.Name == name {
				marker := ""
				if now.After(e.When) {
					marker = " ✓"
				}
				fmt.Printf("  %-8s %s%s\n", name, prayer.FormatTime(e.When, f.ampm), marker)
				break
			}
		}
	}

	next := prayer.NextEventAfter(events, now)
	if next != nil {
		rem := prayer.HumanDuration(next.When.Sub(now))
		fmt.Printf("\n🕌 Next: %s in %s\n", next.Name, rem)
	}
}

func runNext(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	f := parseFlags(args, cfg)
	setupLogger(f.verbose)

	if err := validateLocation(f); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	resp, err := fetchWithCache(ctx, cfg, f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching timings: %v\n", err)
		os.Exit(1)
	}

	loc := prayer.TimezoneFromResp(resp)
	events := prayer.ParseTimes(resp, loc)
	now := time.Now().In(loc)

	next := prayer.NextEventAfter(events, now)
	if next == nil {
		fmt.Println("No upcoming prayer found")
		os.Exit(0)
	}

	rem := prayer.HumanDuration(next.When.Sub(now))
	timeStr := prayer.FormatTime(next.When, f.ampm)

	fmt.Printf("🕌 %s at %s (%s)\n", next.Name, timeStr, rem)

	hijri := prayer.HijriString(resp, f.arabic)
	if hijri != "" {
		fmt.Printf("📅 %s\n", hijri)
	}
}

func runNotify(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	f := parseFlags(args, cfg)
	setupLogger(f.verbose)

	if err := validateLocation(f); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	resp, err := fetchWithCache(ctx, cfg, f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching timings: %v\n", err)
		os.Exit(1)
	}

	loc := prayer.TimezoneFromResp(resp)
	events := prayer.ParseTimes(resp, loc)
	now := time.Now().In(loc)

	next := prayer.NextEventAfter(events, now)
	if next == nil {
		fmt.Fprintln(os.Stderr, "no upcoming prayer found")
		os.Exit(1)
	}

	hijri := prayer.HijriString(resp, f.arabic)
	notify.Prayer(*next, hijri)

	fmt.Printf("Sent notification: %s at %s\n", next.Name, prayer.FormatTime(next.When, f.ampm))
}

func runServe(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	f := parseFlags(args, cfg)
	setupLogger(f.verbose)

	if err := validateLocation(f); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	interval := max(f.interval, 10*time.Second)

	slog.Info("starting serve mode", "interval", interval)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	client := api.NewClient()
	c := cache.New(time.Duration(cfg.CacheSecs) * time.Second)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	scheduleEvents := func() {
		params := buildParams(f)

		var resp *api.Response
		if cached, ok := c.Get(params); ok {
			resp = cached
		} else {
			var err error
			resp, err = client.FetchTimings(ctx, params)
			if err != nil {
				slog.Debug("fetch error", "error", err)
				return
			}
			_ = c.Set(params, resp)
		}

		loc := prayer.TimezoneFromResp(resp)
		events := prayer.ParseTimes(resp, loc)

		if len(events) == 0 {
			slog.Debug("no prayer times parsed")
			return
		}

		now := time.Now().In(loc)
		upcoming := prayer.UpcomingEvents(events, now, 24*time.Hour)

		slog.Debug("scheduling events", "count", len(upcoming))

		hijri := prayer.HijriString(resp, f.arabic)

		for _, ev := range upcoming {
			if ev.When.Before(now) {
				continue
			}
			go scheduleNotification(ev, hijri)
		}
	}

	scheduleEvents()

	for {
		select {
		case <-ctx.Done():
			slog.Info("shutting down")
			return
		case <-ticker.C:
			scheduleEvents()
		}
	}
}

func scheduleNotification(ev prayer.Event, hijri string) {
	d := time.Until(ev.When)
	slog.Debug("scheduled notification", "prayer", ev.Name, "in", d)

	timer := time.NewTimer(d)
	defer timer.Stop()

	<-timer.C
	notify.Prayer(ev, hijri)
}

func runWaybar(args []string) {
	cfg, err := config.Load()
	if err != nil {
		waybar.Print(waybar.Output{Text: "adhanctl: config error", Tooltip: err.Error()})
		os.Exit(0)
	}

	short := cfg.Short
	waybarArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "--short" {
			short = true
		} else {
			waybarArgs = append(waybarArgs, args[i])
		}
	}

	f := parseFlags(waybarArgs, cfg)
	setupLogger(f.verbose)

	if err := validateLocation(f); err != nil {
		waybar.Print(waybar.Output{Text: "adhanctl: no location", Tooltip: err.Error()})
		os.Exit(0)
	}

	ctx := context.Background()
	resp, err := fetchWithCache(ctx, cfg, f)
	if err != nil {
		waybar.Print(waybar.Output{Text: "adhanctl: error", Tooltip: err.Error()})
		os.Exit(0)
	}

	out := waybar.Build(resp, f.ampm, f.arabic, short)
	waybar.Print(out)
}

func runConfig(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "config subcommand required: init, show")
		os.Exit(1)
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "init":
		runConfigInit(subArgs)
	case "show":
		runConfigShow(subArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown config subcommand: %s\n", sub)
		os.Exit(1)
	}
}

func runConfigInit(args []string) {
	fs := flag.NewFlagSet("config init", flag.ContinueOnError)
	verbose := fs.Bool("v", false, "verbose")
	_ = fs.Parse(args)

	setupLogger(*verbose)

	interactor := &config.StdioInteractor{}
	cfg, err := config.RunConfigInit(interactor)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nConfiguration saved to %s\n", config.ConfigPath())
	fmt.Println("\nYou can now run:")
	fmt.Println("  adhanctl today  - View today's schedule")
	fmt.Println("  adhanctl serve  - Run background notifier")
	fmt.Println("  adhanctl waybar - Output for Waybar")
}

func runConfigShow(args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config file: %s\n\n", config.ConfigPath())
	fmt.Println("Current configuration:")

	if cfg.City != "" {
		fmt.Printf("  City:      %s\n", cfg.City)
	}
	if cfg.Country != "" {
		fmt.Printf("  Country:   %s\n", cfg.Country)
	}
	if cfg.Latitude != 0 {
		fmt.Printf("  Latitude:  %.6f\n", cfg.Latitude)
	}
	if cfg.Longitude != 0 {
		fmt.Printf("  Longitude: %.6f\n", cfg.Longitude)
	}
	fmt.Printf("  Method:    %d (%s)\n", cfg.Method, config.CalculationMethods[cfg.Method])
	fmt.Printf("  School:    %d (%s)\n", cfg.School, config.Schools[cfg.School])
	fmt.Printf("  12-hour:   %t\n", cfg.AmPm)
	fmt.Printf("  Arabic:    %t\n", cfg.Arabic)
	fmt.Printf("  Short:     %t\n", cfg.Short)
	fmt.Printf("  Cache:     %d seconds\n", cfg.CacheSecs)
	fmt.Printf("  Interval:  %s\n", cfg.Interval)
}
