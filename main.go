package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := LoadConfig()
	if cfg.Token == "" {
		log.Fatal("SERVICE_TOKEN belum diisi. Isi di file .env (harus sama dengan CHECKER_SERVICE_TOKEN di Laravel)")
	}

	// Ctrl+C menghentikan service dengan rapi.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := NewClient(cfg)
	log.Printf("checker berjalan: laravel=%s interval=%s workers=%d", cfg.LaravelURL, cfg.Interval, cfg.Workers)

	runOnce(ctx, client, cfg)

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("checker berhenti")
			return
		case <-ticker.C:
			runOnce(ctx, client, cfg)
		}
	}
}

func runOnce(ctx context.Context, client *Client, cfg Config) {
	monitors, err := client.FetchMonitors(ctx)
	if err != nil {
		log.Printf("gagal mengambil monitor: %v", err)
		return
	}
	if len(monitors) == 0 {
		return
	}

	start := time.Now()
	results := RunChecks(ctx, monitors, cfg.Workers)

	for _, r := range results {
		if r.Status == "up" {
			log.Printf("[UP]   monitor=%d code=%d %dms", r.MonitorID, r.StatusCode, r.ResponseMs)
		} else {
			log.Printf("[DOWN] monitor=%d code=%d %dms error=%s", r.MonitorID, r.StatusCode, r.ResponseMs, r.Error)
		}
	}

	if err := client.SendResults(ctx, results); err != nil {
		log.Printf("gagal mengirim hasil ke Laravel: %v", err)
		return
	}
	log.Printf("terkirim %d hasil (%s)", len(results), time.Since(start).Round(time.Millisecond))
}
