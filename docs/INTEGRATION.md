# Integrasi Temu Kas

## Browser sampai database

1. Browser mengambil index.html dan assets.
2. app.js mengambil /api/v1/config dan /api/v1/auth/me.
   Pada EFA live, `api-config.js` menambahkan origin `temukas.madgo.my.id`;
   seluruh fetch mengirim `credentials: include`, SSE memakai withCredentials.
   Lokal tetap memakai proxy dev.mjs. [Detail deployment](VERCEL_TEAM_API.md).
3. Login/register menghasilkan cookie HttpOnly dan csrf.
4. State/suggestions menggunakan identitas sesi; browser tidak menentukan hak akses workspace.
5. Mutasi membawa X-CSRF-Token. Handler memvalidasi input lalu memanggil service.
6. Service memeriksa ruang kerja, lock, saldo, query sqlc, audit dan version dalam satu transaksi.
7. UI mengambil ulang data. Perhitungan tampilan tidak menggantikan validasi server.

## Pemetaan prototipe ke API tim

| Lama | Baru |
| --- | --- |
| /api/config | GET /api/v1/config |
| /api/register, /api/login | POST /api/v1/auth/register, /auth/login |
| /api/me, /api/logout | GET /api/v1/auth/me, POST /auth/logout |
| POST /api/password | PUT /api/v1/auth/password |
| /api/state, /api/suggestions | GET /api/v1/umkm/state, /umkm/suggestions |
| /api/imports/preview, /api/imports | POST /api/v1/umkm/imports/preview, /umkm/imports |
| /api/imports/:id/cancel | POST /api/v1/umkm/imports/:id/cancel |
| /api/allocations, /api/allocations/:id/undo | POST /api/v1/umkm/allocations, /umkm/allocations/:id/undo |
| /api/export | GET /api/v1/umkm/export |
| /api/events | GET /api/v1/umkm/events; EventSource dengan cookie sesi |

Cookie tim: rku_session. Error tim memakai field message. Frontend tidak membaca database dan tidak membawa credential server.

Preview impor kini membaca transaksi workspace untuk melewati baris identik.
Respons menambahkan `skipped`; `rows`, `count`, dan `total` hanya menghitung
baris baru. Commit tetap memvalidasi ulang dalam transaksi. Impor tanpa
baris baru tidak menulis audit/batch/version atau mengirim event baru.
Frontend memfilter preview juga untuk kompatibilitas selama server tim
belum menjalankan versi baru. Lihat [hasil audit](audit/README.md).

## Yang dipertahankan

Dari Evan: tampilan, foto, branding, animasi, form, navigasi dan interaksi kas. Dari tim: Echo/service/sqlc, konfigurasi, normalized PostgreSQL, migration, CI, dan API /api/v1.

Tambahan: cookie Secure melalui proxy HTTPS, CORS CSRF header, limiter auth, seed demo manual, proxy lokal, tes, serta panduan domain yang sama.

Tidak dibawa: UI/aset Grader dan Facility, monolit tiga prototipe, .env, cache, binary, dan konfigurasi Vercel lama. Dokumen awal disimpan utuh di docs/reference. Placeholder arsitektur dan migration tim tetap utuh.

Main 6cc2d6e sudah menggabungkan branch SSE tim. Integrasi ini mempertahankannya, menyambungkan EventSource, memeriksa snapshot saat reconnect, dan memakai polling sebagai fallback. Hasil merge check berlaku pada revision yang diperiksa; update berikutnya perlu diperiksa lagi.
