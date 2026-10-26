# Mantra Backend — Agent Scaffolding Blueprint

> Dokumen ini adalah kontrak arsitektur untuk agent yang akan membuat project baru atau menambah domain di backend ini. Isinya disusun dari struktur dan pola nyata repo **mantra-backend** per 26 Oktober 2026.
>
> Nama file sengaja **AGENTS.md** agar otomatis dibaca agent ketika berada di root repo. Untuk project baru, salin file ini ke root project dan ikuti bagian 13. Untuk fitur baru, ikuti bagian 14–15.
>
> Aturan di bawah berlaku untuk **kode baru dan scaffolding**. Jangan melakukan refactor besar pada kode lama yang tidak berhubungan dengan tugas, kecuali diminta.

## 0. Aturan emas

1. **Composition root hanya di cmd/.** Semua pembuatan config, database pool, client eksternal, dan service dilakukan di entrypoint. Tidak ada DI container.
2. **Handler HTTP harus tipis.** Handler hanya membaca request, bind, validasi bentuk, mengambil identity dari context, memanggil service, lalu menulis response. Tidak ada SQL dan tidak ada transaksi di handler.
3. **Business logic ada di service.** Authorization, validasi aturan bisnis, perhitungan, pemanggilan client eksternal, mapping data, dan transaksi semuanya di internal/service.
4. **Transaksi hanya dibuka di service.** Repository sqlc adalah generated code dan tidak boleh mengelola transaksi. Gunakan **pgxpool.BeginTx** di method service, lalu **sqlc.New(tx)** untuk semua query dalam unit of work tersebut.
5. **Folder internal/repository/mantra_repository_sqlc adalah generated.** Jangan mengedit db.go, models.go, atau file *.sql.go secara manual. Ubah SQL di scripts/postgres/queries, lalu regenerate.
6. **SQL source of truth ada di scripts/postgres.** Migration di scripts/postgres/migrations, query di scripts/postgres/queries, konfigurasi sqlc di scripts/postgres/sqlc.yaml.
7. **Error memakai sentinel dan central mapping.** Service mengembalikan error dari pkg/errors (dibungkus bila perlu). Handler cukup return error. Pemetaan HTTP status ada di internal/echohttp/echohttp_error_handler.go.
8. **Context selalu diteruskan.** Gunakan c.Request().Context() dari handler dan teruskan ke service, sqlc, serta client eksternal. Jangan membuat context.Background() di tengah request.
9. **Waktu bisnis memakai Asia/Jakarta.** Gunakan internal/config.LocationJakarta atau helper waktu yang sudah ada. Jangan mengandalkan timezone lokal server.
10. **Generated code ikut di-commit.** Setelah make gen.pg.models, file generated yang berubah harus ikut masuk commit agar build tidak bergantung pada sqlc saat deploy.

## 1. Stack dan pilihan teknologi

| Komponen | Pilihan di repo | Fungsi |
|---|---|---|
| Bahasa | Go 1.24.2 | Bahasa utama |
| HTTP | Echo v4.13.3 | Routing, middleware, binding |
| Database driver | pgx/v5 v5.7.1 + pgxpool | PostgreSQL, pool, transaksi |
| Query generator | sqlc v1.31.1, config v2, sql_package pgx/v5 | Generate type-safe query |
| Migration | golang-migrate | Migration up/down |
| Validasi | go-playground/validator v10 | Validasi request |
| Config | Viper + toolkit springcloud | Config remote/local |
| Observability | toolkit tracer + OpenTelemetry | Span handler/service |
| Storage | Google Cloud Storage | File dan lampiran |
| Cache | go-redis/v8 | Scaffold repository, belum aktif |
| Test DB | testcontainers-go + go-testfixtures | Integration test |
| Build/deploy | Docker multi-stage, distroless | Image API |

Gunakan **-mod=vendor** pada build dan run karena folder vendor di-commit. Jangan menambah dependency tanpa menjalankan go mod tidy dan go mod vendor.

### Mengapa struktur ini siap di-scale

- **API stateless.** Identity datang dari JWT dan session, bukan state di memori proses. Beberapa replica API bisa berjalan di belakang load balancer selama session store dipakai bersama.
- **Pool per instance, bukan global state.** Setiap replica memiliki pgxpool sendiri; ukuran pool dan koneksi database menjadi parameter deployment.
- **Layer terpisah.** Perubahan transport, aturan bisnis, dan SQL tidak saling menembus. Domain baru ditambahkan sebagai file baru tanpa mengubah handler domain lain.
- **sqlc dan migration versioned.** Query memiliki tipe hasil generate dan schema berubah lewat migration yang bisa direview. Ini mengurangi risiko drift antara kode dan database.
- **Config terpusat.** Config dan secret berasal dari Spring Cloud/Secret Manager, sehingga image yang sama bisa dipakai di beberapa environment.
- **Beban background dipisah.** Job dan consumer punya entrypoint sendiri, sehingga proses panjang tidak menahan request API.
- **Observability dan shutdown sudah disiapkan.** Span handler/service, logging middleware, health check, metrics, dan graceful shutdown tersedia lewat toolkit.
- **Titik ekstensi jelas.** Redis, worker, dan client eksternal punya folder sendiri; mengaktifkannya tidak merusak struktur inti.

Jika jumlah domain tumbuh sangat besar, satu struct Service masih bisa dipecah menjadi service per domain dengan constructor terpisah. Pertahankan batas layer yang sama: handler tetap memanggil service, dan transaksi tetap di service.

## 2. Struktur folder final

~~~text
.
├── cmd/
│   ├── api/
│   │   └── main.go                 # composition root HTTP API
│   ├── job1/
│   │   └── main.go                 # one-shot job (saat ini stub)
│   └── pubsub-worker/
│       └── main.go                 # consumer pub/sub (saat ini stub)
├── internal/
│   ├── apiclient/
│   │   ├── bnm/                    # client eksternal BNM
│   │   │   ├── bnm_client.go
│   │   │   └── bnm_models.go
│   │   └── thirdpartytwo/          # contoh client eksternal lain
│   ├── config/
│   │   ├── config.go               # helper timezone dan config umum
│   │   ├── config_keys.go          # konstanta key config
│   │   └── header_keys.go          # nama header HTTP
│   ├── echohttp/
│   │   ├── http_server.go          # setup Echo, middleware, route
│   │   ├── <domain>_handler.go     # handler per domain
│   │   ├── middleware_handler.go   # auth JWT + session
│   │   ├── logging_middleware.go
│   │   ├── security_middleware.go
│   │   ├── security_config.go
│   │   └── echohttp_error_handler.go
│   ├── models/
│   │   ├── <domain>.go             # request/response/domain model
│   │   ├── kode_status_model.go    # konstanta status
│   │   └── feature_keys.go
│   ├── repository/
│   │   ├── mantra_repository_sqlc/ # GENERATED, jangan diedit
│   │   │   ├── db.go
│   │   │   ├── models.go
│   │   │   └── *.sql.go
│   │   └── redis/
│   │       ├── redis_repository.go
│   │       └── <entity>_redis_repository.go
│   ├── service/
│   │   ├── service.go              # struct Service + constructor
│   │   ├── <domain>_service.go     # business logic per domain
│   │   ├── helper.go               # helper lintas domain
│   │   ├── service_errors.go
│   │   └── service_test_utils.go   # fake client untuk test
│   ├── testdata/                   # fixture YAML
│   └── worker/
│       └── <consumer>_worker.go    # satu file per consumer
├── pkg/
│   ├── errors/
│   │   └── errors.go               # sentinel error + pesan user
│   ├── http/
│   │   ├── models/                 # DTO HTTP publik, paging
│   │   └── service_http_client.go
│   ├── pb/                         # protobuf legacy
│   └── utils/                      # helper generik
├── scripts/
│   └── postgres/
│       ├── migrations/             # 000001_<nama>.up.sql / .down.sql
│       ├── queries/                # <domain>.sql atau <domain>.<op>.sql
│       └── sqlc.yaml
├── docs/
├── vendor/                         # dependency vendored
├── AGENTS.md                       # dokumen ini
├── Dockerfile
├── Makefile
├── config.yml.example
├── go.mod
├── go.sum
└── ...
~~~

Folder **internal/** tidak bisa diimpor project lain. Folder **pkg/** boleh dipakai lintas project, jadi jangan menaruh business logic spesifik Mantra di sana.

### Peta domain yang sudah ada

| Domain | Handler | Service | Query utama |
|---|---|---|---|
| Auth dan pekerja | pekerja_handler.go | pekerja_service.go | pekerja.*.sql |
| Home, fitur, app version | home_handler.go | home_service.go | features.sql |
| Survey, agenda, mitra, SPK | mitra_handler.go | agenda_mitra_service.go, mitra_service.go, spk.go | mitra.sql, spk.sql |
| Pekerjaan dan absensi | pekerjaan_handler.go | pekerjaan_get_service.go, pekerjaan_service.go, pekerjaan_absensi_service.go | pekerjaan.get.sql, pekerjaan.insert.sql, pekerjaan.update.sql |
| Serah terima | serah_terima.go | serah_terima_service.go | serah_terima.get.sql, serah_terima.insert.sql, serah_terima.update.sql |

Beberapa handler lama menggabungkan lebih dari satu domain. Untuk domain baru, tetap gunakan satu file handler per domain agar file tidak semakin besar.

## 3. Arah dependency

~~~text
cmd/api
  └──> internal/echohttp ──> internal/service ──> internal/repository/mantra_repository_sqlc ──> PostgreSQL
                                 ├──> internal/apiclient/*
                                 ├──> internal/models
                                 ├──> internal/config
                                 └──> pkg/utils, pkg/errors

internal/echohttp ──> internal/models, pkg/http/models, pkg/errors
pkg/*             ──> stdlib dan third-party (tidak import internal)
~~~

Aturan import:

- cmd/api boleh mengimpor echohttp, service, config, apiclient, utils, dan toolkit.
- echohttp boleh mengimpor service, models, pkg/http/models, pkg/errors, config. **Tidak boleh mengimpor sqlc.**
- service boleh mengimpor sqlc, models, config, apiclient, pkg/errors, pkg/utils. **Tidak boleh mengimpor echo.**
- repository/sqlc hanya generated; tidak mengimpor service atau echohttp.
- models tidak mengimpor echohttp atau sqlc. Model database tetap di package hasil generate.
- pkg tidak mengimpor internal.

Jika aturan ini dilanggar, dependency cycle akan muncul atau layer menjadi tidak bisa dites.

## 4. Bootstrap dan wiring

Entrypoint API mengikuti urutan ini:

1. Buat signal-aware context dengan toolkit runtimekit.
2. Buat HTTP client standar dan load Spring Cloud config.
3. Jika bukan environment STL, ambil secret dari Secret Manager dan tulis config lokal.
4. Aktifkan OpenTelemetry jika feature tracing aktif.
5. Buat pgxpool dengan toolkit postgres dan key db.mantra.
6. Buat client GCS dan client BNM.
7. Buat satu instance service.NewService(...).
8. Jalankan echohttp.RunEchoHTTPService(...).
9. Tutup resource saat shutdown.

Kerangka main.go:

~~~go
func main() {
    var err error

    appContext, cancel := runtimekit.NewRuntimeContext()
    defer func() {
        cancel()
        if err != nil {
            log.FromCtx(appContext).Error(err, "found error")
        }
    }()

    httpClient := httpclient.NewStdHTTPClient()
    appConfig := springcloud.NewRemoteConfig(httpClient)
    if err = appConfig.Load(appContext); err != nil {
        return
    }

    dbpool, err := postgres.NewFromConfigWithPGX(appConfig, "db.mantra", nil)
    if err != nil {
        return
    }
    defer dbpool.Close()

    gcsClient, err := utils.AuthenticateGCS(appContext, appConfig)
    if err != nil {
        return
    }

    bnmClient := bnm.NewHTTPClient(httpClient, appConfig)

    svc := service.NewService(httpClient, appConfig, dbpool, gcsClient, bnmClient)

    echohttp.RunEchoHTTPService(
        appContext,
        svc,
        appConfig,
        dbpool,
        tracer.GetOtelTracer(),
    )
}
~~~

Catatan penting:

- service.NewService adalah satu-satunya tempat dependency service dirakit.
- Struct Service menyimpan dependency sebagai field private. Tambah dependency baru berarti menambah parameter constructor dan field, lalu mengubah pemanggil di cmd/api.
- RunEchoHTTPService saat ini menerima dbpool dan tracer tetapi mengabaikannya. Service sudah memegang pool. Jangan membuat akses database kedua dari handler.
- Versi aplikasi diisi lewat -ldflags ke service.Version, bukan hardcode.
- Jangan membuat package bootstrap baru untuk API selama satu entrypoint masih cukup. Jika worker nyata ditambahkan, bootstrap bersama boleh diekstrak agar cmd/api dan cmd/worker tidak menduplikasi urutan wiring.

## 5. Layer HTTP

### Route

Semua route didaftarkan di satu tempat: internal/echohttp/http_server.go, di dalam RunEchoHTTPService. Tidak ada file router terpisah dan tidak ada controller struct.

~~~go
e := echo.New()
e.HTTPErrorHandler = handleEchoError(cfg)
e.Validator = &CustomValidator{Validator: validator.New()}

e.Use(SecurityHeadersMiddlewareWithConfig(cfg))
e.Use(CORSSecureMiddlewareWithConfig(cfg))
e.Use(APISecurityMiddleware())
e.Use(LoggingMiddleware)

v1 := e.Group("/v1", workerMiddlewareHandler(cfg, s))
v1.GET("/home", home(s))
v1.GET("/widget", listWidgets(s))
v1.POST("/widget", createWidget(s))
~~~

Urutan middleware global: security headers → CORS → API security → logging → middleware runtime echokit (validator translator, request timeout, prometheus, health). Auth JWT dipasang pada group /v1, bukan global, agar endpoint login/refresh/app-version tetap terbuka.

### Handler

Handler adalah factory yang mengembalikan echo.HandlerFunc:

~~~go
func createWidget(s *service.Service) echo.HandlerFunc {
    return func(c echo.Context) error {
        ctx, span := tracer.NewSpan(c.Request().Context(), tracer.SpanLvlHandler)
        defer span.End()

        workerInfo := c.Get(intCfg.KeyWorkerInfoJWTContextKey).(*models.WorkerInfo)

        var req models.CreateWidgetRequest
        if err := c.Bind(&req); err != nil {
            return err
        }
        if err := c.Validate(req); err != nil {
            return err
        }

        resp, err := s.CreateWidget(ctx, workerInfo, req)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusCreated, resp)
    }
}
~~~

Konvensi handler:

- Nama fungsi lower camel-case, file snake_case, satu file per domain.
- Bind JSON atau multipart dengan c.Bind, lalu c.Validate untuk aturan bentuk.
- Identity diambil dari context: worker_info, timestamp, signature.
- Jangan menulis error response manual. Return error dan biarkan central handler memetakan.
- Status sukses: 200 untuk read, 201 untuk create, 202 untuk accepted, 204 untuk no content. Gunakan status yang konsisten dengan endpoint yang sudah ada.
- Response sukses biasanya langsung DTO service, bukan envelope buatan handler. Pesan sederhana boleh memakai map[string]string.
- Pagination memakai pkg/http/models.CursorPaging dengan query next_cursor dan limit. Default limit 5. Response menyertakan meta_data.

### Context keys

Key ada di internal/config/config_keys.go:

- KeyWorkerInfoJWTContextKey berisi *models.WorkerInfo.
- KeyTimestampContextKey dan KeySignatureContextKey berisi string dari header X-Timestamp dan X-Signature.
- Middleware workerMiddlewareHandler memvalidasi JWT, mengecek session lewat service, lalu mengisi ketiga key tersebut.
- Role check tetap dilakukan di service (IsMandor atau IsTukang), bukan hanya di handler.

### Error mapping

Service mengembalikan sentinel error. Handler return error apa adanya. Fungsi mappingMsgError di echohttp_error_handler.go menentukan status dan pesan.

Saat menambah error baru:

1. Tambah sentinel di pkg/errors/errors.go.
2. Tambah pesan user-facing bila perlu.
3. Tambah case errors.Is di mappingMsgError.
4. Pastikan default 500 hanya untuk error yang memang tidak dikenal.

Jangan mengembalikan error mentah database ke client. Bungkus dengan errors.Wrap untuk konteks log, lalu map ke sentinel yang sesuai.

## 6. Layer service

Service adalah aggregate object tunggal. Semua domain adalah method pada *Service, bukan struct service terpisah.

~~~go
type Service struct {
    httpClient *http.Client
    cfg        config.KVStore
    db         *pgxpool.Pool
    gcsClient  utils.IBMTGCS
    bnmClient  bnm.BMTClient
}

func NewService(
    httpClient *http.Client,
    cfg config.KVStore,
    db *pgxpool.Pool,
    gcsClient utils.IBMTGCS,
    bnmClient bnm.BMTClient,
) *Service {
    return &Service{
        httpClient: httpClient,
        cfg:        cfg,
        db:         db,
        gcsClient:  gcsClient,
        bnmClient:  bnmClient,
    }
}
~~~

Struktur method:

~~~go
func (s *Service) CreateWidget(
    ctx context.Context,
    workerInfo *models.WorkerInfo,
    req models.CreateWidgetRequest,
) (models.Widget, error) {
    ctx, span := tracer.ServiceFuncSpan(ctx)
    defer span.End()

    if !workerInfo.IsMandor() {
        return models.Widget{}, pkgErr.ErrInvalidAccessRole
    }

    if workerInfo.KodeMitra == "" {
        return models.Widget{}, pkgErr.ErrInvalidRequest
    }

    // baca data pendukung, hitung aturan bisnis, lalu tulis lewat transaksi
}
~~~

Pembagian tanggung jawab:

- **Handler:** transport, bind, validasi bentuk, identity, status code.
- **Service:** authorization, aturan bisnis, perhitungan, orkestrasi, transaksi, mapping response.
- **sqlc:** query dan row type. Jangan menaruh aturan bisnis di SQL bila aturan itu lebih jelas dan bisa dites di Go.
- **models:** DTO dan domain type. Model generated sqlc tidak dipakai langsung sebagai response HTTP.

Konvensi file:

- Satu file per domain: [domain]_service.go.
- Helper privat yang masih satu domain tetap di file yang sama.
- Helper yang dipakai beberapa domain masuk helper.go.
- Method publik diberi komentar bila perlu; method privat tidak perlu diekspor.
- Gunakan tracer.ServiceFuncSpan(ctx) dan defer span.End().

## 7. Transaksi — aturan utama project ini

Ini bagian yang paling penting. Transaksi **tidak** dibuka di handler dan **tidak** dibuka di repository. Transaksi dibuka di method service yang menjadi satu unit of work.

Repo ini memakai pgxpool dan pgx.Tx, bukan database/sql.DB. Import database/sql hanya untuk tipe nullable seperti sql.NullString. Jangan mencampur dua transaction manager dalam satu unit of work.

Pola yang benar:

~~~go
func (s *Service) CreateWidget(
    ctx context.Context,
    workerInfo *models.WorkerInfo,
    req models.CreateWidgetRequest,
) (resp models.Widget, err error) {
    ctx, span := tracer.ServiceFuncSpan(ctx)
    defer span.End()

    tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return resp, pkgErr.ErrFailInitializeTx
    }

    defer func() {
        if err != nil {
            _ = tx.Rollback(ctx)
        }
    }()

    q := sqlc.New(tx)

    row, err := q.CreateWidget(ctx, sqlc.CreateWidgetParams{
        Name:      req.Name,
        KodeMitra: workerInfo.KodeMitra,
    })
    if err != nil {
        return resp, errors.Wrap(err, "create widget")
    }

    if err = q.InsertWidgetLog(ctx, sqlc.InsertWidgetLogParams{
        WidgetID: row.ID,
        Action:   "created",
    }); err != nil {
        return resp, errors.Wrap(err, "insert widget log")
    }

    if err = tx.Commit(ctx); err != nil {
        return resp, pkgErr.ErrFailCommitTx
    }

    return models.Widget{
        ID:        row.ID,
        Name:      row.Name,
        KodeMitra: row.KodeMitra,
        CreatedAt: row.CreatedAt,
    }, nil
}
~~~

Mengapa memakai named return **err**? Deferred rollback membaca variable err. Jika return value tidak dinamai, closure defer bisa melihat err lama dari BeginTx (nil) dan melewatkan rollback saat query gagal. Dengan signature (resp T, err error), setiap return error akan mengisi err sebelum defer berjalan.

Aturan transaksi:

1. **BeginTx hanya di service.** Cari dengan rg "BeginTx" dan pastikan tidak ada di echohttp atau repository.
2. **Semua query tulis memakai q := sqlc.New(tx).** Jangan memakai sqlc.New(s.db) di dalam blok transaksi.
3. **Helper yang ikut transaksi menerima *sqlc.Queries.** Contoh: func (s *Service) insertLog(ctx context.Context, q *sqlc.Queries, id string) error. Jangan membuat helper yang mengambil pool sendiri.
4. **Commit satu kali di akhir.** Jika commit gagal, kembalikan sentinel commit, jangan menganggap sukses.
5. **Rollback lewat defer.** Rollback setelah commit yang sukses akan menghasilkan error tx closed; abaikan dengan _ = tx.Rollback(ctx).
6. **Jangan menahan transaksi saat memanggil HTTP/GCS.** Lakukan panggilan eksternal sebelum BeginTx atau setelah Commit. Jika hasil eksternal harus ikut atomik, gunakan pola outbox/status, bukan transaksi database yang menunggu network.
7. **Jangan nested transaction.** Saat ini tidak ada savepoint. Jika butuh beberapa langkah, gunakan satu transaksi dan urutan query yang jelas. Jika benar-benar butuh partial rollback, tambahkan savepoint secara eksplisit.
8. **Untuk read-only yang butuh konsistensi,** transaksi boleh dipakai. Untuk list sederhana, cukup sqlc.New(s.db).
9. **Concurrency:** gunakan SELECT ... FOR UPDATE di query untuk row yang diperebutkan, lalu tetap lakukan pengecekan status di service.
10. **Context cancellation:** ctx dari request harus dipakai di BeginTx, query, dan Commit. Jangan memakai context.Background().

Isolation default pgx.TxOptions{} adalah read committed. Gunakan opsi lain hanya jika ada alasan konkret dan dokumentasikan di komentar.

## 8. Data layer sqlc

### Konfigurasi

File: scripts/postgres/sqlc.yaml.

Panduan implementasi langkah demi langkah, termasuk package database internal dan cara membundle CLI sqlc: docs/SQLC_DATABASE_GUIDE.md.

~~~yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "queries"
    schema: "migrations"
    gen:
      go:
        package: "sqlc"
        sql_package: "pgx/v5"
        out: "../../internal/repository/mantra_repository_sqlc"
        emit_json_tags: true
        emit_prepared_queries: false
        emit_interface: false
        emit_exact_table_names: false
~~~

Konsekuensi konfigurasi:

- Generated package bernama sqlc dan diimpor dengan alias sqlc.
- Schema diambil dari folder migrations. Migration harus valid dan berurutan.
- emit_interface false, jadi tidak ada interface Querier per domain. Service memakai *sqlc.Queries langsung.
- Generated code menyediakan New(db DBTX) dan WithTx(tx pgx.Tx). Repo ini memakai sqlc.New(tx) langsung; WithTx tersedia tetapi bukan konvensi yang dipakai.
- Tipe nullable dipetakan ke sql.NullString, sql.NullTime, sql.NullBool, sql.NullInt32, sql.NullInt64, dan pgtype.Json. Kolom citext dipetakan ke string.

### Migration

Gunakan golang-migrate dengan pasangan file:

~~~text
scripts/postgres/migrations/000012_add_widget.up.sql
scripts/postgres/migrations/000012_add_widget.down.sql
~~~

Aturan:

- Nomor enam digit, berurutan, jangan mengubah migration yang sudah pernah dipakai environment lain.
- Selalu buat down migration yang benar-benar menghapus objek yang dibuat.
- Satu migration untuk satu perubahan logis.
- Tambahkan index untuk kolom yang dipakai filter/join.
- Gunakan varchar dengan prefix ID bila mengikuti gaya repo, atau bigserial bila domain baru tidak butuh kode bisnis.
- Jangan memakai SELECT * di query aplikasi; tulis kolom eksplisit agar generated type stabil.

Perintah:

~~~bash
make gen.pg.migration name=add_widget
make pg.migrate.up PG_DB_URL='postgres://user:pass@host:5432/db?sslmode=disable'
make pg.migrate.down PG_DB_URL='postgres://user:pass@host:5432/db?sslmode=disable'
~~~

### Query

Satu file per domain, boleh dipecah per operasi:

~~~text
scripts/postgres/queries/widget.sql
scripts/postgres/queries/widget.get.sql
scripts/postgres/queries/widget.insert.sql
scripts/postgres/queries/widget.update.sql
~~~

Setiap statement diawali komentar anotasi sqlc:

~~~sql
-- name: CreateWidget :one
INSERT INTO public.widget (name, kode_mitra)
VALUES (@name, @kode_mitra)
RETURNING id, name, kode_mitra, created_at;

-- name: ListWidgets :many
SELECT id, name, kode_mitra, created_at
FROM public.widget
WHERE kode_mitra = @kode_mitra
ORDER BY created_at DESC
LIMIT @limit_data OFFSET @offset_data;
~~~

Anotasi yang dipakai:

| Anotasi | Hasil |
|---|---|
| :one | Satu row; return row type atau error pgx.ErrNoRows |
| :many | Slice row |
| :exec | Hanya error |
| :execrows | Jumlah row terdampak (belum dipakai di repo ini) |
| :batchexec | Batch insert (belum dipakai di repo ini) |

Konvensi parameter:

- Gunakan named parameter @snake_case, bukan ? atau $1.
- Untuk array, pertahankan cast PostgreSQL: @kode_status::int[].
- Untuk parameter opsional, gunakan sqlc.narg (belum dipakai di repo ini) atau dua query terpisah agar tipe tetap jelas.
- Untuk parameter yang bisa NULL, generated field menjadi sql.Null* atau pgtype.* dan harus diisi Valid: true saat dipakai.

### Generate

~~~bash
make gen.pg.models
~~~

Perintah di balik target:

~~~bash
cd scripts/postgres && sqlc generate -f sqlc.yaml
~~~

Target membutuhkan sqlc dan migrate terpasang. Setelah generate:

1. Periksa signature generated: [QueryName]Params, [QueryName]Row, atau struct tabel.
2. Sesuaikan pemanggilan service dengan tipe nullable yang dihasilkan.
3. Commit file generated yang berubah.
4. Jangan mengedit file generated untuk "memperbaiki" tipe. Perbaiki SQL atau override di sqlc.yaml.

Jika ada file generated lama tanpa sumber query, jangan dijadikan contoh. Regenerate dan hapus file yatim setelah dipastikan tidak dipakai.

## 9. Model, error, dan config

### Model

- internal/models: domain type, request/response payload, status constants, helper kecil.
- pkg/http/models: DTO HTTP publik dan shared paging seperti CursorPaging.
- internal/repository/mantra_repository_sqlc: generated row type. Jangan dipakai langsung sebagai response handler.
- Mapping dari generated row ke model dilakukan di service. Ini menjaga kontrak HTTP stabil saat schema berubah.

Contoh model:

~~~go
type CreateWidgetRequest struct {
    Name string `json:"name" validate:"required,min=3,max=120"`
}

type Widget struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    KodeMitra string    `json:"kode_mitra"`
    CreatedAt time.Time `json:"created_at"`
}
~~~

Status bisnis ditaruh di internal/models/kode_status_model.go sebagai konstanta bernomor, misalnya status survey 11–14, SPK 51–53, pekerjaan 61–64, serah terima 71–77, dan status pekerjaan tukang 81–84. Saat menulis ke kolom int2, konversi ke pgtype.Int2 dengan Valid: true.

### Error

- Sentinel error di pkg/errors/errors.go.
- Bungkus error teknis dengan errors.Wrap/Wrapf dari github.com/pkg/errors.
- Jangan menelan error. Jika ingin mengabaikan, tulis _ = ... secara eksplisit.
- Pesan error user-facing berbahasa Indonesia ada di pkg/errors; jangan bocorkan detail SQL.
- Setiap sentinel baru harus dipetakan di mappingMsgError agar status HTTP-nya benar.

Contoh:

~~~go
var ErrWidgetNotFound = errors.New("widget tidak ditemukan")
~~~

Lalu di echohttp_error_handler.go:

~~~go
case errors.Is(err, pkgErr.ErrWidgetNotFound):
    statusCode = http.StatusNotFound
    message = pkgErr.ErrWidgetNotFound.Error()
~~~

### Config

- Key string hanya di internal/config/config_keys.go.
- Nilai contoh hanya di config.yml.example; jangan commit config.yml berisi secret.
- Akses nilai lewat interface config.KVStore: GetString, GetBool, GetInt, GetDuration.
- Config production memakai Spring Cloud dan Secret Manager. Local memakai SPRING_CLOUD_CONFIG_URL=file://config.yml dan SPRING_CLOUD_CONFIG_PATHS=myapp/local.
- Jika menambah key: tambah konstanta, tambah contoh YAML, tambah default bila perlu, lalu baca lewat konstanta tersebut. Jangan menulis string key langsung di service.

Contoh:

~~~go
const KeyWidgetMaxPerPage = "app.widget.max-per-page"
~~~

~~~yaml
app:
  widget:
    max-per-page: 50
~~~

## 10. Client eksternal, Redis, dan worker

### Client eksternal

- Satu folder per service di internal/apiclient/[service].
- Client struct menyimpan *http.Client dan config yang dibutuhkan.
- File model request/response terpisah dari implementasi.
- Method dengan JSON response: ([response], *http.Response, error). Tanpa JSON: (*http.Response, error).
- Client tidak boleh mengakses database.
- Untuk test, buat fake di service_test_utils.go atau interface kecil di package client.

### Redis

- Repository ada di internal/repository/redis, satu file per entity.
- Struct Repository menyimpan client dan config.KVStore.
- Key prefix dan TTL adalah config, bukan hardcode.
- Di repo ini Redis belum di-wire ke cmd/api dan belum ada blok config Redis. Jika fitur membutuhkan Redis:
  1. Tambah key config dan blok config.yml.example.
  2. Buat client di cmd/api/main.go.
  3. Tambah field ke service.Service dan parameter NewService.
  4. Gunakan Redis untuk cache/session, bukan source of truth data bisnis.
  5. Tangani cache miss dan error Redis sebagai fallback, jangan menggagalkan request read hanya karena cache error kecuali memang dirancang demikian.

### Worker

- cmd/job1 untuk one-shot job.
- cmd/pubsub-worker untuk consumer.
- internal/worker satu file per consumer.
- Saat ini ketiganya masih placeholder. Jangan menganggap pubsub sudah berjalan.
- Saat mengaktifkan worker, buat entrypoint yang benar: load config, buat pool, buat service, jalankan consumer, tangani SIGTERM, dan tunggu in-flight message selesai.
- Consumer harus idempotent karena pub/sub bisa mengirim pesan lebih dari sekali.
- Jangan memakai pool yang berbeda dari service tanpa alasan. Worker dan API boleh berbagi constructor, tetapi deployment-nya terpisah.
- Jika wiring mulai duplikatif, ekstrak internal/bootstrap dengan fungsi BuildApp yang dipakai cmd/api dan cmd/worker.

## 11. Testing

Test utama adalah integration test service dengan PostgreSQL container. Bukan unit test yang mem-mock sqlc.

- TestMain membuat container PostgreSQL 16-3.5-alpine.
- Semua file *.up.sql dari scripts/postgres/migrations dijalankan.
- Fixture YAML di internal/testdata dimuat dengan go-testfixtures.
- Service dibuat dengan pool nyata; GCS dan BNM memakai fake.
- Package test adalah service_test (external test package), sehingga kontrak publik service ikut teruji.

Contoh:

~~~go
func TestCreateWidget(t *testing.T) {
    ctx := context.Background()

    svcProxy, teardown := NewServiceProxy(ctx, t)
    defer teardown()

    if ok := pgtestutil.AssertLoadPgxDBFixtures(
        t,
        svcProxy.bmtPgx,
        testfixtures.FilesMultiTables("../testdata/widget.yaml"),
    ); !ok {
        return
    }

    workerInfo := &models.WorkerInfo{
        KodeMitra: "MTR_0001",
        Jabatan:   models.RoleNameMandor,
    }

    resp, err := svcProxy.svc.CreateWidget(ctx, workerInfo, models.CreateWidgetRequest{
        Name: "AC Split",
    })

    assert.NoError(t, err)
    assert.NotEmpty(t, resp.ID)
}
~~~

Aturan test:

- Satu test = satu skenario bisnis yang jelas.
- Setiap test memuat fixture yang dibutuhkan; jangan bergantung pada sisa data test lain.
- Jika terkena error sequence log agenda, panggil query.ResetLogAgendaSequence(ctx) seperti contoh di TestAutoCancelAgenda.
- Fake GCS/BNM ditaruh di service_test_utils.go agar tidak tersebar.
- Test membutuhkan Docker. Di Windows, jalankan lewat WSL/Git Bash atau Docker Desktop yang aktif.
- Jalankan go test ./internal/service -run TestNama -v untuk iterasi cepat, lalu go test ./... sebelum selesai.

## 12. Tooling dan deployment

| Target | Fungsi |
|---|---|
| make run-service-local | Jalankan API dengan config.yml lokal |
| make gen.pg.models | Generate sqlc |
| make gen.pg.migration name=... | Buat pasangan migration baru |
| make pg.migrate.up / down | Jalankan migration |
| make test | Test semua package dan buat coverage HTML |
| make lint | go fmt + golangci-lint --fix |
| make fmt | go fmt semua package |
| make deps | go get -u, tidy, vendor |

Build API yang benar untuk module ini:

~~~bash
go build -ldflags "-X git.alfamikro.com/back-office/mantra-backend/internal/service.Version=$(git rev-parse --short HEAD)" -mod=vendor -o bin/app cmd/api/main.go
~~~

Deployment:

- Dockerfile multi-stage: golang:1.24-alpine lalu gcr.io/distroless/base.
- Image API menyalin config.yml dan service account GCS. Keduanya tidak boleh berisi secret yang di-commit.
- Container memakai SPRING_CLOUD_CONFIG_URL=file:///config.yml.
- CI GitLab menjalankan go build -mod=vendor ./... dan golangci-lint. Tidak ada job go test di pipeline saat ini.
- Sebelum deploy, samakan port di config restapi.port dengan port container/service. Contoh config memakai 8088, Dockerfile mengekspos 8080.

Catatan Makefile: SHELL := /bin/bash dan memakai grep, date, rm. Di Windows gunakan Git Bash/WSL untuk make, atau jalankan perintah Go/sqlc langsung.

## 13. Scaffold project baru dari nol

Ikuti urutan ini agar agent tidak membuat struktur setengah jadi.

1. Buat module Go dan folder dasar.

~~~bash
mkdir -p cmd/api cmd/job1 cmd/pubsub-worker
mkdir -p internal/apiclient internal/config internal/echohttp internal/models
mkdir -p internal/repository/mantra_repository_sqlc internal/repository/redis
mkdir -p internal/service internal/testdata internal/worker
mkdir -p pkg/errors pkg/http/models pkg/pb pkg/utils
mkdir -p scripts/postgres/migrations scripts/postgres/queries docs

go mod init <module-path>
~~~

2. Salin baseline dari repo ini:

- go.mod, go.sum, Makefile, Dockerfile, .golangci.yaml, .editorconfig, .gitignore
- config.yml.example
- scripts/postgres/sqlc.yaml
- internal/config/*
- pkg/errors/errors.go
- pkg/utils/*
- internal/echohttp/{http_server.go,middleware_handler.go,logging_middleware.go,security_middleware.go,security_config.go,echohttp_error_handler.go}
- cmd/api/main.go
- internal/service/service.go

Catatan: jangan menjalankan go mod init setelah menyalin go.mod. Pilih salah satu jalur: mulai dari module kosong lalu salin baseline kecuali go.mod/go.sum, atau salin go.mod/go.sum lalu jalankan go mod edit -module [module-path].

3. Ganti semua import module path lama ke module path baru. Jangan biarkan satu import pun menunjuk ke git.alfamikro.com/back-office/mantra-backend.

4. Sesuaikan nama service di config.yml.example, Dockerfile, cloudbuild, dan ldflags. Jangan mempertahankan nama service-template.

5. Jalankan go mod tidy lalu go mod vendor. Pastikan build memakai -mod=vendor.

6. Buat migration pertama dan query pertama. Jalankan make gen.pg.models. Periksa package generated dan sesuaikan import sqlc.

7. Implementasikan satu vertical slice lengkap (bagian 14) sebelum menambah domain lain. Ini memvalidasi wiring config, DB, service, handler, route, error, dan test.

8. Jalankan:

~~~bash
go build -mod=vendor ./...
go test ./...
make lint
~~~

9. Baru tambahkan domain berikutnya dengan menyalin pola vertical slice, bukan menyalin isi domain lain.

## 14. Template vertical slice lengkap

Contoh domain widget. Ganti nama widget sesuai domain nyata, tetapi pertahankan urutan dan pembagian layer.

### 14.1 Migration

~~~sql
-- scripts/postgres/migrations/000012_add_widget.up.sql
CREATE SEQUENCE public.widget_seq START 1;

CREATE TABLE public.widget (
    id varchar(32) NOT NULL DEFAULT ('WGT_' || lpad(nextval('public.widget_seq')::text, 4, '0')),
    name varchar(120) NOT NULL,
    kode_mitra varchar(20) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT widget_pkey PRIMARY KEY (id)
);

CREATE INDEX widget_kode_mitra_idx ON public.widget (kode_mitra);

CREATE TABLE public.widget_log (
    id bigserial NOT NULL,
    widget_id varchar(32) NOT NULL,
    action varchar(32) NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT widget_log_pkey PRIMARY KEY (id),
    CONSTRAINT widget_log_widget_id_fkey FOREIGN KEY (widget_id) REFERENCES public.widget (id)
);

CREATE INDEX widget_log_widget_id_idx ON public.widget_log (widget_id);
~~~

~~~sql
-- scripts/postgres/migrations/000012_add_widget.down.sql
DROP TABLE IF EXISTS public.widget_log;
DROP TABLE IF EXISTS public.widget;
DROP SEQUENCE IF EXISTS public.widget_seq;
~~~

### 14.2 Query

~~~sql
-- scripts/postgres/queries/widget.sql
-- name: CreateWidget :one
INSERT INTO public.widget (name, kode_mitra)
VALUES (@name, @kode_mitra)
RETURNING id, name, kode_mitra, created_at;

-- name: ListWidgets :many
SELECT id, name, kode_mitra, created_at
FROM public.widget
WHERE kode_mitra = @kode_mitra
ORDER BY created_at DESC
LIMIT @limit_data OFFSET @offset_data;

-- name: InsertWidgetLog :exec
INSERT INTO public.widget_log (widget_id, action)
VALUES (@widget_id, @action);
~~~

Jalankan make gen.pg.models. Akan terbentuk CreateWidgetParams, ListWidgetsParams, dan tipe row hasil query. sqlc bisa memakai struct tabel Widget atau CreateWidgetRow/ListWidgetsRow tergantung kolom yang dikembalikan; selalu cek signature hasil generate.

### 14.3 Model

~~~go
// internal/models/widget.go
package models

import "time"

type CreateWidgetRequest struct {
    Name string `json:"name" validate:"required,min=3,max=120"`
}

type Widget struct {
    ID        string    `json:"id"`
    Name      string    `json:"name"`
    KodeMitra string    `json:"kode_mitra"`
    CreatedAt time.Time `json:"created_at"`
}
~~~

### 14.4 Service read

~~~go
func (s *Service) ListWidgets(
    ctx context.Context,
    workerInfo *models.WorkerInfo,
    paging pkgModels.CursorPaging,
) ([]models.Widget, error) {
    ctx, span := tracer.ServiceFuncSpan(ctx)
    defer span.End()

    offset := 0
    if paging.NextCursor != "" {
        parsed, err := strconv.Atoi(paging.NextCursor)
        if err != nil {
            return nil, pkgErr.ErrInvalidRequest
        }
        offset = parsed
    }

    q := sqlc.New(s.db)
    rows, err := q.ListWidgets(ctx, sqlc.ListWidgetsParams{
        KodeMitra:  workerInfo.KodeMitra,
        LimitData:  int32(paging.GetLimit()),
        OffsetData: int32(offset),
    })
    if err != nil {
        return nil, errors.Wrap(err, "list widgets")
    }

    result := make([]models.Widget, 0, len(rows))
    for _, row := range rows {
        result = append(result, models.Widget{
            ID:        row.ID,
            Name:      row.Name,
            KodeMitra: row.KodeMitra,
            CreatedAt: row.CreatedAt,
        })
    }

    return result, nil
}
~~~

### 14.5 Service create dengan transaksi

~~~go
func (s *Service) CreateWidget(
    ctx context.Context,
    workerInfo *models.WorkerInfo,
    req models.CreateWidgetRequest,
) (resp models.Widget, err error) {
    ctx, span := tracer.ServiceFuncSpan(ctx)
    defer span.End()

    if !workerInfo.IsMandor() {
        return resp, pkgErr.ErrInvalidAccessRole
    }

    tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return resp, pkgErr.ErrFailInitializeTx
    }

    defer func() {
        if err != nil {
            _ = tx.Rollback(ctx)
        }
    }()

    q := sqlc.New(tx)

    row, err := q.CreateWidget(ctx, sqlc.CreateWidgetParams{
        Name:      req.Name,
        KodeMitra: workerInfo.KodeMitra,
    })
    if err != nil {
        return resp, errors.Wrap(err, "create widget")
    }

    if err = q.InsertWidgetLog(ctx, sqlc.InsertWidgetLogParams{
        WidgetID: row.ID,
        Action:   "created",
    }); err != nil {
        return resp, errors.Wrap(err, "insert widget log")
    }

    if err = tx.Commit(ctx); err != nil {
        return resp, pkgErr.ErrFailCommitTx
    }

    return models.Widget{
        ID:        row.ID,
        Name:      row.Name,
        KodeMitra: row.KodeMitra,
        CreatedAt: row.CreatedAt,
    }, nil
}
~~~

Catatan: InsertWidgetLog sudah disertakan pada migration dan query di atas agar contoh transaksi benar-benar lengkap. Jika domain nyata hanya butuh satu insert, transaksi boleh dilewati; jika ada dua efek tulis atau lebih, gunakan pola ini.

### 14.6 Handler

~~~go
func createWidget(s *service.Service) echo.HandlerFunc {
    return func(c echo.Context) error {
        ctx, span := tracer.NewSpan(c.Request().Context(), tracer.SpanLvlHandler)
        defer span.End()

        workerInfo := c.Get(intCfg.KeyWorkerInfoJWTContextKey).(*models.WorkerInfo)

        var req models.CreateWidgetRequest
        if err := c.Bind(&req); err != nil {
            return err
        }
        if err := c.Validate(req); err != nil {
            return err
        }

        resp, err := s.CreateWidget(ctx, workerInfo, req)
        if err != nil {
            return err
        }

        return c.JSON(http.StatusCreated, resp)
    }
}
~~~

### 14.7 Route

~~~go
v1.POST("/widget", createWidget(s))
v1.GET("/widget", listWidgets(s))
~~~

### 14.8 Error dan config

Tambahkan sentinel ke pkg/errors/errors.go dan case ke mappingMsgError. Jika domain punya batas atau feature flag, tambahkan key di internal/config/config_keys.go dan contoh di config.yml.example.

### 14.9 Test

Tambahkan fixture internal/testdata/widget.yaml dan test di internal/service/widget_service_test.go dengan pola NewServiceProxy + AssertLoadPgxDBFixtures.

## 15. Checklist definition of done

- [ ] Migration up dan down lengkap, nomor urut benar.
- [ ] Query ditulis di scripts/postgres/queries dengan anotasi sqlc yang benar.
- [ ] make gen.pg.models berhasil dan file generated ikut di-commit.
- [ ] Service memuat aturan bisnis, authorization, dan mapping model.
- [ ] Transaksi dibuka di service dengan pgxpool.BeginTx.
- [ ] Semua query dalam transaksi memakai sqlc.New(tx), bukan sqlc.New(s.db).
- [ ] Error baru punya sentinel dan mapping HTTP.
- [ ] Handler hanya bind, validate, ambil identity, panggil service, tulis JSON.
- [ ] Route terdaftar di http_server.go pada group yang tepat.
- [ ] Config baru memakai konstanta dan ada contoh YAML.
- [ ] Test integration untuk happy path dan minimal satu failure path.
- [ ] go build -mod=vendor ./... lulus.
- [ ] go test ./... lulus (Docker aktif).
- [ ] make lint lulus tanpa error baru.
- [ ] Tidak ada generated file yang diedit manual.
- [ ] Tidak ada secret di config.yml, fixture, atau kode.

## 16. Kondisi placeholder di repo ini

Bagian ini penting agar agent tidak menganggap fitur yang belum ada sebagai fitur yang sudah jalan.

| Area | Kondisi | Implikasi |
|---|---|---|
| cmd/job1 | Hanya log lalu exit | Belum ada job nyata |
| cmd/pubsub-worker | main() kosong | Belum ada consumer |
| internal/worker | Hanya package kosong | Tambahkan consumer per file saat dipakai |
| Redis | Repository ada, belum di-wire dan belum ada config | Jangan pakai sebelum menambah wiring |
| Mandor middleware | Ada tetapi tidak didaftarkan | Role check saat ini di service |
| survey.sql.go | Generated lama, sumber query tidak ada | Jangan dijadikan contoh; regenerate/hapus bila aman |
| Dockerfile/cloudbuild | Masih memakai nama service-template | Ganti saat membuat project baru |
| config.yml.example | Belum memuat blok security | Lengkapi untuk environment production |
| Port | Config 8088, Dockerfile 8080 | Samakan saat deploy |
| CI | Belum ada job go test | Jalankan test lokal sebelum merge |

Placeholder bukan alasan untuk menyalin kebiasaan buruk. Untuk kode baru, gunakan pola yang benar di dokumen ini.

## 17. Prompt siap pakai untuk init agent

Tempel prompt berikut ke agent baru setelah file AGENTS.md ada di root project:

~~~text
Kamu adalah coding agent untuk project Go ini. Baca AGENTS.md sampai selesai sebelum menulis kode.

Tugasmu: scaffold project baru yang mengikuti blueprint ini persis, atau tambahkan satu domain baru ke project yang sudah ada.

Aturan yang tidak boleh dilanggar:
- cmd/ hanya untuk wiring dan bootstrap.
- Handler di internal/echohttp harus tipis; tidak ada SQL dan tidak ada transaksi.
- Business logic dan transaksi hanya di internal/service.
- Transaksi memakai pgxpool.BeginTx dan sqlc.New(tx); jangan membuat repository hand-written.
- SQL source di scripts/postgres/queries; generated code di internal/repository/mantra_repository_sqlc tidak boleh diedit.
- Error baru harus punya sentinel di pkg/errors dan mapping di echohttp_error_handler.go.
- Selalu teruskan context dan gunakan Asia/Jakarta untuk waktu bisnis.

Langkah kerja:
1. Baca struktur folder dan file terkait sebelum mengubah apa pun.
2. Buat migration up/down dan query sqlc.
3. Jalankan make gen.pg.models.
4. Implementasikan service, handler, route, model, dan error mapping.
5. Tambahkan test integration dan fixture.
6. Jalankan go build -mod=vendor ./..., go test ./..., dan make lint.
7. Laporkan file yang berubah, hasil test, dan risiko yang tersisa.

Jika ada aturan yang tidak jelas, ikuti pola kode terdekat di repo dan jelaskan asumsinya.
~~~

Dokumen ini adalah baseline. Jika struktur project berubah secara material, perbarui AGENTS.md pada perubahan yang sama.

