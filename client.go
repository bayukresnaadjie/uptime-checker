package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client berbicara dengan API internal Laravel.
type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.cfg.LaravelURL+path, rdr)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s -> HTTP %d: %s", method, path, resp.StatusCode, msg)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// FetchMonitors mengambil monitor yang sudah waktunya dicek.
func (c *Client) FetchMonitors(ctx context.Context) ([]Monitor, error) {
	var monitors []Monitor
	if err := c.do(ctx, http.MethodGet, "/api/internal/monitors", nil, &monitors); err != nil {
		return nil, err
	}
	return monitors, nil
}

// SendResults mengirim hasil ke Laravel. Laravel membatasi 1000 hasil per
// request, jadi dikirim per potongan 500.
func (c *Client) SendResults(ctx context.Context, results []Result) error {
	const chunk = 500
	for i := 0; i < len(results); i += chunk {
		end := min(i+chunk, len(results))
		payload := map[string]any{"results": results[i:end]}
		if err := c.do(ctx, http.MethodPost, "/api/internal/results", payload, nil); err != nil {
			return err
		}
	}
	return nil
}
