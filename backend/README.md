# Rekonsiliasi Kas UMKM — Backend

Scaffold backend Go dengan Echo, PostgreSQL (`pgx`), sqlc, Viper, dan struktur
layer yang mengikuti `init-project.md`. Project ini sengaja tidak memakai
`vendor`; dependency diambil lewat Go module seperti kebiasaan project
`go-chat`.

Status saat ini adalah kerangka yang bisa dijalankan dengan lapisan database dan
query yang sudah disiapkan. Migration baseline PBL, tabel UMKM ternormalisasi,
backfill dari JSONB, dan generated sqlc sudah tersedia. Service domain, endpoint
authenticated, serta handler bisnis belum diimplementasikan; endpoint HTTP yang
aktif baru `/health/live` dan `/health/ready`.

Asal-usul schema, tabel `pblg_*`, dan keputusan mengenai JSONB dijelaskan di
[`docs/DATABASE_FROM_PBL.md`](docs/DATABASE_FROM_PBL.md). Inventaris halaman,
endpoint, query, dan aturan bisnis ada di
[`docs/PBL_QUERY_INVENTORY.md`](docs/PBL_QUERY_INVENTORY.md).

## Schema dan query

- `000001_pbl_baseline` memindahkan enam tabel runtime prototipe PBL.
- `000002_umkm_normalized` menambahkan `umkm_imports`, `umkm_entries`, dan
  `umkm_allocations` beserta constraint/index.
- `000003_backfill_umkm_state` memindahkan data UMKM lama dari JSONB ke tabel
  ternormalisasi. Timestamp pembatalan/pembalikan lama tidak tersedia, sehingga
  backfill memakai waktu eksekusi untuk status tersebut.
- Query source ada di `scripts/postgres/queries/auth.sql` dan `umkm.sql`; hasil
  generate ada di `internal/repository/rekonsiliasi_repository_sqlc`.

Service auth/UMKM dan endpoint `/api/v1` untuk alur inti sudah memakai query
tersebut. Endpoint SSE events, worker, dan frontend belum ada.

## API v1

Kontrak lengkap untuk frontend, termasuk request/response dan catatan auth/CSRF,
ada di [`docs/API_ROUTES.md`](docs/API_ROUTES.md).

Endpoint publik:

```text
GET  /api/v1/config
POST /api/v1/auth/register
POST /api/v1/auth/login
GET  /health/live
GET  /health/ready
```

Endpoint dengan cookie sesi dan CSRF (`X-CSRF-Token` untuk non-GET):

```text
GET  /api/v1/auth/me
POST /api/v1/auth/logout
PUT  /api/v1/auth/password
GET  /api/v1/umkm/state
GET  /api/v1/umkm/suggestions
POST /api/v1/umkm/imports/preview
POST /api/v1/umkm/imports
POST /api/v1/umkm/imports/:id/cancel
POST /api/v1/umkm/allocations
POST /api/v1/umkm/allocations/:id/undo
GET  /api/v1/umkm/export
```

## Testing

Unit test tidak membutuhkan database:

```powershell
go test ./...
```

Integration test memakai `TEST_DATABASE_URL`; test akan reset schema `public`,
jadi DSN wajib menunjuk database khusus test:

```powershell
docker compose up -d db-test
$env:TEST_DATABASE_URL="postgres://postgres:postgres@127.0.0.1:55432/rku_test?sslmode=disable"
go test ./internal/echohttp -run TestUMKMIntegrationFlow -count=1 -v
```

Alternatif WSL:

```powershell
wsl -d Ubuntu -- bash -lc 'cd /mnt/d/appwebsite/rekonsiliasi-kas-umkm/backend && TEST_DATABASE_URL=postgres://postgres:postgres@127.0.0.1:55432/rku_test?sslmode=disable go test ./internal/echohttp -run TestUMKMIntegrationFlow -count=1 -v'
```

Pola dan batasan test yang dipelajari dari project `go-chat` dicatat di
[`docs/TESTING_FROM_GO_CHAT.md`](docs/TESTING_FROM_GO_CHAT.md). Test integration
akan di-skip tanpa `TEST_DATABASE_URL`; set `TEST_REQUIRE_DB=1` bila database
harus tersedia dan ketiadaannya harus menggagalkan test.

## Debug di VS Code

Folder ini memiliki `.vscode/launch.json` dan `.vscode/tasks.json`; root repo
juga memiliki konfigurasi yang sama dengan path disesuaikan. Buka folder
`backend` atau root repo di VS Code, lalu pilih salah satu konfigurasi:

- `API + PostgreSQL (full env)` — menyalakan `db` lewat Docker Compose dan
  menjalankan API dengan seluruh env development.
- `API health only (database off)` — menjalankan API tanpa database.
- `Integration Test: UMKM flow` — menyalakan `db-test` dan menjalankan flow HTTP
  end-to-end dengan PostgreSQL test.
- `Unit Test: service domain` dan `Unit Test: HTTP validation` — debug unit test
  tanpa database.

Env development dan test sudah tertulis di launch config. Kredensial di sana
hanya untuk Docker lokal, jangan dipakai untuk production.

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

Compose memakai port host `5433` untuk database development agar tidak bentrok
dengan PostgreSQL lokal yang biasanya memakai `5432`.

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
├── docs/
│   ├── DATABASE_FROM_PBL.md
│   └── PBL_QUERY_INVENTORY.md
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
