package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Monitor adalah data yang diterima dari Laravel (GET /api/internal/monitors).
type Monitor struct {
	ID             int    `json:"id"`
	URL            string `json:"url"`
	Method         string `json:"method"`
	TimeoutMs      int    `json:"timeout_ms"`
	ExpectedStatus int    `json:"expected_status"`
}

// Result adalah data yang dikirim ke Laravel (POST /api/internal/results).
type Result struct {
	MonitorID  int    `json:"monitor_id"`
	Status     string `json:"status"` // "up" atau "down"
	StatusCode int    `json:"status_code,omitempty"`
	ResponseMs int64  `json:"response_ms"`
	Error      string `json:"error,omitempty"`
}

// Satu http.Client dipakai bersama agar koneksi bisa dipakai ulang.
// Timeout per monitor diatur lewat context, bukan lewat client.
var sharedClient = &http.Client{}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// Check mengecek satu monitor.
func Check(ctx context.Context, m Monitor) Result {
	timeout := time.Duration(m.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	method := strings.ToUpper(m.Method)
	if method == "" {
		method = http.MethodGet
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	res := Result{MonitorID: m.ID, Status: "down"}

	req, err := http.NewRequestWithContext(ctx, method, m.URL, nil)
	if err != nil {
		res.Error = truncate("request tidak valid: "+err.Error(), 500)
		return res
	}
	req.Header.Set("User-Agent", "UptimeChecker/1.0")

	start := time.Now()
	resp, err := sharedClient.Do(req)
	res.ResponseMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Error = truncate(err.Error(), 500)
		return res
	}
	defer resp.Body.Close()
	// Baca sebagian body agar koneksi bisa dipakai ulang, batasi 1 MB.
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	res.StatusCode = resp.StatusCode
	if resp.StatusCode == m.ExpectedStatus {
		res.Status = "up"
	} else {
		res.Error = "status tidak sesuai harapan"
	}
	return res
}

// RunChecks mengecek banyak monitor secara paralel dengan worker pool.
func RunChecks(ctx context.Context, monitors []Monitor, workers int) []Result {
	if workers > len(monitors) {
		workers = len(monitors)
	}

	jobs := make(chan Monitor)
	out := make(chan Result)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for m := range jobs {
				out <- Check(ctx, m)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, m := range monitors {
			select {
			case jobs <- m:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		wg.Wait()
		close(out)
	}()

	results := make([]Result, 0, len(monitors))
	for r := range out {
		results = append(results, r)
	}
	return results
}
