# Verifikasi Perbaikan Bug EFA dan Dampak Backend

Tanggal: 30 Oktober 2026.

- Commit audit: `ca1a04a` (PR #13, “Fix EFA audit issues and redesign landing hero”)
- Author perubahan: **Wowieee4 / Evan William** (`evanwilliam303@gmail.com`)
- Merge oleh: Ahmad Kusumah (`ca1a04a`)
- `origin/main` saat audit: `cdf1c3e` (PR #15 observability)
- Frontend live: `https://efa.madgo.my.id/app`
- Backend live: `https://temukas.madgo.my.id`

## Ringkasan

Perbaikan di commit Evan **sudah benar dan sudah ada di `origin/main`**. Dari 9
bug yang sebelumnya tervalidasi, semuanya tertutup di kode; beberapa perbaikan
lain masih parsial. Build, vet, dan test pada snapshot commit lulus.

Masalah utama bukan pada kode, melainkan pada **deployment backend yang masih
lama**. Frontend live sudah versi `origin/main`, tetapi API `temukas` masih
berperilaku seperti sebelum PR #13. Rebuild API diperlukan; tidak perlu
migration, env baru, atau cherry-pick manual.

## Status bug yang sebelumnya tervalidasi

| Bug | Status | Bukti |
| --- | --- | --- |
| Halaman basi setelah simpan pasangan | **Fixed** | `app.js:1014-1016` menutup modal lalu `refresh(true)`; `app.js:378-386` menerapkan data dan render sebelum menunggu suggestions |
| `#login`/`#register` saat sudah login | **Fixed** | `app.js:638-640` mengalihkan ke `#overview`; terverifikasi live |
| Pratinjau impor menghitung duplikat | **Fixed** | `validation.js:6-20` dedupe klien; server `umkm_service.go:134-163`; terverifikasi live di UI |
| Impor duplikat menulis audit/versi | **Fixed di kode** | `umkm_service.go:185-197,226-228` no-op sebelum audit/version; **belum live** |
| Klik backdrop tidak menutup dialog | **Fixed** | `app.js:864-868`; terverifikasi live |
| Fokus awal dialog ke tombol Tutup | **Fixed** | `app.js:433` memfokuskan field pertama; terverifikasi live ke “Jenis transaksi” |
| Label “Menyimpan…” untuk semua aksi | **Fixed** | `app.js:916`; live menampilkan “Memproses…” dan “Memeriksa…” |
| Batas password byte vs karakter | **Fixed di kode** | `auth_service.go:262-264`: minimal 12 rune, maksimal 72 byte; pesan UI menyebut byte; **belum live** |
| Ekspor 401 menampilkan JSON mentah | **Fixed** | `app.js:605-628` memakai fetch, menangani 401 dan error jaringan |

## Status temuan high/medium

| Temuan | Status | Catatan |
| --- | --- | --- |
| Alokasi jatuh ke pasangan pertama | **Fixed** | `validation.js:21-27` menolak ID yang hilang; `app.js:982-987` memvalidasi ulang |
| Refresh membuang state bila suggestions gagal | **Fixed** | Data diterapkan lebih dulu; suggestions ditangani terpisah (`app.js:378-408`) |
| Snapshot state/suggestions tidak konsisten | **Fixed** | Kandidat difilter terhadap snapshot `data` yang sama (`app.js:389-396`) |
| Event SSE hilang saat refresh | **Fixed sebagian** | `pendingVersion` diantrikan (`app.js:751-761`), tetapi dibuang jika dialog terbuka (`:420-423`) |
| CSRF 403 setelah login ulang | **Fixed** | `api-client.js:29-32` + `app.js:88-96` menyegarkan CSRF lewat `/auth/me` |
| Login sukses tetapi state gagal menampilkan form login | **Fixed** | `app.js:935-940` memakai layar error workspace, bukan kembali ke form |
| Sesi habis menutup form dan membuang isian | **Parsial** | Ada peringatan (`app.js:100-107`), tetapi isian tetap tidak dipulihkan |
| Batas 500 baris hanya di server | **Belum** | Klien tetap hanya memeriksa 1 MB (`app.js:946`) |
| Batas nominal alokasi tidak dihitung ulang | **Fixed** | `app.js:496-518` memperbarui batas setiap pilihan berubah |
| Kuota 10.000 termasuk entri batal | **Parsial** | Dedupe sudah sebelum kuota (`umkm_service.go:222-230`), tetapi `CountUmkmEntries` menghitung entri batal |
| CSV non-UTF-8 rusak diam-diam | **Fixed** | `TextDecoder(fatal:true)` (`app.js:948-950`) dan validasi server |
| Impor semua-duplikat membuat batch kosong | **Fixed di kode** | `umkm_service.go:226-228`; **belum live** |
| Pagination/pencarian hanya klien; audit 120 | **Belum** | `app.js:207-222`, `umkm_service.go:43-46` |
| Kontras fokus permukaan gelap | **Parsial** | `.auth-umkm`, `.cash-account`, `#mobile-menu` diperbaiki; tombol tutup toast masih ~1,51:1 |

## Bukti verifikasi live

### Frontend (sudah benar)

- Membuka `/app#login` saat sesi aktif langsung dialihkan ke `#overview`.
- Dialog impor memfokuskan field “Jenis transaksi”, bukan tombol Tutup.
- Klik di luar panel menutup dialog (dengan konfirmasi jika ada isian).
- Tombol submit menampilkan “Memproses…” untuk login dan “Memeriksa…” untuk
  pratinjau CSV.
- Pratinjau CSV berisi `P101` (sudah ada) dan `S2` baru menampilkan
  **1 baris baru / Rp50.000 / 1 baris identik dilewati**. Ini dedupe klien,
  sehingga UI tetap benar meski server masih lama.
- Tombol lihat kata sandi sudah berupa checkbox dengan label yang dapat
  dibaca pembaca layar.
- Blob `app.js` live sama dengan `origin/main` (`d4648159...`). Selisih ukuran
  file sebelumnya hanya konversi CRLF, bukan versi berbeda.

### Backend live (masih lama)

Pengujian dengan akun demo menunjukkan perilaku sebelum PR #13:

- `POST /api/v1/umkm/imports/preview` untuk `P101` + `S2` mengembalikan
  `count=2`, `total=200000`, dan tidak mengirim `skipped`.
- Impor ulang file yang sama mengembalikan `201` dengan
  `{"imported":0,"skipped":5}`, lalu **menaikkan versi 8 → 9, menambah audit,
  dan membuat batch kosong**.

Artinya container API di VPS belum dibangun ulang dari `origin/main`.

## Dampak commit backend Evan

Commit `ca1a04a` mengubah tujuh file backend, tanpa migration baru:

- `internal/echohttp/umkm_handler.go` — preview memakai session dan workspace.
- `internal/models/umkm.go` — `PreviewResponse.Skipped` (aditif).
- `internal/service/umkm_service.go` — preview per workspace, dedupe, no-op
  untuk duplikat, kuota setelah dedupe.
- `internal/service/umkm_domain.go` — `filterImportRows` dipakai preview dan commit.
- `internal/service/auth_service.go` — minimal 12 rune, maksimal 72 byte UTF-8.
- `internal/echohttp/import_audit_test.go` — tes duplikat, no-op, preview.
- `internal/service/auth_domain_test.go` — tes batas Unicode.

Hasil audit:

- **Tidak breaking di wire format**: `skipped` aditif; status tetap 200/201.
- **Semantik berubah**: `count`/`total` preview kini hanya baris baru; preview
  dapat mengembalikan `409` untuk konflik source ID.
- **Tidak perlu migration**; query `ListUmkmEntriesByKindAndSourceIDs` sudah ada
  dan memakai index unik `(workspace_id, kind, source_id)`.
- **Transaksi aman**: `defer tx.Rollback` terpasang; jalur duplikat/no-op keluar
  sebelum commit sehingga tidak menulis batch, audit, atau version.
- **Isolasi workspace benar** di kode: query memakai `user.WorkspaceID`.
- **Tidak ada konflik dengan PR #15 observability**; jangan menyalin seluruh
  snapshot backend karena akan menimpa perubahan Dockerfile/go.mod observability.

Risiko yang perlu dicatat:

- Respons no-op seluruh-duplikat mengembalikan `{duplicate:true}` tanpa `id`.
- Password multibyte 4–11 karakter (atau 6 emoji) yang dulu lolos kini ditolak.
- Kuota 10.000 masih menghitung entri yang dibatalkan.
- Tes belum mencakup dua workspace dan commit campuran baris baru + duplikat.

## Temuan baru / perbaikan yang belum tuntas

1. **Versi tertunda hilang saat dialog terbuka** — `app.js:420-423` membuang
   `pendingVersion`; setelah dialog ditutup data baru menunggu polling.
2. **Pratinjau lama tetap bisa di-commit** jika dedupe klien melempar error
   setelah server 200 (`app.js:956-957`); hasil lama tidak dibersihkan.
3. **Preview tidak memperingatkan kuota**; tombol simpan tetap muncul, penolakan
   baru terjadi saat commit (`umkm_service.go:229`).
4. **Respons duplicate tanpa ID** menyulitkan klien yang ingin merujuk batch.
5. **Dokumentasi** `backend/docs/PBL_QUERY_INVENTORY.md:62` masih menyatakan
   preview tidak membaca database.

## Langkah sync yang disarankan

1. Di VPS, pastikan repo di `main` dan sudah sama dengan `origin/main`:

```bash
cd ~/rekonsiliasi-kas-umkm
git fetch origin
git switch main
git pull --ff-only
git log --oneline -1
```

2. Rebuild image API (wajib; `docker compose up -d` saja tidak cukup):

```bash
cd ~/rekonsiliasi-kas-umkm/backend
sudo docker compose up -d --build
```

3. Verifikasi setelah rebuild:

- Preview `P101` + `S2` harus mengembalikan `count=1`, `skipped=1`, `total=50000`.
- Impor duplikat harus `imported=0`, `skipped=5`, tanpa kenaikan versi, audit,
  atau batch baru.
- `GET /health/live` harus menunjukkan uptime baru.

4. Jangan mengubah `backend/compose.yaml` di VPS; file itu modifikasi lokal.
   Tidak ada perubahan compose pada PR #13/#15, jadi `git pull` aman.

## Artefak uji

- Batch kosong dari uji backend lama: `c392b68885d6215231e4c43d2544385d`,
  sudah dibatalkan. Tetap terlihat di Riwayat sebagai batch batal.
- Akun uji `qa-20261028-1@temukas.test` dan batch `INV-SSE-1` dari pengujian
  sebelumnya masih ada.
- Tidak ada entri keuangan baru yang ditambahkan oleh pengujian ini.

## Kesimpulan

Perbaikan Evan benar, sudah termerge, dan berisiko rendah. Yang salah adalah
deployment: frontend live sudah baru, backend live masih lama. Setelah rebuild
API, semantik preview dan impor akan cocok dengan frontend. Sebelum rilis,
putuskan apakah audit percobaan duplikat perlu dipertahankan dan tambahkan tes
isolasi dua workspace serta commit campuran.
