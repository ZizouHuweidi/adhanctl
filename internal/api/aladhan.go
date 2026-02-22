package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL       = "https://api.aladhan.com/v1"
	UserAgent     = "adhanctl/1.0"
	DefaultMethod = 2
	MaxRetries    = 6
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Logger     *slog.Logger
}

func NewClient() *Client {
	return &Client{
		BaseURL: BaseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		Logger: slog.Default(),
	}
}

type Response struct {
	Code int    `json:"code"`
	Data Data   `json:"data"`
	Msg  string `json:"status"`
}

type Data struct {
	Timings map[string]string `json:"timings"`
	Date    Date              `json:"date"`
	Meta    Meta              `json:"meta"`
}

type Date struct {
	Gregorian Gregorian `json:"gregorian"`
	Hijri     Hijri     `json:"hijri"`
}

type Gregorian struct {
	Date    string `json:"date"`
	Format  string `json:"format"`
	Day     string `json:"day"`
	Weekday struct {
		En string `json:"en"`
	} `json:"weekday"`
	Month struct {
		Number int    `json:"number"`
		En     string `json:"en"`
	} `json:"month"`
	Year  string `json:"year"`
	Hijri string `json:"hijri"`
}

type Hijri struct {
	Date    string `json:"date"`
	Format  string `json:"format"`
	Day     string `json:"day"`
	Weekday struct {
		En string `json:"en"`
		Ar string `json:"ar"`
	} `json:"weekday"`
	Month struct {
		Number int    `json:"number"`
		En     string `json:"en"`
		Ar     string `json:"ar"`
	} `json:"month"`
	Year      string `json:"year"`
	Gregorian string `json:"gregorian"`
}

type Meta struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Timezone  string  `json:"timezone"`
	Method    struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"method"`
	School any `json:"school"`
}

type TimingsParams struct {
	City      string
	Country   string
	Latitude  float64
	Longitude float64
	Method    int
	School    int
	Date      time.Time
}

func (c *Client) FetchTimings(ctx context.Context, params TimingsParams) (*Response, error) {
	var apiURL string
	dateStr := params.Date.Format("02-01-2006")

	if params.Latitude != 0 && params.Longitude != 0 {
		apiURL = fmt.Sprintf("%s/timings/%s?latitude=%f&longitude=%f&method=%d",
			c.BaseURL, dateStr, params.Latitude, params.Longitude, params.Method)
	} else {
		apiURL = fmt.Sprintf("%s/timingsByCity/%s?city=%s&country=%s&method=%d",
			c.BaseURL, dateStr,
			url.QueryEscape(params.City),
			url.QueryEscape(params.Country),
			params.Method)
	}

	if params.School != 0 {
		apiURL += fmt.Sprintf("&school=%d", params.School)
	}

	return c.fetchWithRetries(ctx, apiURL)
}

func (c *Client) fetchWithRetries(ctx context.Context, apiURL string) (*Response, error) {
	var lastErr error
	backoff := 500 * time.Millisecond

	for i := range MaxRetries {
		resp, err := c.fetchURL(ctx, apiURL)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		c.Logger.Debug("fetch attempt failed, retrying",
			"attempt", i+1,
			"error", err,
			"backoff", backoff)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > 8*time.Second {
			backoff = 8 * time.Second
		}
	}

	return nil, fmt.Errorf("fetch failed after %d retries: %w", MaxRetries, lastErr)
}

func (c *Client) fetchURL(ctx context.Context, apiURL string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("api status %d: %s", resp.StatusCode, string(body))
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if result.Code != 200 {
		return &result, fmt.Errorf("api error code %d: %s", result.Code, result.Msg)
	}

	return &result, nil
}
