# migrations

Setiap perubahan schema memakai pasangan file bernomor enam digit:

```text
000001_create_users.up.sql
000001_create_users.down.sql
```

Buat dengan `make gen.pg.migration name=create_users`. Jangan mengubah migration
yang sudah pernah dijalankan di environment bersama. Folder ini menjadi schema
source untuk sqlc.

## Baseline saat ini

- `000001_pbl_baseline` memindahkan schema runtime dari prototipe PBL.
- Schema tersebut berisi tabel `pblg_*` dan menyimpan sebagian besar data bisnis
  di kolom JSONB `pblg_workspaces.state`.
- Penjelasan lengkap dan peringatan rollback ada di
  `docs/DATABASE_FROM_PBL.md`.
