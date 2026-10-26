# pkg/database

Package ini hanya mengurus koneksi PostgreSQL berbasis `pgxpool`: membuka pool,
validasi konfigurasi, ping/health check, menutup pool, dan menyediakan
`BeginTx`. Paket ini tidak berisi aturan bisnis, query domain, atau commit
transaksi.

Transaksi tetap dibuka di `internal/service`. Service memakai pool dari paket
ini, lalu meneruskan `pgx.Tx` ke query sqlc hasil generate.

```go
tx, err := svc.db.BeginTx(ctx, pgx.TxOptions{})
if err != nil {
	return err
}
defer func() {
	if err != nil {
		_ = tx.Rollback(ctx)
	}
}()
```

SQL source dan konfigurasi sqlc berada di `scripts/postgres`, bukan di paket
ini. Hasil generate masuk ke
`internal/repository/rekonsiliasi_repository_sqlc`.
