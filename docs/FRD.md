# Functional Requirements Document — Temu Kas

Kontrak aktif UI Temu Kas dan backend tim. Dokumen lengkap sebelumnya tetap ada di [FRD prototipe](reference/FRD-prototype.md).

## Arsitektur

frontend/assets/app.js mengelola form, navigasi hash dan API; design.js menyusun UI; motion.js/css mengatur animasi/reduced motion. Handler internal/echohttp memanggil service. Aturan dan transaksi ada di internal/service; SQL berasal dari scripts/postgres/queries dan generated sqlc. Saldo diperiksa ulang di server.

## Akun dan keamanan

| ID | Perilaku |
| --- | --- |
| AUTH-01 | Daftar membuat akun owner dan ruang kerja baru |
| AUTH-02 | Login memakai email/password, bukan pilihan peran |
| AUTH-03 | Password 12–72 karakter, hash bcrypt; token sesi disimpan sebagai hash |
| AUTH-04 | Cookie rku_session: HttpOnly, SameSite=Strict, Secure melalui TLS/proxy HTTPS terpercaya |
| AUTH-05 | Mutasi terautentikasi memerlukan X-CSRF-Token dari auth/me |
| AUTH-06 | Logout menghapus sesi; ganti password memerlukan password lama dan mengakhiri seluruh sesi |
| AUTH-07 | Register/login berbagi limiter per IP: burst 25, refill 25 token/600 detik, 429 saat habis |
| AUTH-08 | Semua objek dibatasi workspace sesi; 401 sesi tidak valid, 403 CSRF tidak cocok |

Limiter berada di memori per proses. Backend hanya boleh mempercayai forwarded header dari proxy yang dikelola.

## Impor

Header tujuh kolom berurutan:

```csv
id,tanggal,referensi,nama,nominal,kanal,catatan
```

| ID | Aturan |
| --- | --- |
| IMP-01 | Kind sale/receipt, UTF-8, BOM diterima |
| IMP-02 | Maksimal 1 MiB, 500 baris data/file, 10.000 entri/workspace |
| IMP-03 | Header dinormalisasi setelah trim; field maksimal 400 karakter tanpa NUL |
| IMP-04 | Source ID wajib, maksimal 80 karakter, unik dalam file; nama dan kanal wajib |
| IMP-05 | Tanggal YYYY-MM-DD valid; Rupiah integer positif sampai 1.000.000.000.000 |
| IMP-06 | Preview berisi rows, total, checksum; tidak menyimpan |
| IMP-07 | Commit validasi ulang dan all-or-nothing dalam transaksi |
| IMP-08 | Checksum file aktif identik menghasilkan duplicate tanpa data ganda |
| IMP-09 | Source ID dan isi identik dilewati; isi berbeda atau ID dibatalkan menghasilkan konflik |
| IMP-10 | File yang batch-nya dibatalkan tidak dipulihkan dengan impor ulang |
| IMP-11 | Ganti file/jenis menghapus preview lama; kontrol dikunci saat commit |
| IMP-12 | Batalkan batch hanya setelah alokasi aktif dibalikkan; data tetap tersimpan dengan penanda |

## Usulan dan keputusan

| ID | Aturan |
| --- | --- |
| REC-01 | Referensi terisi dan sama tanpa membedakan huruf, kanal/nominal sama, penerimaan 0–3 hari sesudah penjualan |
| REC-02 | Kandidat unik di kedua sisi; kandidat ambigu tidak dipilih otomatis |
| REC-03 | Hanya konfirmasi pemilik menyimpan keputusan |
| REC-04 | Amount > 0, fee >= 0; amount <= sisa penerimaan; amount + fee <= sisa penjualan |
| REC-05 | Fee > 0 memerlukan alasan minimal 5 karakter; alasan maksimal 400 |
| REC-06 | Beberapa alokasi mendukung pembayaran sebagian dan penerimaan gabungan |
| REC-07 | Key 32 hex untuk idempotensi; ulang identik tidak menduplikasi, isi berbeda ditolak |
| REC-08 | Lock workspace dan transaksi mencegah pemakaian saldo berlebih saat dua penulis bersamaan |
| REC-09 | Undo menandai reversed dan mengembalikan saldo tanpa menghapus riwayat |

Sisa penjualan = nominal − amount aktif − fee aktif. Sisa penerimaan = nominal − amount aktif. Fee tidak menambah penerimaan. Penerimaan bukan laba.

## Endpoint

Prefix **/api/v1** untuk semua path di tabel.

| Method | Path | Hasil |
| --- | --- | --- |
| GET | /config | kind, brand, tagline, roles |
| POST | /auth/register | 201 user + csrf dan cookie |
| POST | /auth/login | 200 user + csrf dan cookie |
| GET | /auth/me | user + csrf |
| POST | /auth/logout | Akhiri sesi |
| PUT | /auth/password | Ganti password dan akhiri seluruh sesi |
| GET | /umkm/state | workspace, state, users, events |
| GET | /umkm/suggestions | Kandidat unik |
| POST | /umkm/imports/preview | Validasi tanpa tulis |
| POST | /umkm/imports | Commit batch |
| POST | /umkm/imports/:id/cancel | Pembatalan batch |
| POST | /umkm/allocations | Simpan keputusan |
| POST | /umkm/allocations/:id/undo | Balikkan keputusan |
| GET | /umkm/export | Rekap CSV |
| GET | /umkm/events | SSE workspace; state version, resync, server_close |

Health: GET /health/live dan /health/ready. Database nonaktif dilaporkan disabled; aplikasi yang memerlukan data harus mengaktifkannya.

Error JSON: {code,status,message,request_id}. 400 input; 401 sesi/kredensial; 403 CSRF; 404 objek; 409 konflik saldo/ID/status; 429 limiter; 503 dependency; 500 tak terduga. UI menampilkan message, bukan detail SQL. Endpoint lama seperti /api/state tidak dipakai sebagai alias.

## Penyimpanan

| Tabel | Isi |
| --- | --- |
| pblg_workspaces | Usaha dan version mutasi |
| pblg_users | Pemilik dan hash password |
| pblg_sessions | Hash token, CSRF, expiry |
| pblg_audit | Aktor, aksi, detail dan waktu |
| umkm_imports | Batch, checksum, jenis, waktu cancel |
| umkm_entries | Penjualan/penerimaan, source ID |
| umkm_allocations | Pasangan, nominal, biaya, alasan, waktu reversal |

000001 adalah baseline historis; 000002 normalisasi; 000003 backfill idempotent. Tabel jobs/photos dan state JSON historis tidak dihapus. Endpoint Grader/Facility tidak didaftarkan. Integrasi UI tidak menambah migration atau mengubah generated sqlc.

## Ketahanan UI

- Ringkasan memakai semua data aktif. Tabel menyediakan pencarian, status, dan 20/50/100 baris.
- Form dinonaktifkan saat submit; commit tidak bisa ditutup selama menulis.
- Isian berubah meminta konfirmasi penutupan. Kegagalan mempertahankan input.
- Timeout client 32 detik bukan bukti transaksi gagal di server; refresh state sebelum mengulang hasil yang belum jelas.
- SSE state membawa version workspace; UI memuat ulang melalui REST. Saat koneksi dibuka/resync, state diperiksa kembali. Polling verifikasi 60 detik ketika stream sehat, 15 detik saat fallback, retry 30 detik jika gagal; hanya saat tab terlihat. Form aktif tidak ditimpa; tombol refresh disediakan.
- CSV ekspor menetralkan formula spreadsheet; riwayat mempertahankan pembatalan.
- Sesi kedaluwarsa kembali ke login; unknown hash menampilkan tautan kembali.
- Navigasi mobile, skip link, dialog native, dan reduced motion dipertahankan.

## Rujukan operasional

[Demo](DEMO.md), [API](INTEGRATION.md), [hosting](HOSTING.md), [tes](VERIFICATION.md). SSE berasal dari main tim terbaru dan terhubung ke EventSource frontend. Tidak ada perubahan Dockerfile, schema, atau kontrak API versi tim.
