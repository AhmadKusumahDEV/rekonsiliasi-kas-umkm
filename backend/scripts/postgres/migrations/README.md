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
- `000002_umkm_normalized` menambahkan tabel domain `umkm_imports`,
  `umkm_entries`, dan `umkm_allocations` beserta constraint dan index.
- `000003_backfill_umkm_state` memindahkan data UMKM lama dari JSONB ke tabel
  ternormalisasi. Timestamp cancel/reverse lama memakai waktu backfill karena
  prototipe hanya menyimpan boolean.
- Penjelasan lengkap dan peringatan rollback ada di
  `docs/DATABASE_FROM_PBL.md`.

Jalankan `make pg.migrate.up` satu kali untuk database kosong; golang-migrate
akan menerapkan 000001, 000002, dan 000003 berurutan. Jangan menjalankan down
pada database berisi data tanpa backup.
