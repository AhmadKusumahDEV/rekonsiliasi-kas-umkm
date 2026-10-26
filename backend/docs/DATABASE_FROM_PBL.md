# Database PBL ke Migration Backend

Dokumen ini menjelaskan sumber schema yang dipakai pada migration baseline.
Bagian yang penting: **PBL tidak menyimpan schema di folder migration atau file
SQL terpisah**. Schema dibuat saat aplikasi start dari konstanta `schema` di:

```text
Tiga-Prototipe-PBL/internal/platform/store.go
```

File `docs/flow.txt` menjelaskan alur aplikasi, bukan daftar tabel. Karena itu
flow terlihat jauh lebih besar daripada schema database-nya.

## Schema yang dipindahkan

Migration `000001_pbl_baseline.up.sql` membuat enam tabel dan dua index:

```mermaid
erDiagram
    pblg_workspaces ||--o{ pblg_users : "memiliki akun"
    pblg_workspaces ||--o{ pblg_jobs : "memiliki antrean"
    pblg_workspaces ||--o{ pblg_audit : "memiliki audit"
    pblg_workspaces ||--o{ pblg_photos : "memiliki foto"
    pblg_users ||--o{ pblg_sessions : "memiliki sesi"
    pblg_users ||--o{ pblg_photos : "mengunggah"

    pblg_workspaces {
        text id PK
        text app
        text name
        text code UK
        bigint version
        jsonb state
        timestamptz created_at
    }

    pblg_users {
        text id PK
        text workspace_id FK
        text app
        text name
        text email
        text password_hash
        text role
    }

    pblg_sessions {
        text token_hash PK
        text user_id FK
        text csrf
        timestamptz expires_at
    }

    pblg_jobs {
        text id PK
        text workspace_id FK
        text app
        text kind
        jsonb payload
        text dedupe
        text status
        integer attempts
        timestamptz run_at
        timestamptz deadline
        timestamptz lease_until
        text locked_by
        text last_error
        timestamptz created_at
    }

    pblg_audit {
        bigserial id PK
        text workspace_id FK
        text actor
        text action
        text detail
        timestamptz created_at
    }

    pblg_photos {
        text id PK
        text workspace_id FK
        text owner_id FK
        text mime
        bytea body
        timestamptz created_at
    }
```

## Di mana data bisnis disimpan?

PBL memakai pendekatan **workspace state JSONB**, bukan satu tabel per fitur.
`pblg_workspaces.state` berisi array seperti:

- `entries`, `imports`, dan `allocations` untuk Temu Kas.
- `assignments` dan `submissions` untuk Tilik.
- `tickets` untuk Ruang.

Tabel lain menyimpan identitas, sesi, job queue, audit, dan foto. Jadi fitur
baru belum tentu perlu tabel baru pada schema PBL; perubahan state cukup lewat
JSONB. Ini juga alasan flow aplikasi tidak bisa dibaca sebagai relasi tabel
langsung.

## Yang ikut dan tidak ikut dimigrasikan

Ikut dimigrasikan:

- Definisi enam tabel dan dua index.
- Foreign key dan aturan `ON DELETE CASCADE` dari schema asli.

Tidak ikut dimigrasikan:

- Data demo, akun demo, dan hash password. Itu dibuat oleh kode seed PBL.
- State workspace yang sudah ada. Data lama tetap berada di database lama.
- Worker, auth, scheduler, dan logic aplikasi PBL.

Migration hanya schema. Jika ingin memindahkan data lama, gunakan `pg_dump`
untuk data atau buat seed/import terpisah setelah schema selesai.

## Cara menerapkan

Untuk database kosong, jalankan migration biasa:

```powershell
cd backend
make pg.migrate.up DATABASE_URL="postgres://postgres:postgres@localhost:5432/rekonsiliasi_kas_umkm?sslmode=disable"
```

Untuk database yang **sudah dibuat PBL**, migration memakai `IF NOT EXISTS`.
Artinya tabel yang sudah ada tidak dibuat ulang dan data tidak dihapus. Namun
migration tidak memperbaiki perbedaan kolom bila schema lama sudah berbeda.

Down migration menghapus seluruh tabel `pblg_*` beserta datanya. Jangan jalankan
`pg.migrate.down` pada database berisi data tanpa backup.

## Arah pengembangan berikutnya

Baseline ini menjaga kompatibilitas dengan database prototipe. Untuk backend
baru, keputusan desain berikutnya adalah:

1. Tetap memakai `pblg_workspaces.state` JSONB seperti prototipe, atau
2. menormalisasi entitas seperti entry, allocation, submission, dan ticket ke
   tabel sendiri lewat migration baru.

Jangan mengubah migration `000001` setelah migration ini pernah dijalankan.
Tambahkan migration baru untuk perubahan schema berikutnya.
