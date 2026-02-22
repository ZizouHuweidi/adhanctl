# adhanctl

A minimal CLI tool for Islamic prayer times, designed for Waybar integration and desktop notifications.

## Features

- Prayer times from [AlAdhan API](https://aladhan.com/)
- Waybar module support with JSON output
- Desktop notifications (notify-send)
- Background daemon mode for automatic notifications
- Hijri date display (English or Arabic)

## Installation

### Build from Source

**Requirements:** Go 1.23 or later

```bash
git clone https://github.com/zizouhuweidi/adhanctl.git
cd adhanctl

# User install (default: ~/.local/bin)
make install

# System-wide install
sudo make install PREFIX=/usr/local
```

## Quick Start

### First Run Setup

```bash
adhanctl config init
```

This will prompt you for:
- City and country
- Calculation method (check [reference](https://aladhan.com/calculation-methods))
- Fiqh school (Shafi or Hanafi)

### View Today's Schedule

```bash
adhanctl today
```

### Next Prayer

```bash
adhanctl next
```

## Commands

```
adhanctl [command] [flags]

Commands:
  today       Show today's prayer schedule (default)
  next        Show next prayer with countdown
  notify      Send desktop notification for next prayer
  serve       Run background notifier daemon
  waybar      Output JSON for Waybar module
  config      Manage configuration (init, show)
  version     Show version
```

### Flags

```
  -c, --city string          City name
  -C, --country string       Country name
      --lat, --latitude      Latitude (takes precedence over city)
      --lon, --longitude     Longitude (takes precedence over city)
  -m, --method int           Calculation method (default: 3)
  -s, --school int           Asr calculation school: 0=Shafi, 1=Hanafi (default: 0)
      --ampm                 Use 12-hour format
      --ar                   Display Hijri in Arabic
  -v, --verbose              Enable debug logging
      --interval duration    Refresh interval for serve (default: 1m)
```

## Waybar Integration

Example integration to your Waybar config:

```json
"custom/adhanctl": {
  "format": "{text} 🕌",
  "tooltip": true,
  "interval": 60,
  "exec": "adhanctl waybar",
  "return-type": "json",
},
```

For short output (no countdown in text):

```json
"custom/adhanctl": {
  "format": "{text} 🕌",
  "tooltip": true,
  "interval": 60,
  "exec": "adhanctl waybar --short",
  "return-type": "json",
},
```

Add to your `style.css`:

```css
#custom-adhanctl {
  padding: 0 1.5em 0 0.3em;
  border-radius: 0 10 10 0;
  color: #98971a;
}
```

## Background Service

For automatic notifications, configure `adhanctl serve` to run at startup, either manually through your DE/WM config or using systemd

### Sway

```
exec adhanctl serve
```

### Systemd Service

Create `~/.config/systemd/user/adhanctl.service`:

```ini
[Unit]
Description=Adhanctl Prayer Time Notifier
After=graphical-session.target

[Service]
Type=simple
ExecStart=adhanctl serve
Restart=on-failure
RestartSec=10

[Install]
WantedBy=default.target
```

Enable and start:

```bash
systemctl --user enable --now adhanctl
```

## Configuration

Config file location: `~/.config/adhanctl/config`

Example configuration:

```
city = London
country = United Kingdom
latitude = 51.5074
longitude = -0.1278
method = 3
school = 0
ampm = false
arabic = false
short = false
cache_secs = 10800
interval = 1m
```

### Configuration Options

| Option | Description | Default |
|--------|-------------|---------|
| `city` | City name | - |
| `country` | Country name | - |
| `latitude` | Latitude (takes precedence) | - |
| `longitude` | Longitude (takes precedence) | - |
| `method` | Calculation method | 3 |
| `school` | Asr Calculation school (0=Shafi, 1=Hanafi) | 0 |
| `ampm` | Use 12-hour format | false |
| `arabic` | Display Hijri in Arabic | false |
| `short` | Short output for Waybar (no countdown) | false |
| `cache_secs` | Cache TTL in seconds | 10800 |
| `interval` | Refresh interval for serve | 1m |


# Credit
- https://github.com/Onizuka893/prayerbar
- https://github.com/abdalrahmanshaban0/Islamic-Prayer-Timings
