package main

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config berisi pengaturan service checker.
type Config struct {
	LaravelURL string        // alamat Laravel, contoh http://127.0.0.1:8000
	Token      string        // harus sama dengan CHECKER_SERVICE_TOKEN di Laravel
	Interval   time.Duration // seberapa sering menanyakan monitor yang jatuh tempo
	Workers    int           // jumlah goroutine pengecek paralel
}

// loadDotEnv membaca file .env sederhana (KEY=VALUE).
// Variabel yang sudah ada di environment tidak ditimpa.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func LoadConfig() Config {
	loadDotEnv(".env")
	return Config{
		LaravelURL: strings.TrimRight(getenv("LARAVEL_URL", "http://127.0.0.1:8000"), "/"),
		Token:      getenv("SERVICE_TOKEN", ""),
		Interval:   time.Duration(getenvInt("TICK_SECONDS", 10)) * time.Second,
		Workers:    getenvInt("WORKERS", 20),
	}
}
