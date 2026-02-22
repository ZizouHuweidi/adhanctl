package cache

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zizouhuweidi/adhanctl/internal/api"
)

const CacheDirName = "adhanctl"

type Cache struct {
	Dir    string
	TTL    time.Duration
	Logger *slog.Logger
}

func New(ttl time.Duration) *Cache {
	return &Cache{
		Dir:    xdgCacheDir(),
		TTL:    ttl,
		Logger: slog.Default(),
	}
}

func xdgCacheDir() string {
	if x := os.Getenv("XDG_CACHE_HOME"); x != "" {
		return filepath.Join(x, CacheDirName)
	}
	home := os.Getenv("HOME")
	if home == "" {
		home = "."
	}
	return filepath.Join(home, ".cache", CacheDirName)
}

func (c *Cache) filePath(params api.TimingsParams) string {
	var key string
	if params.Latitude != 0 && params.Longitude != 0 {
		key = fmt.Sprintf("coords-%.4f-%.4f", params.Latitude, params.Longitude)
	} else {
		key = fmt.Sprintf("city-%s-%s", sanitize(params.City), sanitize(params.Country))
	}
	date := params.Date.Format("2006-01-02")
	filename := fmt.Sprintf("%s-method%d-school%d-%s.json", key, params.Method, params.School, date)
	return filepath.Join(c.Dir, filename)
}

func (c *Cache) Get(params api.TimingsParams) (*api.Response, bool) {
	path := c.filePath(params)
	if c.TTL <= 0 {
		return nil, false
	}

	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}

	if time.Since(info.ModTime()) > c.TTL {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var resp api.Response
	if err := json.Unmarshal(data, &resp); err != nil {
		c.Logger.Debug("cache unmarshal failed", "error", err)
		return nil, false
	}

	c.Logger.Debug("cache hit", "path", path)
	return &resp, true
}

func (c *Cache) Set(params api.TimingsParams, resp *api.Response) error {
	if err := os.MkdirAll(c.Dir, 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	path := c.filePath(params)
	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshaling response: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing cache file: %w", err)
	}

	c.Logger.Debug("cache written", "path", path)
	return nil
}

func sanitize(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result = append(result, r)
		} else if r == ' ' {
			result = append(result, '_')
		}
	}
	if len(result) == 0 {
		return "default"
	}
	return string(result)
}
