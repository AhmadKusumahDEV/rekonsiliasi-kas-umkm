# Backend Agent Guide

`init-project.md` adalah blueprint referensi dari project Mantra. Dokumen itu
menjadi target struktur, tetapi aturan project ini berbeda pada dependency dan
arti folder `pkg`:

## Identitas telemetry

- `service.name`: `rekonsiliasi-kas-api`
- `service.namespace`: `efaudit`
- `deployment.environment.name`: `dev`, `staging`, atau `prod`
- Package: `pkg/observability` (disalin dari template pusat)
- Endpoint same-host native: `http://127.0.0.1:4317` dengan `grpc`
- Endpoint remote: `https://otlp.madgo.my.id` dengan `http/protobuf`

Jangan menambahkan Grafana, Prometheus, Loki, Tempo, Collector, atau
Alertmanager ke Compose aplikasi. Konfigurasi ada di `otel.env.example`;
`otel.env` berisi nilai deployment dan diabaikan Git.

- Dependency memakai Go module biasa. **Jangan** memakai `vendor`, `-mod=vendor`,
  atau `go mod vendor`.
- `pkg/` berisi package milik project yang boleh diimpor lintas project, termasuk
  `pkg/database`. `internal/` berisi komponen yang spesifik aplikasi.
- Framework HTTP adalah Echo. Server, middleware, route, dan handler berada di
  `internal/echohttp`.
- SQL source berada di `scripts/postgres`. Hasil generated sqlc berada di
  `internal/repository/rekonsiliasi_repository_sqlc` dan tidak boleh diedit.

## Aturan kerja

1. Composition root hanya di `cmd/`. Jangan membuat DI container atau global
   pool di package domain.
2. Handler harus tipis. Tidak ada SQL, transaksi, atau business rule di
   `internal/echohttp`.
3. Business logic, authorization, mapping response, dan transaksi ada di
   `internal/service`.
4. Transaksi dibuka di service dengan `database.DB.BeginTx` dan query di dalamnya
   memakai `sqlc.New(tx)`. Jangan membuka transaksi di handler atau repository.
5. Jangan mengedit file generated sqlc. Ubah SQL/migration lalu jalankan
   `make gen.pg.models`.
6. Error baru harus punya sentinel di `pkg/errors/errors.go` dan mapping HTTP di
   `internal/echohttp/echohttp_error_handler.go`.
7. Selalu teruskan `c.Request().Context()` ke service, database, dan client
   eksternal. Jangan membuat `context.Background()` di tengah request.
8. Gunakan `pkg/utils.Jakarta()` atau `utils.NowJakarta()` untuk waktu bisnis.
9. Jangan menaruh secret di config yang di-commit. File lokal adalah
   `configs/local.yaml` dan sudah diabaikan Git.
10. `pkg/` tidak boleh mengimpor `internal/`; `internal/echohttp` tidak boleh
    mengimpor package sqlc; `internal/service` tidak boleh mengimpor Echo.
11. API domain baru memakai prefix `/api/v1`; session/CSRF ditangani middleware,
    dan handler mengambil identity dari session context.
12. Integration test memakai `internal/testkit` + `TEST_DATABASE_URL` yang
    menunjuk database test disposable. Jangan arahkan reset schema ke database
    produksi/development berisi data nyata.

## Alur menambah domain

1. Tambah migration up/down di `scripts/postgres/migrations`.
2. Tambah query beranotasi di `scripts/postgres/queries`.
3. Jalankan `make gen.pg.models` dan periksa signature generated.
4. Tambah model request/response di `internal/models`.
5. Implementasikan service dan transaksi di `internal/service`.
6. Tambah handler di `internal/echohttp` dan daftarkan route di
   `internal/echohttp/http_server.go`.
7. Tambah test dan fixture yang relevan.
8. Jalankan `go test ./...`, `go build ./...`, dan `make lint` bila lint
   terpasang. Jangan memakai flag vendor.

Jangan membuat fitur domain fiktif hanya untuk mengisi folder. Folder yang belum
punya kebutuhan nyata cukup berisi README sampai domain pertama ditambahkan.
