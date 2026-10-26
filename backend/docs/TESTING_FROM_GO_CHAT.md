# Testing from go-chat

> Implementasi aktual di backend ini: `internal/testkit/postgres.go` memakai
> `TEST_DATABASE_URL` (bukan testcontainers), me-reset schema `public`, lalu
> menjalankan migration 000001–000003. CI menyediakan service `postgres:16-alpine`
> dan mengeset `TEST_REQUIRE_DB=1`. Lokal memakai service `db-test` pada
> `compose.yaml`. Panduan testcontainers di bawah tetap menjadi referensi bila
> nanti harness ingin dibuat sepenuhnya self-contained.

This document records how the reference project `D:/appwebsite/go-chat` tests a Go
service backed by PostgreSQL, and translates those patterns into a concrete plan
for this Echo/sqlc backend. The reference was read only. No file in go-chat was
modified, and this document is the only file added to the target project.

Versions observed in go-chat: `testcontainers-go v0.44.0`,
`testcontainers-go/modules/postgres v0.44.0`, `go-testfixtures/v3 v3.19.0`,
`lib/pq v1.12.3`, and the PostgreSQL image `postgres:16-alpine`. The target uses
Go 1.24.2, pgx v5, Echo v4, and sqlc v1.31.1.

The short version: copy the harness shape (one disposable database per package,
real migrations, per-test reset, YAML fixtures, explicit Docker gating), but do
not copy go-chat's test mix as-is. Most of go-chat's service tests are database
integration tests. This backend should add a genuine pure-unit layer for UMKM
rules and auth validation, a narrow fake-query layer for service orchestration,
and Docker-backed tests for the SQL and transaction behaviour.

## 1. What go-chat tests, and what each layer proves

| Layer | Main files | What it verifies | Needs Docker |
|---|---|---|---|
| Pure unit | `pkg/httpclient/*_test.go`, `pkg/auth/*_test.go`, `internal/config/*_test.go`, `internal/health/health_test.go`, `internal/worker/*_test.go` | Parsing, validation, retry policy, URL redaction, JWT/cookie helpers, health handlers, worker classification. No database or network. | No |
| Service-proxy integration | `internal/services/*_test.go`, especially `internal/services/service_test_utils_test.go` | Real use cases against real PostgreSQL and real repositories, with miniredis and fakes for external systems. Covers transactions, cache invalidation, idempotency, outbox rows, OAuth callback flow. | Yes |
| Repository integration | `internal/repository/order_repository_test.go`, `internal/repository/oauth_states_repository_test.go`, `internal/repository/firebase_repository_test.go` | Generated/query behaviour and constraints. Tests call the repository, then read rows directly to prove the write. | Yes |
| HTTP/router | `internal/router/router_test.go`, `pkg/auth/http/http_test.go`, `internal/health/health_test.go` | Route registration, middleware rejection, bearer/cookie handling, JSON handler envelopes. Mostly `httptest`, no listening socket. | No |
| External-system integration | `pkg/storage/minio_integration_test.go`, `pkg/message-broker/kafka/integration_test.go`, `pkg/message-broker/rabbitmq/consumer_integration_test.go` | Real MinIO, Kafka/Redpanda, RabbitMQ. These use env-gated containers and skip when Docker is absent unless a `TEST_REQUIRE_*` variable is set. | Yes |
| Legacy reference only | `unitesting-example/service/*` and `unitesting-example/testdata/*` | An older Mantra pattern with a `ServiceProxy`, a private `pgtestutil` package, and per-test fixture loading. It is a nested module and is not compiled by `go build ./...`. | Yes |

Two observations matter for this backend:

1. go-chat's service tests are not isolated unit tests. `newTestEnv` wires every
   service to real repositories over a PostgreSQL container. Fakes are used for
   Redis, storage, the event publisher, and OAuth providers, not for SQL.
2. The pure layer is package-level. Business rules that can run without a
   database, such as `CreateDataTransactionAt`, are exported and tested directly.
   This is the pattern to extend for UMKM CSV parsing, suggestions, balances,
   and export escaping.

The `internal/services/oauth_provider_fakes_test.go` file is the best example of
edge fakes: `httptest.Server` instances stand in for GitHub and Google, including
PKCE verification, so the callback flow runs without credentials or internet.

## 2. The PostgreSQL harness in go-chat

### 2.1 Lifecycle

`internal/testkit/testkit.go` is a small package imported by every integration
test package. It owns the whole lifetime:

- `func TestMain(m *testing.M) { os.Exit(testkit.Main(m)) }` is repeated in each
  integration package. `Main` runs the package tests and then terminates the
  container with a 30 second timeout.
- One container is started per package, lazily on the first `testkit.New(t)`,
  guarded by `sync.Once`. Every test in the package shares it.
- The image is pinned in code: `postgres:16-alpine`.
- Startup waits for the log line `database system is ready to accept connections`
  with occurrence 2 and a 90 second timeout.
- The container database is named `chat_test`; the user and password are the
  throwaway values `test`/`test`.
- `New(t)` skips the test with a clear message when Docker cannot be reached,
  unless `TEST_REQUIRE_DB` is set. CI sets `TEST_REQUIRE_DB=1` so a broken Docker
  provider fails the job.

### 2.2 Migrations

`testkit.migrate` does the following:

1. If `TEST_DATABASE_URL` is set, connect to that database instead of starting a
   container.
2. Skip migrations when a sentinel table already exists. go-chat checks
   `to_regclass('public.users')`.
3. Otherwise find `migrations/*.up.sql`, sort the file names lexicographically,
   read each file, and execute it as one `Exec`.
4. Fail with the migration file name in the error, for example
   `apply 000002_umkm_normalized.up.sql: ...`.
5. Discover the migrations directory by walking up from the package directory,
   or use `TEST_MIGRATIONS_DIR`.

This is sufficient for a fresh container but it has two weaknesses that matter
for this project:

- The sentinel check is hard-coded to `public.users`. This backend has no such
  table.
- Executing a multi-statement file through pgx's default prepared-statement mode
  fails with `cannot insert multiple commands into a prepared statement`. The
  target migrations are multi-statement SQL. The adapted harness must either use
  a migration library such as golang-migrate (which the Makefile already uses),
  or open a database/sql connection with
  `default_query_exec_mode=simple_protocol` and keep a migration ledger.

### 2.3 Reset and fixtures

`env.Reset(t)` is called at the start of every test. It queries
`pg_tables` where `schemaname = 'public'`, quotes every table name, and runs:

```sql
TRUNCATE TABLE "users", "rooms", ... RESTART IDENTITY CASCADE;
```

`env.Load(t, "testdata/rooms.yaml", ...)` then loads go-testfixtures multi-table
YAML files. Each top-level key is a table and each item is a row. Files are
loaded in the order passed by the test. Fixture loading replaces the contents of
the tables it declares, so `Reset` first guarantees no rows leak from an earlier
test. A test that needs several related tables passes all of them in dependency
order, for example the Mantra example loads `ms_store`, `ms_mitra`,
`ms_karyawan`, `ms_pekerja`, and so on before `agenda_mitra`.

Timestamps that must be in the future are written as literal dates, for example
`expired_at: "2099-01-01T00:00:00Z"`. This avoids clock-dependent fixtures.

The reference relies on the container user being a superuser so go-testfixtures
can disable foreign-key triggers while loading. That works for a disposable
container. When `TEST_DATABASE_URL` points at an existing database, the YAML
order must satisfy foreign keys, and go-chat enables
`testfixtures.DangerousSkipTestDatabaseCheck`.

### 2.4 Cleanup and parallel safety

- Container termination: `testkit.Shutdown`, called by `TestMain`.
- Per-test cleanup: tests use `t.Cleanup` for miniredis clients, HTTP test
  servers, and other closeable resources. The database itself is reset by
  truncation, not by a transaction rollback.
- The shared database means tests in one package are sequential. go-chat has no
  `t.Parallel()` calls anywhere. Parallel packages are safe because each package
  gets its own container.
- A per-test transaction rollback is not used, because services open their own
  transactions and commit them. Truncation is the reset mechanism.
- If `TestMain` is missing from an integration package, the container is not
  terminated by the harness. Every integration package must define it.

### 2.5 Adapted harness skeleton for this backend

Create `backend/internal/testkit/testkit.go` (package `testkit`). The important
differences from go-chat are a project-specific migration ledger, a guard against
truncating a non-test database, and both a `*sql.DB` for fixtures and a
`*database.DB` for service code. The `start` and `shutdown` helpers are omitted
from the snippet; use go-chat's shape with the 90 second startup wait and the
30 second terminate.

```go
package testkit

import (
    "context"
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "sync"
    "testing"
    "time"

    "github.com/AhmadKusumahDEV/rekonsiliasi-kas-umkm/backend/pkg/database"
    "github.com/go-testfixtures/testfixtures/v3"
    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/testcontainers/testcontainers-go"
    tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

const image = "postgres:16-alpine"

type Env struct {
    SQL    *sql.DB // fixtures, resets, direct assertions
    DSN    string
    reused bool
}

var (
    once    sync.Once
    shared  *Env
    initErr error
)

func Main(m *testing.M) int {
    code := m.Run()
    shutdown()
    return code
}

func New(t *testing.T) *Env {
    t.Helper()
    once.Do(func() { shared, initErr = start(context.Background()) })
    if initErr != nil {
        if os.Getenv("TEST_REQUIRE_DB") == "" {
            t.Skipf("postgres test container unavailable: %v", initErr)
        }
        t.Fatalf("testkit: %v", initErr)
    }
    return shared
}

func (e *Env) NewDB(t *testing.T) *database.DB {
    t.Helper()
    db, err := database.NewFromURL(context.Background(), e.DSN)
    if err != nil {
        t.Fatalf("testkit: open service database: %v", err)
    }
    t.Cleanup(db.Close)
    return db
}

func (e *Env) Reset(t *testing.T) {
    t.Helper()
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    rows, err := e.SQL.QueryContext(ctx, `
        select tablename from pg_tables
        where schemaname = 'public' and tablename <> 'schema_migrations'`)
    if err != nil {
        t.Fatalf("testkit: list tables: %v", err)
    }
    defer rows.Close()

    var tables []string
    for rows.Next() {
        var name string
        if err := rows.Scan(&name); err != nil {
            t.Fatalf("testkit: scan table: %v", err)
        }
        tables = append(tables, quoteIdentifier(name))
    }
    if err := rows.Err(); err != nil {
        t.Fatalf("testkit: list tables: %v", err)
    }
    if len(tables) == 0 {
        return
    }
    if _, err := e.SQL.ExecContext(ctx,
        "truncate table "+strings.Join(tables, ", ")+" restart identity cascade"); err != nil {
        t.Fatalf("testkit: truncate: %v", err)
    }
}

func quoteIdentifier(name string) string {
    return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (e *Env) Load(t *testing.T, files ...string) {
    t.Helper()
    if len(files) == 0 {
        return
    }
    opts := []func(*testfixtures.Loader) error{
        testfixtures.Database(e.SQL),
        testfixtures.Dialect("postgres"),
        testfixtures.FilesMultiTables(files...),
    }
    if e.reused {
        if os.Getenv("TEST_ALLOW_NON_TEST_DB") == "" {
            t.Fatalf("testkit: TEST_DATABASE_URL must name a disposable test database")
        }
        opts = append(opts, testfixtures.DangerousSkipTestDatabaseCheck())
    }
    loader, err := testfixtures.New(opts...)
    if err != nil {
        t.Fatalf("testkit: build fixture loader: %v", err)
    }
    if err := loader.Load(); err != nil {
        t.Fatalf("testkit: load fixtures: %v", err)
    }
}
```

Startup follows the reference: `tcpostgres.Run` with database
`rekonsiliasi_test`, user `test`, password `test`, and the same log wait strategy.
Connect with `sql.Open("pgx", dsn)`. The DSN returned by the container should be
opened with `default_query_exec_mode=simple_protocol` for the migration step.
After migrations, `database.NewFromURL` can use the same DSN with pgx's normal
mode. If golang-migrate is used instead, keep its `schema_migrations` table out
of the truncate list as shown above.

### 2.6 Migration strategy

Use one of these, consistently:

1. **golang-migrate library** (preferred if adding a test dependency is
   acceptable). The migration files already follow its
   `NNNNNN_name.up.sql` / `.down.sql` convention, and the Makefile already uses
   the CLI. This gives a real `schema_migrations` ledger and avoids the
   multi-statement problem. Point the file source at the absolute
   `scripts/postgres/migrations` directory discovered by walking up from the
   package.
2. **Simple protocol plus a small ledger** (no new dependency). Open a
   database/sql connection with `default_query_exec_mode=simple_protocol`, read
   `*.up.sql` in sorted order, and record each applied file name in a
   `schema_migrations` table inside the same transaction as the DDL. Skip files
   already recorded. This is the least surprising option for a test-only
   harness.

Do not copy go-chat's "check whether one sentinel table exists" rule. It hides
schema drift when a new migration is added and the developer reuses a local
database.

### 2.7 Fixture rules for this schema

Fixtures must respect the constraints in
`scripts/postgres/migrations/000002_umkm_normalized.up.sql`:

- `umkm_imports.id`, `umkm_entries.id`, and `umkm_allocations.id` are exactly
  32 lowercase hex characters. Use fixed values such as
  `aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`.
- `pblg_workspaces.code` is globally unique. Tests that create multiple
  workspaces need distinct codes, even across subtests.
- `umkm_entries.channel` must be lowercase, `amount` is a positive bigint up to
  `1000000000000`, and `source_id` is 1-80 characters.
- `(workspace_id, kind, source_id)` is unique. A fixture cannot contain two
  sales with the same source ID.
- `umkm_entries` belongs to an import through a composite foreign key on
  `(workspace_id, batch_id)`. Load imports before entries.
- `umkm_allocations` references entries through two composite foreign keys.
  Load entries before allocations.
- Load order for a full UMKM fixture:
  `pblg_workspaces`, `pblg_users`, `pblg_sessions`, `umkm_imports`,
  `umkm_entries`, `umkm_allocations`, `pblg_audit`.
- Do not put a real bcrypt hash in YAML. Use a placeholder nonempty value for
  tests that never log in, and seed a real hash with a helper for login tests:

```go
func SetPassword(t *testing.T, db *sql.DB, userID, plaintext string) {
    t.Helper()
    hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.MinCost)
    if err != nil {
        t.Fatalf("hash password: %v", err)
    }
    if _, err := db.Exec("update pblg_users set password_hash = $1 where id = $2",
        string(hash), userID); err != nil {
        t.Fatalf("set password: %v", err)
    }
}
```

Use `bcrypt.MinCost` only for test data generation; production code keeps cost
12. For session fixtures, do not try to write a plaintext token: the table stores
`sha256(token)`. Insert the session in a helper so the test knows the plaintext:

```go
func InsertSession(t *testing.T, db *sql.DB, token, userID, csrf string, expires time.Time) {
    t.Helper()
    sum := sha256.Sum256([]byte(token))
    if _, err := db.Exec(`
        insert into pblg_sessions (token_hash, user_id, csrf, expires_at)
        values ($1, $2, $3, $4)`,
        hex.EncodeToString(sum[:]), userID, csrf, expires); err != nil {
        t.Fatalf("insert session: %v", err)
    }
}
```

## 3. How go-chat separates business logic from database access

go-chat declares repository interfaces next to their implementations, for
example `repository.RepositoryUser`, `repository.RepositoryRoom`, and
`repository.OrderRepository`. Service constructors accept those interfaces, and
the service-proxy test environment injects concrete repositories built on the
test database. External dependencies are the only things faked:

- `miniredis` runs the real cache implementation in memory.
- `testkit.FakePublisher` records events instead of publishing them.
- `testkit.FakeStorage` satisfies the storage interface with fixed URLs.
- `httptest.Server` fakes OAuth providers.
- Worker tests define narrow fake stores such as `fakeDeviceStore` and
  `fakeOauthStateStore` where only one method matters.

There is no generated-sqlc mock and no hand-written fake of `pgx.Tx`. That is
the right boundary. Faking pgx transactions is brittle; transaction semantics are
exactly what needs a real database.

For this backend, add two seams rather than mocking sqlc internals:

1. **Pure rule functions** in `internal/service`. Keep them free of
   `database.DB` and sqlc. Examples:

```go
func parseImportCSV(kind string, data []byte) ([]models.Entry, string, error)
func suggestMatches(sales, receipts []models.Entry, active []models.Allocation) []models.Match
func planAllocation(sale, receipt models.Entry, active []models.Allocation, amount, fee int64) (models.Allocation, error)
func exportCSV(entries []models.Entry, allocations []models.Allocation) ([]byte, error)
```

These get table-driven tests with no Docker. The tests are fast enough to run on
every save and cover most of the difficult arithmetic and escaping.

2. **Narrow query interfaces owned by the service**, plus a transaction runner.
   The interfaces should be declared where they are consumed, in
   `internal/service`, and should use the generated parameter/row types so no
   mapping layer is invented just for tests.

```go
// internal/service/auth_queries.go
type AuthQueries interface {
    GetSessionUser(context.Context, sqlc.GetSessionUserParams) (sqlc.GetSessionUserRow, error)
    GetUserForLogin(context.Context, sqlc.GetUserForLoginParams) (sqlc.GetUserForLoginRow, error)
    CreateSession(context.Context, sqlc.CreateSessionParams) error
    DeleteSessionByTokenHash(context.Context, string) (int64, error)
    GetUserPasswordHash(context.Context, string) (string, error)
    UpdateUserPasswordHash(context.Context, sqlc.UpdateUserPasswordHashParams) (int64, error)
    DeleteSessionsByUserID(context.Context, string) (int64, error)
}

var _ AuthQueries = (*sqlc.Queries)(nil)

// internal/service/umkm_queries.go
type UMKMQueries interface {
    GetUmkmAllocationByID(context.Context, sqlc.GetUmkmAllocationByIDParams) (sqlc.UmkmAllocation, error)
    GetUmkmEntriesByIDs(context.Context, sqlc.GetUmkmEntriesByIDsParams) ([]sqlc.UmkmEntry, error)
    ListUmkmActiveAllocationsByEntryIDs(context.Context, sqlc.ListUmkmActiveAllocationsByEntryIDsParams) ([]sqlc.UmkmAllocation, error)
    CreateUmkmAllocation(context.Context, sqlc.CreateUmkmAllocationParams) (sqlc.UmkmAllocation, error)
    MarkUmkmAllocationReversed(context.Context, sqlc.MarkUmkmAllocationReversedParams) (int64, error)
    ListUmkmUnallocatedEntries(context.Context, string) ([]sqlc.UmkmEntry, error)
}

var _ UMKMQueries = (*sqlc.Queries)(nil)

// internal/service/tx.go
type TxRunner interface {
    InTx(ctx context.Context, fn func(UMKMQueries) error) error
}
```

The production adapter still opens the transaction in the service layer, as the
project rules require:

```go
type dbTxRunner struct{ db *database.DB }

func (r dbTxRunner) InTx(ctx context.Context, fn func(UMKMQueries) error) (err error) {
    tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
    if err != nil {
        return pkgErr.ErrFailInitializeTx
    }
    defer func() {
        if err != nil {
            _ = tx.Rollback(ctx)
        }
    }()
    if err = fn(sqlc.New(tx)); err != nil {
        return err
    }
    if err = tx.Commit(ctx); err != nil {
        return pkgErr.ErrFailCommitTx
    }
    return nil
}
```

A service test then uses a fake transaction runner that calls the callback with a
fake query set. No database is involved:

```go
type fakeTx struct {
    q     *fakeUMKMQueries
    calls int
}

func (f *fakeTx) InTx(_ context.Context, fn func(UMKMQueries) error) error {
    f.calls++
    return fn(f.q)
}
```

The current `internal/service/auth_service.go` calls
`sqlc.New(s.db.Pool())` directly. It does not need a large refactor to get
useful tests: extract the validators and session helpers first
(`normalizeEmail`, `validEmail`, `passwordOK`, `validName`, `hashToken`), test
them in a white-box file, and then introduce `AuthQueries`/`TxRunner` when the
service is split into per-domain constructors. Do not introduce a database
interface that pretends to be pgx; keep the transaction runner small.

For auth speed and determinism, add two more seams:

```go
type PasswordHasher interface {
    Hash(plain string) (string, error)
    Compare(hash, plain string) error
}

type Clock func() time.Time
```

Production uses bcrypt cost 12 and `utils.NowJakarta`. Tests use
`bcrypt.MinCost` or a fake hasher, and a fixed clock. This avoids hundreds of
milliseconds per login test and makes session expiry deterministic.

The target service should keep returning the project's sentinel errors from
`pkg/errors`. Tests should assert with `errors.Is` against those sentinels and
check the safe message separately, rather than matching raw SQL errors.

## 4. Recommended test layout for this backend

```text
backend/
├── internal/
│   ├── testkit/
│   │   ├── testkit.go                 # container, migrations, reset, fixtures
│   │   ├── seed.go                    # SetPassword, InsertSession, seedOwner
│   │   └── README.md
│   ├── testdata/
│   │   ├── auth/
│   │   │   ├── owner.yaml
│   │   │   └── two_workspaces.yaml
│   │   ├── umkm/
│   │   │   ├── sale_receipt.yaml
│   │   │   ├── allocated.yaml
│   │   │   ├── canceled_import.yaml
│   │   │   └── export_mixed.yaml
│   │   └── csv/
│   │       ├── sale_valid.csv
│   │       ├── receipt_valid.csv
│   │       ├── duplicate_source.csv
│   │       ├── invalid_header.csv
│   │       └── formula_injection.csv
│   ├── service/
│   │   ├── auth_domain_test.go        # package service (exists; extend)
│   │   ├── auth_service_test.go       # package service_test
│   │   ├── umkm_domain_test.go        # package service (exists; extend)
│   │   ├── umkm_import_test.go        # package service_test
│   │   ├── umkm_suggestions_test.go   # package service_test
│   │   ├── umkm_allocate_test.go      # package service_test
│   │   ├── umkm_cancel_test.go        # package service_test
│   │   ├── umkm_export_test.go        # package service_test
│   │   ├── service_test_utils_test.go # package service_test
│   │   └── fakes_test.go              # package service_test
│   ├── echohttp/
│   │   ├── error_handler_test.go      # package echohttp (white box)
│   │   ├── auth_handler_test.go       # package echohttp_test
│   │   ├── umkm_handler_test.go       # package echohttp_test
│   │   └── umkm_flow_test.go          # package echohttp_test, Docker-backed
│   └── repository/
│       └── rekonsiliasi_repository_sqlc/
│           └── query_integration_test.go # package sqlc_test
├── integration/
│   └── migrations_test.go             # package integration_test, backfill
└── docs/
    └── TESTING_FROM_GO_CHAT.md
```

### 4.1 Pure UMKM and auth tests

These files have no Docker, no build tag, and no `testkit` import. Two pure test
files already exist on this branch; extend them instead of adding parallel rule
files with different names.

- `internal/service/auth_domain_test.go` (package `service`, exists): keep the
  table tests for `normalizeEmail`, `validEmail`, `passwordOK`, `validName`,
  `hashToken`, the allowed owner role, and the rejection of a role supplied by
  the client. Add cases for a 12-byte minimum password, a 72-byte maximum, and a
  name containing NUL.
- `internal/service/umkm_domain_test.go` (package `service`, exists): keep the
  CSV parsing, suggestion, allocation, and export tests here. Expand them with
  BOM and non-BOM input, exact seven-column header, invalid UTF-8, NUL rejection,
  per-cell length, duplicate source IDs in one file, invalid date, non-integer
  and over-limit amounts, checksum stability; suggestion key normalization,
  0-3 day window, uniqueness in both directions, and exclusion of entries with
  active allocations; balance arithmetic including fee; export BOM and formula
  neutralization. Use table-driven subtests rather than one long assertion block.
- `internal/echohttp/error_handler_test.go` (exists; package `echohttp`): `mapHTTPError`
  and `validationMessage` for every sentinel, Echo's 404/405/413/415, context
  cancellation, and deadline exceeded. This is much cheaper than reproducing
  all mappings through the database.

### 4.2 Service tests with query interfaces and fakes

These are black-box tests in `package service_test`. They construct the service
with a fake `AuthQueries`/`UMKMQueries`, a fake `TxRunner`, a fake hasher, and a
fixed clock. They cover orchestration and rule ordering, not SQL semantics.

- `internal/service/auth_service_test.go`: valid login returns a session and
  records the token hash; unknown email and wrong password return the same
  `ErrUnauthorized`; register rejects a non-owner role before opening a
  transaction; a duplicate email from `CreateUser` returns `ErrConflict` and the
  transaction is rolled back; change password rejects a wrong current password,
  replaces the hash, and deletes all sessions in the same unit of work; expired
  session is unauthorized.
- `internal/service/umkm_import_test.go`: duplicate checksum with an active
  import returns duplicate with zero new rows; duplicate checksum on a canceled
  import is a conflict; identical source IDs are skipped; a conflicting source ID
  fails the whole import and the fake transaction sees a rollback; the 10,000
  entry cap and 500 row file cap are enforced before any insert.
- `internal/service/umkm_suggestions_test.go`: candidates come only from active,
  referenced, unallocated entries; a pair is not suggested if either side has
  more than one candidate; no write methods are called.
- `internal/service/umkm_allocate_test.go`: kind mismatch, canceled entry,
  missing entry, sale balance exceeded, receipt balance exceeded, fee without a
  reason, short reason, and a replayed key with a different payload. The happy
  path asserts `CreateUmkmAllocation`, audit, and version bump happen in one
  transaction.
- `internal/service/umkm_cancel_test.go`: an active allocation blocks cancel;
  after undo, cancel marks the import and all entries; a repeated cancel is a
  successful no-op.
- `internal/service/umkm_export_test.go`: a fake query result with active and
  canceled entries proves the CSV contains both, computes remaining amounts, and
  prefixes cells beginning with `=`, `+`, `-`, or `@`.

The fake query types live in `fakes_test.go`, not in `testkit`. `testkit` should
know about PostgreSQL and fixtures, not about service interfaces.

### 4.3 Docker-backed service and repository tests

Add one `TestMain` to `internal/service` tests (the same package directory can
contain `package service` and `package service_test` test files). The environment
helper always resets and loads a scenario fixture:

```go
type testEnv struct {
    sqlDB *sql.DB
    db    *database.DB
    svc   *service.Service
}

func newTestEnv(t *testing.T) *testEnv {
    t.Helper()
    env := testkit.New(t)
    env.Reset(t)
    db := env.NewDB(t)
    return &testEnv{sqlDB: env.SQL, db: db,
        svc: service.NewService(testConfig(), db, nil)}
}

func TestMain(m *testing.M) {
    os.Exit(testkit.Main(m))
}
```

Use this layer for the things a fake cannot prove:

- `register` creates workspace, user, and session atomically; a failing user
  insert leaves no workspace.
- `login` compares the real bcrypt hash and writes a session row with the
  SHA-256 token hash, not the plaintext token.
- `import` respects the composite foreign key, unique constraints, and the
  checksum uniqueness per workspace.
- `suggestions` is read-only and matches the real `ListUmkmUnallocatedEntries`
  query.
- `allocate` obtains `LockWorkspaceForUpdate`, sees active allocations through
  the real query, and writes allocation, audit, and version bump in one
  transaction.
- `undo` sets `reversed_at` and restores balances without deleting the row.
- `cancel` refuses when `CountUmkmActiveAllocationsForImport` is nonzero and
  otherwise cancels the import and every entry.
- `export` reads the real rows and produces a stable CSV.

Repository tests should call the generated queries directly, not through the
service, to catch SQL mistakes:

```go
func TestListUmkmUnallocatedEntriesSkipsReferencedOrEmptyReferences(t *testing.T) {
    env := testkit.New(t)
    env.Reset(t)
    env.Load(t, "../testdata/umkm/sale_receipt.yaml")

    q := sqlc.New(env.NewDB(t).Pool())
    rows, err := q.ListUmkmUnallocatedEntries(context.Background(), testWorkspaceID)
    // assert only the referenced, active, unallocated rows are returned
}
```

Do not edit or regenerate generated files to make a test pass. If a query is
wrong, change `scripts/postgres/queries` and run `make gen.pg.models`.

### 4.4 Echo handler and router tests

The existing `internal/echohttp/http_server_test.go` is a good start: it builds
the real engine with `NewEcho`, uses `httptest.NewRecorder`, and asserts liveness,
readiness, and the JSON 404 envelope.

For handler tests:

- `internal/echohttp/auth_handler_test.go`: table-driven invalid JSON, missing
  fields, bad email, short password, wrong content type, and oversized body.
  These paths fail before the service needs a database, so a service built with
  `service.NewService(cfg, nil, nil)` is enough.
- `internal/echohttp/umkm_handler_test.go`: invalid 32-hex IDs, negative or zero
  amount, negative fee, fee without reason, oversized CSV, malformed CSV, and
  unknown route. Assert the status and the safe JSON message, and assert that no
  write occurred.
- `internal/echohttp/umkm_flow_test.go`: Docker-backed end-to-end HTTP flow with
  a real cookie jar. This is the scenario in section 7.
- Keep `package echohttp` white-box tests only for unexported mapping functions.
  Handler tests that call the public engine belong in `package echohttp_test`.

If handler success paths become hard to test with the concrete
`*service.Service`, introduce narrow interfaces at the handler boundary, for
example `type authAPI interface { Register(...); Login(...); GetSession(...) }`,
and keep `*service.Service` as the production implementation. Do not introduce
interfaces that mirror every service method; define one small interface per
handler group.

## 5. Running the tests locally and in CI

### 5.1 Makefile targets

The current `backend/Makefile` has only `test: go test ./...`. Add targets that
make the split explicit. The integration tests should skip on `-short` and fail
when Docker is required:

```make
test: ## Semua test; Docker dipakai bila tersedia
	go test -race -count=1 ./...

test.unit: ## Test cepat tanpa Docker
	go test -short -count=1 ./...

test.integration: ## Wajib Docker; gagal bila container tidak bisa start
	TEST_REQUIRE_DB=1 go test -race -count=1 -run 'Integration|Flow' ./internal/service ./internal/echohttp
```

On Windows, `make` may run through Git Bash or WSL; the environment prefix in
`test.integration` assumes a POSIX shell. Document the PowerShell equivalent in
the README or add a `test.integration.ps1` script if Windows developers do not
have `make`.

Every Docker-backed test file should start with:

```go
if testing.Short() {
    t.Skip("integration test: -short disables Docker")
}
```

and then call `testkit.New(t)`. When Docker is missing, `New` skips unless
`TEST_REQUIRE_DB=1`.

### 5.2 Windows with Docker Desktop

1. Start Docker Desktop and wait until `docker info` succeeds.
2. Use the WSL 2 backend. Testcontainers talks to Docker through the named pipe
   or `DOCKER_HOST`; if the Docker CLI works but Go tests cannot connect, set
   `DOCKER_HOST` explicitly for the shell.
3. Pull the image once to separate network problems from test problems:
   `docker pull postgres:16-alpine`.
4. Run from `D:\appwebsite\rekonsiliasi-kas-umkm\backend`:

```powershell
docker info
$env:TEST_REQUIRE_DB = "1"
go test -race -count=1 ./...
```

For a quick loop that does not start containers:

```powershell
go test -short -count=1 ./...
```

If Docker is not usable, point the harness at the compose database instead. This
is destructive to that database because `Reset` truncates every public table:

```powershell
docker compose up -d db
docker compose exec db createdb -U postgres rekonsiliasi_test
$env:TEST_DATABASE_URL = "postgres://postgres:postgres@localhost:5432/rekonsiliasi_test?sslmode=disable"
$env:TEST_ALLOW_NON_TEST_DB = "1"
go test -count=1 ./internal/service ./internal/echohttp
```

Use a dedicated database name containing `test`. The harness should refuse a DSN
whose database name does not look disposable unless the explicit override is set.

### 5.3 WSL

Run the repository from a Linux path, for example `~/src/rekonsiliasi-kas-umkm`,
rather than `/mnt/d/...` when file watching and container mounts are involved.
Docker Desktop must expose the engine to WSL, or Docker must run natively inside
the distribution. Then:

```bash
docker info
cd backend
TEST_REQUIRE_DB=1 go test -race -count=1 ./...
```

If the socket is not the default, export `DOCKER_HOST=unix:///var/run/docker.sock`.
Use `go test -short ./...` for the no-Docker loop. Do not run the Windows Go
toolchain and the WSL Go toolchain against the same module cache at the same
time.

### 5.4 Environment variables

| Variable | Purpose | Recommended value |
|---|---|---|
| `TEST_REQUIRE_DB` | Fail instead of skipping when Docker/Postgres is unavailable | `1` in CI |
| `TEST_DATABASE_URL` | Use an existing database instead of starting a container | Disposable database only |
| `TEST_MIGRATIONS_DIR` | Override migration discovery | `scripts/postgres/migrations` if walk-up fails |
| `TEST_ALLOW_NON_TEST_DB` | Permit `TEST_DATABASE_URL` with a non-`test` database name | Only for an explicitly disposable database |
| `DOCKER_HOST` | Point testcontainers at a non-default Docker endpoint | Set only when Docker Desktop/WSL needs it |
| `TESTCONTAINERS_RYUK_DISABLED` | Diagnostic switch when the cleanup sidecar is blocked | `true` only while diagnosing leaks |

Do not use production credentials. The container credentials are throwaway
values. Do not read any reference-project credential or Firebase file in a test.

### 5.5 CI

The current `.github/workflows/backend-ci.yml` already runs `go vet`,
`go test -race -count=1 ./...`, and a build. Once Docker-backed tests exist,
add a Docker preflight and require the database, otherwise the harness will skip
silently on a broken runner:

```yaml
- name: Docker preflight
  run: docker info

- name: Test
  env:
    TEST_REQUIRE_DB: "1"
  run: go test -race -count=1 ./...
```

Set `timeout-minutes: 20` or `25` for the test job. The container image pull is
the slow part on a cold runner. Keep the build job dependent on the test job.
For a faster feedback loop, split into `unit` (`go test -short ...`) and
`integration` (`TEST_REQUIRE_DB=1 go test -run ...`) jobs, but only after the
suite is large enough to justify the extra checkout and cache setup. Do not add
`-p 1` unless Docker memory is constrained; separate packages run in parallel and
each owns its container.

### 5.6 Failure diagnostics

| Symptom | Likely cause | Action |
|---|---|---|
| `postgres test container could not start: Cannot connect to the Docker daemon` | Docker Desktop or the WSL engine is not running | `docker info`, start Docker Desktop, check `DOCKER_HOST` |
| Test exits as skipped in CI | `TEST_REQUIRE_DB` is missing | Set it to `1` in the workflow |
| `no migration files in ...` | Walk-up from the package did not find `scripts/postgres` | Set `TEST_MIGRATIONS_DIR` |
| `cannot insert multiple commands into a prepared statement` | pgx prepared mode executing a multi-statement migration | Use golang-migrate or `default_query_exec_mode=simple_protocol` |
| `database "..." is not a test database` | go-testfixtures safety check against a reused database | Use a disposable `*_test` database or the explicit override |
| `duplicate key value violates unique constraint` while loading fixtures | `Reset` was not called, or two tests share fixture IDs | Call `Reset` in the environment helper; use per-scenario IDs |
| `insert or update on table "umkm_entries" violates foreign key` | Fixtures loaded out of order or missing workspace/import | Load workspaces, imports, entries, allocations in order |
| `bcrypt: hashedSecret too short` | Placeholder password hash used for a real login test | Seed a real hash with `SetPassword` |
| `context deadline exceeded` during startup | Image pull is slow or the wait strategy is wrong | `docker pull postgres:16-alpine`, increase startup timeout |
| Leaked containers after a failed run | `TestMain` did not call `testkit.Main`, or Ryuk is blocked | Add `TestMain`; inspect `docker ps -a --filter label=org.testcontainers=true` |
| A test passes alone and fails in the package | Shared database state or test order dependence | Reset before every test; never rely on another test's rows |

For a failing query, run the single test with `-v` and inspect the database from
another terminal using `docker ps`, the mapped port, and `psql`. Testcontainers
also exposes container logs through the Docker CLI; the container labels include
the test session, which makes cleanup safe.

## 6. Patterns that should not be copied

- **Skip-on-no-Docker as the only policy.** It is convenient locally but hides
  a broken CI runner. Keep the skip for developer machines and make
  `TEST_REQUIRE_DB=1` mandatory in CI.
- **Truncating every public table without a test-database guard.** The reference
  enables `DangerousSkipTestDatabaseCheck` for an external DSN and truncates
  everything. This backend should require a database name containing `test` and
  an explicit override before doing that.
- **A hard-coded sentinel table for migration detection.** Checking
  `public.users` is specific to go-chat. Use a real migration ledger, or at
  minimum a table this schema actually owns.
- **Executing migration files as one statement under pgx's default mode.** It
  fails on multi-statement DDL. Use the migration tool or simple protocol.
- **A shared mutable database with parallel tests.** go-chat has no
  `t.Parallel()` because `Reset` and fixtures mutate the whole schema. Do not add
  parallelism unless each test gets its own schema or database.
- **Mocking sqlc or pgx.Tx.** It produces tests that pass while the real
  transaction, locks, foreign keys, and constraints are broken. Use the real
  database for those and keep pure rules separate.
- **Router tests that only inspect a route table with nil handlers.** That proves
  registration, not handler behaviour. Add request-level tests for status,
  cookies, CSRF, validation, and the JSON envelope.
- **The private `pgtestutil` package and the nested `unitesting-example` module.**
  It is a useful reference for the `ServiceProxy` idea, but it is not a
  dependency that can be imported, it uses a private module path, and it has no
  per-test reset. Do not vendor or copy it.
- **Using `assert` for control flow.** The legacy example uses testify
  extensively. The target has no testify dependency and the existing tests use
  stdlib `t.Fatalf`/`t.Errorf`. Stay with stdlib unless the team explicitly adds
  an assertion library.
- **Test fixtures that require a real login hash or a real token.** Keep secrets
  and credentials out of YAML. Generate hashes and session token hashes in
  helpers.
- **Service constructors whose production behaviour depends on nil
  dependencies.** The reference sometimes builds services with nil config and
  relies on defaults. In this backend, wire an explicit test environment so a
  missing dependency fails at construction or in a clear test error.
- **Test data that depends on the legacy `pblg_workspaces.state` JSONB.** After
  the UMKM cutover, normalized tables are the source of truth. Only the backfill
  migration test should seed legacy JSONB.

## 7. First Docker-backed integration scenario

This is the smallest end-to-end scenario that proves register, login, import,
suggestions, allocation, undo, cancel, and export without turning the first test
into a full regression suite. Put it in
`backend/internal/echohttp/umkm_flow_test.go` (package `echohttp_test`). Use
`httptest.NewServer` with `NewEcho` so cookie handling is real, and a
`cookiejar` client. Every protected POST or PUT carries the `X-CSRF-Token`
header returned by the register or login response; register and login
themselves are public.

The scenario uses two workspaces: the owner created by register, and a second
fixture user in another workspace for the isolation check. The fixture for the
second user is in `internal/testdata/auth/two_workspaces.yaml`. The first
workspace is created through the API, so the test starts from a clean database.
Call `testkit.SetPassword` for the fixture user before logging in; a YAML
placeholder hash cannot satisfy bcrypt.

| Step | Request | Expected result | Database assertion |
|---|---|---|---|
| 1 | `POST /api/v1/auth/register` with owner name, email, 12+ character password, workspace name | 201, owner user, workspace, session cookie, CSRF token | One `pblg_workspaces`, one `pblg_users` with role `owner`, one `pblg_sessions` row whose `token_hash` is not the cookie plaintext |
| 2 | `POST /api/v1/auth/login` with a wrong password | 401 with the generic credentials message | No new session row |
| 3 | `POST /api/v1/auth/login` with the correct password | 200, new session, same user and workspace | Exactly two live sessions now |
| 4 | `GET /api/v1/auth/me` with the session cookie | 200, same user; no password hash in the body | None |
| 5 | `POST /api/v1/umkm/imports/preview` with a sale CSV | 200, parsed rows, count, total, checksum | Nothing written by preview |
| 6 | `POST /api/v1/umkm/imports` to commit the same sale CSV | 201, imported 2, skipped 0 | One `umkm_imports` and two sale `umkm_entries` |
| 7 | Repeat the same sale commit | 200 or 201 with `duplicate: true` and imported 0 | Still one import and two entries |
| 8 | Commit a receipt CSV with the same reference, amount, and channel, dated 1 day later | 201, imported 1 | One receipt entry |
| 9 | `GET /api/v1/umkm/suggestions` | 200, exactly one sale/receipt pair | No writes; import and entry counts unchanged |
| 10 | `POST /api/v1/umkm/allocations` with the suggestion IDs, full amount, fee 0, reason empty, and a 32-hex key | 201, allocation ID | One active allocation, one audit row, workspace version +1 |
| 11 | Repeat step 10 with the same key and payload | 200/201, same allocation ID | Still one allocation; no duplicate audit/version change |
| 12 | Repeat step 10 with the same key but a different amount | 409 | No new allocation |
| 13 | `GET /api/v1/umkm/suggestions` again | 200, empty | The active allocation excludes both entries |
| 14 | `POST /api/v1/umkm/allocations/{id}/undo` | 200 | `reversed_at` set, row not deleted; audit row added; version +1 |
| 15 | `GET /api/v1/umkm/suggestions` again | 200, the pair is suggested again | No writes |
| 16 | Allocate again with a new 32-hex key, then `POST /api/v1/umkm/imports/{id}/cancel` | 409 because an active allocation exists | Import and entries remain active |
| 17 | Undo the allocation, then cancel the import | 200 | Import and all entries have `canceled_at`; allocations remain |
| 18 | Repeat the cancel | 200, no error | No second change to timestamps |
| 19 | `GET /api/v1/umkm/export` | 200, CSV with UTF-8 BOM, active and canceled rows, no formula injection | No writes |
| 20 | Login as the second workspace user and request `/api/v1/umkm/state` and `/api/v1/umkm/export` | 200 with that user's own workspace only, never the first workspace's data | No cross-workspace rows returned |

Use these assertions to catch the important transaction bugs:

```go
// After a successful allocation:
var version int64
var audit int
err := env.sqlDB.QueryRow("select version from pblg_workspaces where id = $1", workspaceID).Scan(&version)
// compare to the version before the request
err = env.sqlDB.QueryRow("select count(*) from pblg_audit where workspace_id = $1 and action = $2", workspaceID, "allocation.create").Scan(&audit)
// audit must be 1

// After a failed import (conflicting source ID), nothing may be written:
var imports, entries int
env.sqlDB.QueryRow("select count(*) from umkm_imports where workspace_id = $1", workspaceID).Scan(&imports)
env.sqlDB.QueryRow("select count(*) from umkm_entries where workspace_id = $1", workspaceID).Scan(&entries)
// both counts must equal the pre-request values
```

The CSV fixtures are small and readable:

```csv
id,tanggal,referensi,nama,nominal,kanal,catatan
S-001,2026-10-20,INV-001,Penjualan Ana,150000,transfer,
S-002,2026-10-20,INV-002,Penjualan Budi,225000,qris,
```

The receipt file uses the same `referensi`, `nominal`, and `kanal` values, with
`tanggal` one day later. The suggestion rule is exactly what the prototype
specifies: matching lowercase reference, channel, and amount; receipt 0-3 days
after sale; exactly one candidate in each direction.

A minimal test body looks like this:

```go
func TestUMKMFlowIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("integration test: -short disables Docker")
    }

    env := testkit.New(t)
    env.Reset(t)
    env.Load(t, "../testdata/auth/two_workspaces.yaml")

    cfg := testConfig()
    srv := httptest.NewServer(echohttp.NewEcho(cfg,
        service.NewService(cfg, env.NewDB(t), nil),
        slog.New(slog.NewTextHandler(io.Discard, nil))))
    t.Cleanup(srv.Close)

    jar, _ := cookiejar.New(nil)
    client := &http.Client{Jar: jar}

    register := postJSON(t, client, srv.URL+"/api/v1/auth/register", map[string]any{
        "name": "Ana", "email": "ana@example.com",
        "password": "kata-sandi-panjang", "workspace": "Toko Ana",
    })
    // assert 201, session cookie, CSRF

    sale := postJSON(t, client, srv.URL+"/api/v1/umkm/imports/preview", map[string]any{
        "kind": "sale", "csv": string(readFile(t, "../testdata/csv/sale_valid.csv")),
    })
    // preview, then commit with the returned checksum
    // continue with receipt, suggestions, allocate, undo, cancel, export
}
```

Keep request helpers in the test file rather than in production packages:
`postJSON` sets `Content-Type: application/json` and `X-CSRF-Token`, decodes the
body into a typed struct, and fails with the status and body on unexpected
responses. That failure message is what makes an integration test debuggable.

Run this scenario first in isolation:

```bash
cd backend
TEST_REQUIRE_DB=1 go test -race -count=1 -run TestUMKMFlowIntegration -v ./internal/echohttp
```

Once it is stable, add the negative cases that the table above calls out:
duplicate import, conflicting source ID, invalid CSV, cross-workspace access,
and cancel while allocated. Those are the cases most likely to expose a missing
transaction or a missing workspace filter.
