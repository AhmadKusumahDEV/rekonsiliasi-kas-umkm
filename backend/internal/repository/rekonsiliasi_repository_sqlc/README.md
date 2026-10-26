# Generated sqlc package

Folder ini adalah output `sqlc generate` berdasarkan konfigurasi
`scripts/postgres/sqlc.yaml`. Jangan mengedit file generated di sini.

Schema berasal dari `scripts/postgres/migrations`, query dari
`scripts/postgres/queries`. Setelah migration dan query pertama dibuat, jalankan:

```bash
cd backend
make gen.pg.models
```

Service memakai query generated di dalam transaksi dengan `sqlc.New(tx)`.
Handler tidak boleh mengimpor package ini.
