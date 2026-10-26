# Rekonsiliasi Kas UMKM — Backend

Scaffold backend Go dengan Echo, PostgreSQL (`pgx`), sqlc, Viper, dan struktur
layer yang mengikuti `init-project.md`. Project ini sengaja tidak memakai
`vendor`; dependency diambil lewat Go module seperti kebiasaan project
`go-chat`.

Status saat ini adalah kerangka yang bisa dijalankan. Migration baseline dari
prototipe PBL sudah tersedia, tetapi domain bisnis, endpoint authenticated, dan
query generated belum dibuat. Endpoint yang tersedia baru `/health/live` dan
`/health/ready`.

Asal-usul schema, tabel `pblg_*`, dan keputusan mengenai JSONB dijelaskan di
[`docs/DATABASE_FROM_PBL.md`](docs/DATABASE_FROM_PBL.md).

## Mulai

```powershell
cd backend
Copy-Item configs/local.yaml.example configs/local.yaml
go mod download
go run ./cmd/api
```

Buka `http://localhost:8080/health/live`. Database boleh tetap `enabled: false`
sampai domain pertama dibuat. Untuk menyalakan PostgreSQL lokal:

```powershell
docker compose up -d db
```

Lalu ubah `configs/local.yaml` menjadi `database.enabled: true` dan pastikan
DSN/field koneksi sesuai. Perintah penting:

```text
make run                 jalankan API
make build               build untuk OS aktif -> dist/<os>-<arch>/
make build-linux         build Linux amd64 untuk VPS -> dist/linux-amd64/
make test                go test ./...
make tidy                go mod tidy, tanpa vendor
make gen.pg.models       generate sqlc
make gen.pg.migration name=create_users
make pg.migrate.up DATABASE_URL=postgres://...
```

## Build untuk VPS

Build Linux dari mesin lokal (Windows/Linux) dengan:

```powershell
make build-linux
```

Hasilnya ada di `backend/dist/linux-amd64/rekonsiliasi-kas-umkm-api`. Folder
`dist/` sengaja diabaikan Git, tetapi [dist/README.md](dist/README.md) tetap
di-commit sebagai penanda lokasi artifact.

Di VPS, masuk ke root backend lalu jalankan binary. Posisi ini membuat aplikasi
menemukan `configs/local.yaml` secara default:

```bash
cd /path/ke/rekonsiliasi-kas-umkm/backend
make build-linux                 # jika Go dan make tersedia di VPS
chmod +x dist/linux-amd64/rekonsiliasi-kas-umkm-api
./dist/linux-amd64/rekonsiliasi-kas-umkm-api
```

Jika binary diunduh dari artifact GitHub Actions:

```bash
gh run download --name rekonsiliasi-kas-umkm-api-linux-amd64 -D dist/linux-amd64
cd dist/linux-amd64
chmod +x rekonsiliasi-kas-umkm-api
APP_CONFIG_FILE=../../configs/local.yaml ./rekonsiliasi-kas-umkm-api
```

Atau unduh artifact dari halaman Actions GitHub, ekstrak ke
`backend/dist/linux-amd64/`, lalu jalankan dengan `APP_CONFIG_FILE` seperti di
atas. Pastikan `configs/local.yaml` sudah dibuat dan `database.enabled` diatur
sesuai environment VPS.

## Struktur

```text
backend/
├── cmd/
│   ├── api/main.go                 # composition root HTTP
│   └── worker/                     # entrypoint worker saat dibutuhkan
├── internal/
│   ├── apiclient/                  # client layanan eksternal
│   ├── config/                     # config typed, key, dan header
│   ├── echohttp/                   # server, middleware, handler tipis
│   ├── models/                     # domain/DTO aplikasi
│   ├── repository/
│   │   └── rekonsiliasi_repository_sqlc/ # GENERATED sqlc
│   ├── service/                    # bisnis, authorization, transaksi
│   ├── testdata/                   # fixture integration test
│   └── worker/                     # consumer/job
├── pkg/
│   ├── database/                   # pool PostgreSQL yang diimpor service
│   ├── errors/                     # sentinel error + pesan user
│   ├── http/models/                # paging/DTO HTTP bersama
│   └── utils/                      # helper generik
├── scripts/postgres/
│   ├── migrations/                 # schema source
│   ├── queries/                    # query source sqlc
│   └── sqlc.yaml
├── configs/local.yaml.example
├── compose.yaml
├── Dockerfile
└── Makefile
```

## Aturan yang dipakai

- `cmd/` hanya untuk wiring dan bootstrap.
- Handler berada di `internal/echohttp`; handler membaca request, bind,
  validasi, mengambil identity, memanggil service, lalu menulis response.
- Business logic dan transaksi hanya di `internal/service`.
- `pkg/database` mengurus pool dan `BeginTx`, bukan query domain atau commit.
- SQL source ada di `scripts/postgres`; hasil generated ada di
  `internal/repository/rekonsiliasi_repository_sqlc` dan tidak diedit manual.
- `pkg/` berisi package yang boleh diimpor lintas project dan tidak boleh
  mengimpor `internal/`. Business logic spesifik aplikasi tetap di `internal/`.
- Tidak ada `go mod vendor` dan build tidak memakai `-mod=vendor`.

Detail keputusan dan alur dependency ada di
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Panduan query dan migration ada di
[`docs/SQLC_DATABASE_GUIDE.md`](docs/SQLC_DATABASE_GUIDE.md). Dokumen referensi
awal tetap disimpan sebagai [`init-project.md`](init-project.md).
