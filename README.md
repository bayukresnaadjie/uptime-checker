# Uptime Checker

Service pengecek uptime yang ditulis dengan **Go**. Service ini mengambil daftar monitor dari aplikasi [uptime-monitor](https://github.com/bayukresnaadjie/uptime-monitor) (Laravel), mengecek semua URL secara paralel, lalu mengirim hasilnya kembali.

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)

Hanya memakai **library bawaan Go**, tanpa dependensi eksternal.

## Cara kerja

```
Setiap TICK_SECONDS:
  1. GET  /api/internal/monitors   → monitor yang sudah jatuh tempo
  2. Cek semua URL paralel (worker pool + goroutine)
  3. POST /api/internal/results    → kirim hasil dalam satu batch
```

- Laravel hanya mengembalikan monitor yang sudah lewat `interval_seconds`-nya, jadi service ini cukup bertanya berkala.
- Service ini tidak menyentuh database. Semua data lewat API internal Laravel.
- Jika pengiriman hasil gagal, monitor tetap dianggap jatuh tempo dan akan dicek lagi di siklus berikutnya.

## Kebutuhan

- Go 1.22 atau lebih baru
- Aplikasi [uptime-monitor](https://github.com/bayukresnaadjie/uptime-monitor) yang sedang berjalan

## Instalasi

```bash
git clone https://github.com/bayukresnaadjie/uptime-checker.git
cd uptime-checker
cp .env.example .env        # Windows: copy .env.example .env
```

Edit `.env`:

```env
LARAVEL_URL=http://127.0.0.1:8000
SERVICE_TOKEN=isi_sama_dengan_CHECKER_SERVICE_TOKEN_di_laravel
TICK_SECONDS=10
WORKERS=20
```

## Konfigurasi

| Variabel | Bawaan | Keterangan |
|---|---|---|
| `LARAVEL_URL` | `http://127.0.0.1:8000` | Alamat aplikasi Laravel |
| `SERVICE_TOKEN` | *(wajib)* | Harus sama persis dengan `CHECKER_SERVICE_TOKEN` di Laravel |
| `TICK_SECONDS` | `10` | Seberapa sering menanyakan monitor yang jatuh tempo |
| `WORKERS` | `20` | Jumlah pengecekan yang berjalan bersamaan |

Variabel yang sudah ada di environment sistem tidak ditimpa oleh isi `.env`.

## Menjalankan

```bash
go run .
```

Atau build menjadi file eksekusi:

```bash
go build -o checker .       # Windows: go build -o checker.exe .
./checker
```

Jalankan dari dalam folder proyek agar file `.env` terbaca. Hentikan dengan `Ctrl+C`, dan service akan berhenti dengan rapi.

Contoh keluaran:

```
checker berjalan: laravel=http://127.0.0.1:8000 interval=10s workers=20
[UP]   monitor=1 code=200 95ms
terkirim 1 hasil (900ms)
```

## Aturan pengecekan

- Monitor dianggap **up** jika status code sama dengan `expected_status`, dan **down** jika berbeda atau koneksi gagal.
- Batas waktu mengikuti `timeout_ms` tiap monitor (bawaan 10 detik jika kosong).
- Redirect diikuti secara otomatis (maksimal 10 kali, bawaan Go).
- Hasil dikirim ke Laravel per potongan 500 hasil, mengikuti batas 1000 per request di sisi Laravel.

## Struktur

```
main.go      # Loop utama, ticker, shutdown rapi
config.go    # Memuat .env dan konfigurasi
checker.go   # Logika cek URL dan worker pool
client.go    # Klien HTTP ke API internal Laravel
```

## Pemecahan masalah

| Pesan | Penyebab |
|---|---|
| `SERVICE_TOKEN belum diisi` | File `.env` belum dibuat atau salah nama |
| `HTTP 401` | Token berbeda dengan `CHECKER_SERVICE_TOKEN` di Laravel |
| `connection refused` ke port 8000 | Aplikasi Laravel belum berjalan |
| `go: no go files listed` | Perintah tanpa titik. Gunakan `go run .` |

## Batasan saat ini

- Belum ada perlindungan SSRF pada saat koneksi dibuat. Validasi URL saat ini hanya di form Laravel.
- Belum ada percobaan ulang untuk pengiriman hasil yang gagal.
- Hanya mendukung pengecekan HTTP/HTTPS dengan method GET dan HEAD.

## Roadmap

- [x] Pengecekan paralel dengan worker pool
- [x] Pengiriman hasil secara batch
- [x] Shutdown yang rapi
- [ ] Perlindungan SSRF saat koneksi dibuat
- [ ] Pengecekan masa berlaku sertifikat SSL
- [ ] Pengecekan TCP dan ping
- [ ] Dockerfile
