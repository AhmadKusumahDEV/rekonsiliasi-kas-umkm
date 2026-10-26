# SQLC dan Database Guide

## Sumber kebenaran

- Schema: `scripts/postgres/migrations/*.up.sql` dan `*.down.sql`.
- Query: `scripts/postgres/queries/*.sql`.
- Konfigurasi generate: `scripts/postgres/sqlc.yaml`.
- Hasil generate: `internal/repository/rekonsiliasi_repository_sqlc/`.

Schema dan query yang sudah ada:

- `000001_pbl_baseline`: tabel runtime `pblg_*` dari prototipe.
- `000002_umkm_normalized`: tabel `umkm_imports`, `umkm_entries`, dan
  `umkm_allocations` beserta constraint/index.
- `000003_backfill_umkm_state`: pemindahan data UMKM lama dari JSONB state.
- `queries/auth.sql`: auth, sesi, workspace, user, audit, dan workspace lock.
- `queries/umkm.sql`: query baca/tulis domain rekonsiliasi kas.

Jangan membuat query sqlc langsung di package generated. Jangan mengedit file
`*.sql.go`, `db.go`, atau `models.go` hasil generate.

## Instalasi tool

```powershell
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1
go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

Pastikan `$(go env GOPATH)\bin` ada di `PATH`.

## Alur migration

```powershell
make gen.pg.migration name=create_users
```

Isi file up dengan perubahan schema, lalu down dengan rollback yang benar.
Jalankan migration memakai DSN yang sesuai:

```powershell
make pg.migrate.up DATABASE_URL="postgres://postgres:postgres@localhost:5432/rekonsiliasi_kas_umkm?sslmode=disable"
```

## Menulis query

```sql
-- name: GetUserByID :one
SELECT id, name, created_at
FROM users
WHERE id = @id;
```

Anotasi yang umum: `:one`, `:many`, `:exec`, dan `:execrows`. Gunakan named
parameter `@snake_case` dan kolom eksplisit. Untuk parameter NULL, periksa tipe
nullable yang dihasilkan sqlc sebelum memakai `Valid: true`.

## Generate

```powershell
make gen.pg.models
```

Jika tool Windows bermasalah, generate lewat WSL:

```powershell
wsl -d Ubuntu -- bash -lc 'cd /mnt/d/appwebsite/rekonsiliasi-kas-umkm/backend && sqlc generate -f scripts/postgres/sqlc.yaml'
```

Periksa signature `...Params`, `...Row`, atau struct tabel yang muncul. Sesuaikan
pemanggilan service, lalu commit file generated bersama migration dan query.
Project ini tidak memakai `-mod=vendor`; cukup `go test ./...` dan
`go build ./...`.

Kolom `date` dan `timestamptz` dipetakan sqlc ke `pgtype.Date` dan
`pgtype.Timestamptz`. Periksa `Valid` sebelum memakai nilai nullable.

Migration backfill `000003` memakai timestamp saat backfill untuk status
canceled/reversed bila data JSONB lama hanya menyimpan boolean. Jangan mengklaim
waktu pembatalan asli untuk baris hasil backfill.

## Memakai query di service

Read sederhana dapat memakai pool:

```go
queries := sqlc.New(s.db.Pool())
row, err := queries.GetUserByID(ctx, id)
```

Write yang terdiri dari beberapa langkah harus memakai transaksi di service:

```go
tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
if err != nil {
	return pkgErr.ErrFailInitializeTx
}
defer func() {
	if err != nil {
		_ = tx.Rollback(ctx)
	}
}()

queries := sqlc.New(tx)
// panggil query tulis dengan queries yang sama
if err = tx.Commit(ctx); err != nil {
	return pkgErr.ErrFailCommitTx
}
```
