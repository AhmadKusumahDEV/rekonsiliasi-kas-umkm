# Arsitektur Backend

Struktur ini mengikuti target `init-project.md`, dengan penyesuaian pada
kebiasaan project ini dan `go-chat`.

## Keputusan utama

| Area | Keputusan project ini |
|---|---|
| Dependency | Go module, tanpa folder `vendor` |
| HTTP | Echo v4 |
| Database pool | `pkg/database` dengan pgxpool |
| Query | sqlc, source di `scripts/postgres/queries` |
| Schema | migration di `scripts/postgres/migrations` |
| Generated query | `internal/repository/rekonsiliasi_repository_sqlc` |
| Config | Viper dari `configs/local.yaml` + environment |
| Waktu bisnis | `Asia/Jakarta` |
| Domain baru | satu vertical slice per domain |

`pkg/database` adalah package yang diimpor oleh `internal/service` dan `cmd/api`.
Ini berbeda dari blueprint yang tidak memiliki package pool database di `pkg`.
SQL domain tetap tidak ditaruh di `pkg`; SQL source tetap di `scripts/postgres`.

## Arah dependency

```text
cmd/api
  ├──> internal/config
  ├──> internal/service ──> internal/models
  │        ├──> pkg/database
  │        ├──> pkg/errors
  │        ├──> pkg/utils
  │        └──> internal/repository/rekonsiliasi_repository_sqlc
  └──> internal/echohttp ──> internal/service
                              internal/models
                              pkg/http/models
                              pkg/errors

pkg/* ──> stdlib dan third-party saja, tidak pernah internal/*
```

Aturan penting:

- Handler di `internal/echohttp` tidak mengimpor sqlc.
- Service tidak mengimpor Echo.
- Models tidak mengimpor transport atau generated row type.
- Generated code tidak mengimpor service atau handler.

## Composition root

`cmd/api/main.go` melakukan urutan berikut:

1. Membuat signal-aware context.
2. Memuat typed config dari Viper dan environment.
3. Membuat `pkg/database.DB` bila database diaktifkan.
4. Membuat HTTP client.
5. Merakit satu `service.Service`.
6. Menjalankan `echohttp.RunEchoHTTPService` dengan graceful shutdown.

Tidak ada pembuatan dependency di handler atau service constructor yang
tersembunyi. Entry point worker nantinya memakai urutan yang sama, lalu memanggil
consumer dari `internal/worker`.

## Layer HTTP

Route didaftarkan di `internal/echohttp/http_server.go`. Handler adalah closure
yang menerima `*service.Service` dan mengembalikan `echo.HandlerFunc`. Error dari
handler dikembalikan apa adanya; mapping status dan pesan dilakukan satu kali di
`echohttp_error_handler.go`.

Endpoint publik saat ini:

- `GET /health/live`
- `GET /health/ready`

Tambahkan group authenticated setelah middleware auth tersedia:

```go
v1 := e.Group("/v1", authMiddleware)
v1.GET("/contoh", listContoh(s))
```

## Layer service dan transaksi

Service memegang `*database.DB`, HTTP client, config, dan dependency lain yang
dirakit di `cmd/api`. Unit of work ditulis sebagai method service:

```go
func (s *Service) CreateContoh(ctx context.Context, req models.CreateContohRequest) (resp models.Contoh, err error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return resp, pkgErr.ErrFailInitializeTx
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := sqlc.New(tx)
	if err = s.insertContoh(ctx, queries, req); err != nil {
		return resp, err
	}
	if err = tx.Commit(ctx); err != nil {
		return resp, pkgErr.ErrFailCommitTx
	}
	return resp, nil
}
```

Gunakan named return `err` agar deferred rollback melihat error terakhir. Jika
ada panggilan HTTP/storage eksternal, lakukan sebelum transaksi atau setelah
commit; jangan menahan transaksi database selama network call.

## Package yang belum berisi implementasi

`internal/apiclient`, `internal/worker`, `cmd/worker`, dan `internal/testdata`
masih berupa dokumentasi sampai domain pertama membutuhkannya. Ini disengaja;
jangan menambah file Go kosong atau service palsu hanya agar folder terlihat
penuh.
